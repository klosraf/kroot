package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Shells is the accepted vocabulary for `kroot completion <shell>`, in the order
// the usage line and the rejection message present it.
//
// Like KROOT_LOG_LEVEL it is one definition, so the three places that name the
// supported shells cannot drift apart, and an unknown shell is rejected rather
// than guessed at. Adding a shell is one entry here plus one generator.
var Shells = []string{"bash", "fish", "zsh"}

// Entry is one completable command: the word the caller types, and the one-line
// description a shell able to show descriptions puts beside it.
//
// A Summary reaches a shell script as data rather than as prose, so it is
// escaped for the target shell before it is written. A Name is quoted too, even
// though New has already constrained it to lower-case letters, digits and
// dashes: the guarantee belongs to the registry, and a script that stays valid
// only while that guarantee holds is a trap for whoever relaxes it.
type Entry struct {
	Name     string
	Summary  string
	Operands Operands
}

// WriteCompletion writes the completion script for shell to w.
//
// The command list is baked into the script rather than resolved by calling
// kroot back at completion time. That keeps the result a static file a caller
// can read, diff and install once, and it keeps the generator free of any
// hidden protocol between the binary and the shell. The cost is that the script
// goes stale when kroot gains a command, which is why every generated header
// says to regenerate rather than edit.
//
// Entries are sorted by name here rather than trusted from the caller. The
// script is a file people install and regenerate, so the same command set has
// to produce the same bytes every time or each regeneration is a noisy diff. A
// copy is sorted so the caller's slice is left as it was found.
func WriteCompletion(w io.Writer, shell, binary string, entries []Entry) error {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	switch shell {
	case "bash":
		return bashCompletion(w, binary, sorted)
	case "fish":
		return fishCompletion(w, binary, sorted)
	case "zsh":
		return zshCompletion(w, binary, sorted)
	default:
		return shellError(shell)
	}
}

// operandGroup is a set of commands whose first operand accepts the same words.
//
// Commands that share a vocabulary become one branch: `help` and `man` both take
// command names, and a script spelling that out twice is two places to keep in step
// for one behaviour.
type operandGroup struct {
	// names are the subcommands the branch matches, in the order they were met.
	names []string
	// words are the values their first operand accepts.
	words []string
}

// operandGroups resolves each command's declared operand vocabulary to the words a
// shell should offer, merging the commands that share one.
//
// A command that declares no operand is absent, and absence is how a generated
// script declines: no branch matches, the offer stays empty, and the shell falls
// back to its own behaviour. Declaring a command with no operand rather than
// omitting it would put a branch that offers nothing, which is the same result
// written twice.
func operandGroups(entries []Entry) []operandGroup {
	commandNames := make([]string, 0, len(entries))
	for _, e := range entries {
		commandNames = append(commandNames, e.Name)
	}

	groups := make([]operandGroup, 0, len(entries))
	for _, e := range entries {
		var words []string
		switch {
		case e.Operands.CommandNames:
			words = commandNames
		case len(e.Operands.Values) > 0:
			words = e.Operands.Values
		default:
			continue
		}

		merged := false
		for i := range groups {
			if sameWords(groups[i].words, words) {
				groups[i].names = append(groups[i].names, e.Name)
				merged = true
				break
			}
		}
		if !merged {
			groups = append(groups, operandGroup{names: []string{e.Name}, words: words})
		}
	}
	return groups
}

// sameWords compares two vocabularies element by element, in order. The order is
// part of the identity: the same words in another order would be a different file,
// and this merge is about behaviour rather than about bytes.
func sameWords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// maxShellSuggestionDistance is the edit distance at which a mistyped shell name
// is still worth proposing. It is tighter than the command bound because the
// vocabulary is short: at two edits, "tcsh" is within reach of "zsh", "fish" and
// "bash" all at once, so a real request for an unsupported shell would be
// answered with all three supported ones.
//
// One edit already covers every typo these words invite — a dropped letter
// ("bas" for "bash"), a wrong one ("zh" for "zsh") or a transposition.
const maxShellSuggestionDistance = 1

