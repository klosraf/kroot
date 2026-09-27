// Package cli is kroot's command framework: the registry that names every
// subcommand, the help text that describes them, and the suggestions offered
// when a caller mistypes one.
//
// It knows nothing about the process. No command here reads os.Args, writes to
// os.Stdout or calls os.Exit; those arrive through a Registry and an Env. That
// is what keeps the framework testable without spawning a binary, and it keeps
// exit-code policy in package main, where the documented contract lives.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ErrUsage marks a failure caused by how the program was invoked rather than by
// what it was asked to do. It is the sentinel behind exit code 2; package main
// maps it, and nothing in this package decides process codes.
var ErrUsage = errors.New("usage error")

// ErrUnknownCommand is wrapped by Registry.Error alongside ErrUsage, so a caller
// can ask "which command?" as well as "was my invocation wrong?" from one chain.
var ErrUnknownCommand = errors.New("unknown command")

// maxSuggestionDistance is the edit distance at which a typo is still worth
// naming. Two covers one substitution, one insertion, one deletion or one
// transposition of adjacent characters, which is most of what typing produces.
const maxSuggestionDistance = 2

// maxSuggestions bounds the list so a bad guess cannot bury the actual error.
const maxSuggestions = 3

// Env carries everything a command is allowed to touch. Passing writers rather
// than reaching for the process streams is what makes a command's output
// assertable in a test.
type Env struct {
	// Stdout carries what the caller asked for.
	Stdout io.Writer
	// Stderr carries diagnostics.
	Stderr io.Writer
	// Args holds the operands after the command name, unparsed. A command that
	// accepts flags parses its own FlagSet from these.
	Args []string
}

// RunFunc executes a command. The context is cancelled when the process is
// shutting down, so a command that blocks must select on it.
type RunFunc func(ctx context.Context, env Env) error

// Command describes one subcommand.
type Command struct {
	// Name is the word the caller types. Lower-case letters, digits and
	// single dashes only, starting with a letter.
	Name string
	// Summary is one line, used in the command list and in shell completion.
	// It is not a sentence: no trailing period.
	Summary string
	// Usage is the invocation line shown by `help <name>`, for example
	// "kroot rules [glob...]". It is rendered verbatim, so it must name the
	// binary as the caller invokes it.
	Usage string
	// Long is the paragraph `help <name>` and the manual print. When empty,
	// Summary is used instead.
	Long string
	// Run is the behaviour. It must not be nil.
	Run RunFunc
	// Operands declares what the command accepts in its first operand position,
	// so a shell can offer it there instead of declining. It is completion data,
	// not validation: dispatch never reads it, so it must describe the
	// invocation Usage already shows rather than enforce it. The zero value
	// means the command takes no operand, and the generators decline.
	Operands Operands
}

// Operands is what a command accepts in its first operand position, for shell
// completion only.
//
// Exactly one source per position: either Values, a closed vocabulary such as the
// supported shells, or CommandNames, the registry's own names — which is what
// `help` and `man` take. Declaring both is a programmer error the registry
// rejects; declaring neither is a command that takes nothing.
//
// The first operand is the only one modelled, because it is the only one any
// current command accepts. A command that grows a second operand declares its
// first here and its second in the same way, rather than the generators guessing
// positions they do not know.
type Operands struct {
	// Values is a closed vocabulary this position accepts, e.g. Shells.
	Values []string

	// CommandNames asks for the registry's command names instead of Values.
	CommandNames bool
}

// Registry is a validated set of commands. Registration order is irrelevant:
// every rendered view sorts by name, so output does not depend on the order
// commands were declared in.
type Registry struct {
	byName map[string]Command
}

// New validates the command set and returns it as a registry.
//
// Validation is deliberately strict and happens once, at construction: a
// duplicate or malformed name is a programmer error that would otherwise
// surface as a command that silently shadows another.
func New(commands ...Command) (*Registry, error) {
	r := &Registry{byName: make(map[string]Command, len(commands))}

	for _, c := range commands {
		switch {
		case !validName(c.Name):
			return nil, fmt.Errorf("command name %q is invalid: want lower-case letters, digits and dashes, starting with a letter", c.Name)
		case c.Summary == "":
			return nil, fmt.Errorf("command %q has no summary: the command list and completion both need one", c.Name)
		case c.Run == nil:
			return nil, fmt.Errorf("command %q has no Run function", c.Name)
		case c.Operands.CommandNames && len(c.Operands.Values) > 0:
			// Caught here rather than in a generator: by the time a shell script
			// exists the contradiction is already baked into a file someone
			// installed, and a shell is a poor place to report a programmer error.
			return nil, fmt.Errorf("command %q declares both an operand vocabulary and the command names: the position accepts one or the other", c.Name)
		}
		if _, exists := r.byName[c.Name]; exists {
			return nil, fmt.Errorf("command %q is registered twice", c.Name)
		}
		r.byName[c.Name] = c
	}

	return r, nil
}

