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
	Name    string
	Summary string
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
// Only the first operand is completed, because that is the single position
// where a subcommand name is a valid answer; offering names for a later word
// would suggest commands the caller cannot use there. bash's own filename
// completion, re-enabled with -o default, covers everything this declines.
func bashCompletion(w io.Writer, binary string, entries []Entry) error {
	words := make([]string, 0, len(entries))
	for _, e := range entries {
		words = append(words, e.Name)
	}

	_, err := fmt.Fprintf(w, bashScript, binary, functionName(binary), strings.Join(words, " "))
	if err != nil {
		return fmt.Errorf("writing bash completion: %w", err)
	}
	return nil
}

// zshCompletion writes a zsh completion script for binary.
//
// Descriptions are included because _describe renders an entry's text after a
// colon, so a caller browsing the menu sees what each command does rather than
// a bare list of words.
func zshCompletion(w io.Writer, binary string, entries []Entry) error {
	var b strings.Builder
	for _, e := range entries {
		// The pair is one word; zsh splits it back apart on the first colon. A
		// colon inside the summary is therefore harmless and is left alone.
		fmt.Fprintf(&b, "        %s\n", quotePOSIX(e.Name+":"+e.Summary))
	}

	_, err := fmt.Fprintf(w, zshScript, binary, functionName(binary), b.String())
	if err != nil {
		return fmt.Errorf("writing zsh completion: %w", err)
	}
	return nil
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

	_, err := fmt.Fprintf(w, fishScript, binary, b.String())
	if err != nil {
		return fmt.Errorf("writing fish completion: %w", err)
	}
	return nil
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
    local cur
    cur="${COMP_WORDS[COMP_CWORD]}"

    # A subcommand is only ever the first operand. Offering names for any later
    # word would suggest commands the caller cannot use in that position.
    if [ "$COMP_CWORD" -ne 1 ]; then
        return 0
    fi

    # Deliberately unquoted: compgen writes one word per line and COMPREPLY is a
    # word array, so the split *is* the assignment. Quoting it would produce a
    # single element holding all the newlines. The $(...) form rather than
    # mapfile keeps this working on the bash 3.2 that macOS still ships.
    COMPREPLY=($(compgen -W "%[3]s" -- "$cur"))
    return 0
}

# -o default hands back every word this function declines to bash's own filename
# completion, so declining an operand does not leave the caller with nothing.
complete -o default -F _%[2]s_completions %[1]s
`

const zshScript = `# zsh completion for %[1]s
#
# Generated by "%[1]s completion zsh". This file is generated output: edit the
# kroot source rather than this script, and regenerate it after upgrading kroot
# or the command list below goes stale.
#
# Install:
#   %[1]s completion zsh > "${fpath[1]}/_%[2]s"

_%[2]s_completions() {
    local -a commands
    commands=(
%[3]s    )

    # _describe takes the current word from the caller, so filtering the list
    # against what has been typed is zsh's job rather than this script's.
    _describe -t commands '%[1]s command' commands
}

# compdef arrives with compinit, not with a bare zsh. Without this guard a
# caller who has never run compinit gets "command not found: compdef" on their
# next shell startup instead of a completion.
if ! whence compdef >/dev/null 2>&1; then
    autoload -Uz compinit
    compinit
fi

compdef _%[2]s_completions %[1]s
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
%[2]s`