// shellError reports an unsupported shell. It is a usage error, not a runtime
// failure: the caller names a different shell and gets the script, so
// api-compatibility.md's exit code 2 is the honest answer.
//
// The accepted vocabulary is named, and a near miss is proposed, so "bas" does
// not cost a reader a trip to the manual.
func shellError(shell string) error {
	err := fmt.Errorf("%w: completion: unknown shell %q: want one of %s",
		ErrUsage, shell, strings.Join(Shells, "|"))

	if near := suggestWithin(Shells, shell, maxShellSuggestionDistance); len(near) > 0 {
		return fmt.Errorf("%w; did you mean %s?", err, quoted(near))
	}
	return err
}

// bashCompletion writes a bash completion script for binary.
//
// Two positions are completed: the subcommand, and that subcommand's own first
// operand, whose vocabulary each command declares. Offering names for a later word
// would suggest values the caller cannot use there, so everything past the second
// position is declined — and bash's own filename completion, re-enabled with
// -o default, covers what this declines.
func bashCompletion(w io.Writer, binary string, entries []Entry) error {
	words := make([]string, 0, len(entries))
	for _, e := range entries {
		words = append(words, e.Name)
	}

	_, err := fmt.Fprintf(w, bashScript, binary, functionName(binary),
		strings.Join(words, " "), bashOperandCase(operandGroups(entries)))
	if err != nil {
		return fmt.Errorf("writing bash completion: %w", err)
	}
	return nil
}

// bashOperandCase renders the case that maps a typed subcommand to the words its
// first operand accepts, or the empty string when no command declares one.
//
// The vocabulary is a single quoted word rather than a list because compgen splits
// it on whitespace itself, which is what the existing command offer already relies
// on. The catch-all assigns an empty value rather than assigning nothing, so the
// caller tests one variable instead of knowing whether a branch matched.
func bashOperandCase(groups []operandGroup) string {
	if len(groups) == 0 {
		return ""
	}

	var b strings.Builder
	for _, g := range groups {
		fmt.Fprintf(&b, "                %s) words_for=%s ;;\n",
			strings.Join(g.names, "|"), quotePOSIX(strings.Join(g.words, " ")))
	}
	b.WriteString("                *) words_for= ;;\n")
	return b.String()
}

// zshCompletion writes a zsh completion script for binary.
//
// Descriptions are included because _describe renders an entry's text after a
// colon, so a caller browsing the menu sees what each command does rather than
// a bare list of words.
//
// The script has to work in both ways a caller can install it: named `_<binary>`
// in a directory on fpath, where compinit autoloads it, and sourced from a
// startup file. Those are different code paths — an autoloaded file *is* the
// body of the function compinit calls, a sourced one is not — so the template
// carries a dispatch. Only the first is documented by the generated header, and
// only the first is proven against a real zsh, by
// TestZshInstalledCompletionWorksViaCompinit.
func zshCompletion(w io.Writer, binary string, entries []Entry) error {
	var b strings.Builder
	for _, e := range entries {
		// The pair is one word; zsh splits it back apart on the first colon. A
		// colon inside the summary is therefore harmless and is left alone.
		fmt.Fprintf(&b, "        %s\n", quotePOSIX(e.Name+":"+e.Summary))
	}

	_, err := fmt.Fprintf(w, zshScript, binary, functionName(binary), b.String(),
		zshOperandCase(operandGroups(entries)))
	if err != nil {
		return fmt.Errorf("writing zsh completion: %w", err)
	}
	return nil
}