// Lookup returns the command registered under name.
func (r *Registry) Lookup(name string) (Command, bool) {
	c, ok := r.byName[name]
	return c, ok
}

// Names returns every command name, sorted, so output and completion scripts
// are deterministic.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.byName))
	for name := range r.byName {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Commands returns every command sorted by name.
func (r *Registry) Commands() []Command {
	names := r.Names()
	commands := make([]Command, 0, len(names))
	for _, name := range names {
		commands = append(commands, r.byName[name])
	}
	return commands
}

// Suggestions returns the commands close enough to name to be worth proposing,
// best match first and bounded by maxSuggestions. A tie is broken by name so the
// answer is deterministic.
func (r *Registry) Suggestions(name string) []string {
	return suggestWithin(r.Names(), name, maxSuggestionDistance)
}

// suggestWithin returns the entries of known close enough to name to be worth
// proposing, best match first and bounded by maxSuggestions, with a tie broken
// by name so the answer is deterministic.
//
// It is a free function over a vocabulary rather than a Registry method because
// `completion` has to propose a shell name with the same machinery, and a second
// guesser would be a second set of bugs. The bound is a parameter because it
// cannot be one constant for every vocabulary: two edits against a four-letter
// word is half the word, so a bound tuned for command names turns a rejected
// value into a list of everything that vaguely resembles it.
func suggestWithin(known []string, name string, max int) []string {
	type candidate struct {
		name     string
		distance int
	}

	var candidates []candidate
	for _, k := range known {
		d := editDistance(name, k, max)
		if d <= max {
			candidates = append(candidates, candidate{name: k, distance: d})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].distance != candidates[j].distance {
			return candidates[i].distance < candidates[j].distance
		}
		return candidates[i].name < candidates[j].name
	})

	if len(candidates) > maxSuggestions {
		candidates = candidates[:maxSuggestions]
	}

	suggestions := make([]string, 0, len(candidates))
	for _, c := range candidates {
		suggestions = append(suggestions, c.name)
	}
	return suggestions
}

// Error reports an unrecognised command. It wraps ErrUsage and
// ErrUnknownCommand, and appends the closest matches when there are any, so the
// fix does not require reading the manual.
func (r *Registry) Error(name string) error {
	err := fmt.Errorf("%w: %w: %q", ErrUsage, ErrUnknownCommand, name)

	suggestions := r.Suggestions(name)
	if len(suggestions) == 0 {
		return err
	}
	return fmt.Errorf("%w; did you mean %s?", err, quoted(suggestions))
}

// quoted renders a list for a diagnostic: `"a"`, or `one of "a", "b"`.
func quoted(values []string) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, fmt.Sprintf("%q", v))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "one of " + strings.Join(parts, ", ")
}

// validName reports whether name is usable as a command word. The rule matches
// the CLI convention in docs/enterprise/coding-standards.md, and it is checked
// so that a command cannot end up unreachable by typing.
func validName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		case r == '-' && i > 0:
		default:
			return false
		}
	}
	return true
}

// editDistance is the Levenshtein distance between a and b, counted in runes so
// a non-ASCII typo is measured in characters rather than bytes.
//
// A pair further apart than max is reported as max+1 rather than its true
// distance: the caller only ever compares against max, and the exact number
// carries no meaning once the answer is "too far".
//
// It also exits early when the length difference alone exceeds max, which is the
// common case: a caller who typed three characters did not mean a twelve-letter
// command.
func editDistance(a, b string, max int) int {
	ar, br := []rune(a), []rune(b)

	if len(ar)-len(br) > max || len(br)-len(ar) > max {
		return max + 1
	}

	previous := make([]int, len(br)+1)
	current := make([]int, len(br)+1)
	for j := range previous {
		previous[j] = j
	}

	for i := 1; i <= len(ar); i++ {
		current[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			current[j] = min(
				previous[j]+1,      // deletion
				current[j-1]+1,     // insertion
				previous[j-1]+cost, // substitution
			)
		}
		previous, current = current, previous
	}

	return previous[len(br)]
}