// zshOperandCase renders the case that maps a typed subcommand to the words its
// first operand accepts, or the empty string when no command declares one.
//
// Each branch *assigns* the array rather than listing words: a bare word inside a
// case branch is a command to zsh, not an element, and the first one runs and fails
// with "command not found". _describe then splits the array back into entries, the
// same way the command list is offered; a word list would arrive as one candidate and
// the menu would show a single line containing all of them.
func zshOperandCase(groups []operandGroup) string {
	if len(groups) == 0 {
		return ""
	}

	var b strings.Builder
	for _, g := range groups {
		quoted := make([]string, 0, len(g.words))
		for _, word := range g.words {
			quoted = append(quoted, quotePOSIX(word))
		}
		fmt.Fprintf(&b, "        %s)\n            operands=(%s)\n            ;;\n",
			strings.Join(g.names, "|"), strings.Join(quoted, " "))
	}
	b.WriteString("        *) ;;\n")
	return b.String()
}

// fishCompletion writes a fish completion script for binary.
//
// Descriptions are included because fish prints them beside the candidate, and
// __fish_use_subcommand narrows the offer to the one position where a
// subcommand name belongs.
func fishCompletion(w io.Writer, binary string, entries []Entry) error {
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "complete -c %s -n '__fish_use_subcommand' -a %s -d %s\n",
			binary, quoteFish(e.Name), quoteFish(e.Summary))
	}

	_, err := fmt.Fprintf(w, fishScript, binary, b.String(), fishOperandLines(binary, operandGroups(entries)))
	if err != nil {
		return fmt.Errorf("writing fish completion: %w", err)
	}
	return nil
}

// fishOperandLines renders one complete line per operand group.
//
// The token count is what makes this the *first* operand. fish keeps
// __fish_seen_subcommand_from true at every later position too, so without the
// count a vocabulary would keep being offered after the operand that already
// accepted one — shells to `kroot completion bash <TAB>`, which is exactly the
// suggestion a caller cannot use.
func fishOperandLines(binary string, groups []operandGroup) string {
	if len(groups) == 0 {
		return ""
	}

	var b strings.Builder
	for _, g := range groups {
		fmt.Fprintf(&b, "complete -c %s -f -n '__fish_seen_subcommand_from %s; and test (count (commandline -opc)) -eq 2' -a %s\n",
			binary, strings.Join(g.names, " "), quoteFish(strings.Join(g.words, " ")))
	}
	return b.String()
}

// functionName turns a binary name into a shell function name. A dash cannot
// appear in a function name, so a binary called "k-root" must not produce
// "_k-root", which every one of these shells would read as a subtraction rather
// than a name.
func functionName(binary string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			return r
		default:
			return '_'
		}
	}, binary)
}

// quotePOSIX renders s as one single-quoted word that bash and zsh read back
// verbatim. The single quote is the only character that needs escaping: inside
// single quotes every other character, backslash included, is already literal.
func quotePOSIX(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// quoteFish renders s as one single-quoted word for fish. fish honours exactly
// two escapes inside single quotes, so both the backslash and the quote have to
// be neutralised; every other character is already literal.
func quoteFish(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return "'" + strings.ReplaceAll(s, "'", `\'`) + "'"
}

// The three templates below are the whole of each generator's output. They are
// constants rather than built strings so a reviewer can read the script that
// ships without reconstructing it, and so a change to the script is a visible
// diff rather than a refactor.
//
// The bash and zsh templates take the binary name, the derived function name
// and the rendered command list; fish has no function to name and takes the
// binary name and the command list.
const bashScript = `# bash completion for %[1]s
#
# Generated by "%[1]s completion bash". This file is generated output: edit the
# kroot source rather than this script, and regenerate it after upgrading kroot
# or the command list below goes stale.
#
# Install:
#   %[1]s completion bash > /etc/bash_completion.d/%[1]s

_%[2]s_completions() {
    local cur words_for
    cur="${COMP_WORDS[COMP_CWORD]}"

    # Two positions are ever valid: the subcommand, and that subcommand's own
    # first operand. Any later word is declined rather than guessed at, and -o
    # default hands it to bash's filename completion.
    if [ "$COMP_CWORD" -eq 1 ]; then
        # Deliberately unquoted: compgen writes one word per line and COMPREPLY is a
        # word array, so the split *is* the assignment. Quoting it would produce a
        # single element holding all the newlines. The $(...) form rather than
        # mapfile keeps this working on the bash 3.2 that macOS still ships.
        COMPREPLY=($(compgen -W "%[3]s" -- "$cur"))
        return 0
    fi
    if [ "$COMP_CWORD" -ne 2 ]; then
        return 0
    fi

    # The vocabulary belongs to the subcommand that was typed, so the offer cannot
    # be a flat list: shells after completion, command names after help and man,
    # and nothing at all after a command that takes no operand.
    words_for=
    case "${COMP_WORDS[1]}" in
%[4]s    esac

    if [ -n "$words_for" ]; then
        COMPREPLY=($(compgen -W "$words_for" -- "$cur"))
    fi
    return 0
}

# -o default hands back every word this function declines to bash's own filename
# completion, so declining an operand does not leave the caller with nothing.
complete -o default -F _%[2]s_completions %[1]s
`

const zshScript = `#compdef %[1]s
#
# zsh completion for %[1]s
#
# Generated by "%[1]s completion zsh". This file is generated output: edit the
# kroot source rather than this script, and regenerate it after upgrading kroot
# or the command list below goes stale.
#
# Install under exactly this name: the #compdef line above is what compinit
# reads to associate this file with %[1]s, and compinit autoloads it as the
# function _%[2]s, which the dispatch at the end asks for by name:
#   %[1]s completion zsh > "${fpath[1]}/_%[2]s"

_%[2]s_completions() {
    local -a commands operands
    commands=(
%[3]s    )

    # Two positions are ever valid: the subcommand, and that subcommand's own
    # first operand. A later word is declined, the same answer every shell gives.
    if (( CURRENT == 2 )); then
        # _describe takes the current word from the caller, so filtering the list
        # against what has been typed is zsh's job rather than this script's.
        _describe -t commands '%[1]s command' commands
        return 0
    fi
    if (( CURRENT != 3 )); then
        return 1
    fi

    # The vocabulary belongs to the subcommand that was typed: shells after
    # completion, command names after help and man, and nothing at all after a
    # command that takes no operand.
    operands=()
    case "${words[2]}" in
%[4]s    esac

    if (( ${#operands} )); then
        _describe -t commands '%[1]s operand' operands
    fi
    return 0
}

# The file has to work in both ways a caller can install it. When compinit
# autoloads it from fpath, this body *is* the completion function, so it must
# complete here. When a startup file sources it, there is no such function and
# the file must register instead. funcstack names the function that is running,
# so it is what tells the two apart.
if [ "$funcstack[1]" = "_%[2]s" ]; then
    _%[2]s_completions "$@"
else
    # compdef arrives with compinit, not with a bare zsh. Without this guard a
    # caller who has never run compinit gets "command not found: compdef" on
    # their next shell startup instead of a completion.
    if ! whence compdef >/dev/null 2>&1; then
        autoload -Uz compinit && compinit
    fi
    compdef _%[2]s_completions %[1]s
fi
`

const fishScript = `# fish completion for %[1]s
#
# Generated by "%[1]s completion fish". This file is generated output: edit the
# kroot source rather than this script, and regenerate it after upgrading kroot
# or the command list below goes stale.
#
# Install:
#   %[1]s completion fish > ~/.config/fish/completions/%[1]s.fish

# -f: the operands of this program are subcommands, so do not offer filenames.
complete -c %[1]s -f
%[2]s
# The first operand of the subcommand that was typed, where one accepts a value.
# The token count is what confines it to that position: without it fish keeps
# offering the vocabulary after the operand that already accepted one.
%[3]s`
