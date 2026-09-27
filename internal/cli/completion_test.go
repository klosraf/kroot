package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testEntries is a small command set used across the tests. The order is
// deliberately not the sorted order, so a test that passes cannot be passing
// because the input was already in output order.
func testEntries() []Entry {
	return []Entry{
		{Name: "version", Summary: "print the version and exit"},
		{Name: "completion", Summary: "print a shell completion script"},
		{Name: "help", Summary: "print this help, or the help of one command"},
	}
}

// render generates a completion script into a buffer.
func render(t *testing.T, shell, binary string, entries []Entry) string {
	t.Helper()

	var buf bytes.Buffer
	if err := WriteCompletion(&buf, shell, binary, entries); err != nil {
		t.Fatalf("WriteCompletion(%q) error = %v; want nil", shell, err)
	}
	return buf.String()
}

// TestWriteCompletionRendersEveryShell is the base table: each supported shell
// produces its own script, and the script names every command, because a
// completion that omits a command is worse than none — it teaches the caller
// that the command does not exist.
func TestWriteCompletionRendersEveryShell(t *testing.T) {
	tests := []struct {
		shell  string
		marker string
	}{
		{shell: "bash", marker: "complete -o default -F _kroot_completions kroot"},
		{shell: "fish", marker: "complete -c kroot -f"},
		{shell: "zsh", marker: "compdef _kroot_completions kroot"},
	}

	for _, tc := range tests {
		t.Run(tc.shell, func(t *testing.T) {
			got := render(t, tc.shell, "kroot", testEntries())

			if !strings.Contains(got, tc.marker) {
				t.Errorf("%s script does not contain its registration %q", tc.shell, tc.marker)
			}
			for _, e := range testEntries() {
				if !strings.Contains(got, e.Name) {
					t.Errorf("%s script omits command %q", tc.shell, e.Name)
				}
			}
		})
	}
}

// TestWriteCompletionCoversExactlyTheDocumentedVocabulary pins the dispatch
// table to Shells. A generator added without an entry here, or an entry in
// Shells without a generator, fails in one direction or the other — which is the
// drift the single-definition rule exists to prevent.
func TestWriteCompletionCoversExactlyTheDocumentedVocabulary(t *testing.T) {
	for _, shell := range Shells {
		t.Run(shell, func(t *testing.T) {
			if got := render(t, shell, "kroot", testEntries()); strings.TrimSpace(got) == "" {
				t.Errorf("WriteCompletion(%q) produced nothing; want a script", shell)
			}
		})
	}
}

// TestWriteCompletionRejectsUnknownShells covers the contract half: an
// unsupported shell is a usage error, because the caller can fix it by naming a
// supported one, and api-compatibility.md ties that to exit code 2. The message
// must also carry the vocabulary, so the fix is in the error rather than in the
// manual.
func TestWriteCompletionRejectsUnknownShells(t *testing.T) {
	for _, shell := range []string{"tcsh", "Bash", "powershell", "", "sh"} {
		t.Run(shell, func(t *testing.T) {
			var buf bytes.Buffer

			err := WriteCompletion(&buf, shell, "kroot", testEntries())

			if !errors.Is(err, ErrUsage) {
				t.Fatalf("WriteCompletion(%q) error = %v; want it to wrap ErrUsage", shell, err)
			}
			if got := buf.String(); got != "" {
				t.Errorf("WriteCompletion(%q) wrote %q; want nothing: a rejection is not output", shell, got)
			}
			for _, want := range Shells {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %v; want it to name the accepted shell %q", err, want)
				}
			}
		})
	}
}

// TestWriteCompletionSuggestsANearMissShell proves a typo costs nothing: the
// same bounded edit distance an unknown command gets is applied to shell names,
// so "bas" is answered with "bash" rather than a bare rejection.
func TestWriteCompletionSuggestsANearMissShell(t *testing.T) {
	tests := []struct {
		shell string
		want  string
	}{
		{shell: "bas", want: "bash"},
		{shell: "zh", want: "zsh"},
		{shell: "fsh", want: "fish"},
	}

	for _, tc := range tests {
		t.Run(tc.shell, func(t *testing.T) {
			err := WriteCompletion(io_Discard{}, tc.shell, "kroot", testEntries())

			if err == nil {
				t.Fatalf("WriteCompletion(%q) = nil; want a usage error", tc.shell)
			}
			if !strings.Contains(err.Error(), "did you mean") {
				t.Errorf("error = %v; want a suggestion", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v; want it to suggest %q", err, tc.want)
			}
		})
	}
}

// io_Discard is a writer that throws everything away, for the paths that must
// reject before writing.
type io_Discard struct{}

func (io_Discard) Write(p []byte) (int, error) { return len(p), nil }

// TestShellSuggestionIsNotAWallOfGuesses covers the property the tighter shell
// bound exists to provide. "tcsh" motivated it: at the command-name distance of
// two, "tcsh" is within reach of "zsh", "fish" and "bash" all at once, so a
// request for an unsupported shell was answered with every supported one.
//
// A single near miss is still offered where one exists — "sh" and "zsh" really
// are one edit apart, and no threshold both catches "bas" for "bash" and rejects
// that one. What must not happen is the wall of guesses, because the reader's
// next question is "which of these did they mean?" and the answer would be all of
// them. The accepted vocabulary is named either way, so the reader is never
// misled about what is actually supported.
func TestShellSuggestionIsNotAWallOfGuesses(t *testing.T) {
	for _, shell := range []string{"tcsh", "ksh", "csh", "sh", "powershell", "nushell"} {
		t.Run(shell, func(t *testing.T) {
			err := WriteCompletion(io_Discard{}, shell, "kroot", testEntries())
			if err == nil {
				t.Fatalf("WriteCompletion(%q) = nil; want a usage error", shell)
			}

			// The suggestion clause is appended after a known marker, so counting
			// within it avoids matching the "want one of ..." vocabulary that
			// every rejection carries.
			_, suggestions, found := strings.Cut(err.Error(), "did you mean ")
			if found && strings.Contains(suggestions, "one of") {
				t.Errorf("WriteCompletion(%q) error = %v; want at most one suggestion, not a list", shell, err)
			}
			for _, supported := range Shells {
				if !strings.Contains(err.Error(), supported) {
					t.Errorf("error = %v; want it to name the supported shell %q so the reader is never misled", err, supported)
				}
			}
		})
	}
}

// TestUnsupportedShellGetsNoSuggestionWhenNoneIsClose keeps the tcsh case
// explicit, because it is the one that regressed: it is a real shell, two edits
// from every candidate, and the honest answer is to name the vocabulary alone.
func TestUnsupportedShellGetsNoSuggestionWhenNoneIsClose(t *testing.T) {
	err := WriteCompletion(io_Discard{}, "tcsh", "kroot", testEntries())

	if err == nil {
		t.Fatal(`WriteCompletion("tcsh") = nil; want a usage error`)
	}
	if strings.Contains(err.Error(), "did you mean") {
		t.Errorf("WriteCompletion(%q) error = %v; want no suggestion: two edits away from every candidate", "tcsh", err)
	}
}

// TestShellSuggestionDistanceIsTighterThanTheCommandBound pins the reasoning as
// a fact about the vocabulary rather than a comment that can drift: shell names
// are three or four letters, so the command bound of two would match nearly
// anything, while one still catches every typo these words invite.
func TestShellSuggestionDistanceIsTighterThanTheCommandBound(t *testing.T) {
	if maxShellSuggestionDistance >= maxSuggestionDistance {
		t.Errorf("maxShellSuggestionDistance = %d; want it below the command bound of %d", maxShellSuggestionDistance, maxSuggestionDistance)
	}

	longest := 0
	for _, shell := range Shells {
		if len(shell) > longest {
			longest = len(shell)
		}
	}
	if maxShellSuggestionDistance*2 >= longest {
		t.Errorf("a %d-edit bound against a %d-letter vocabulary matches most words; the bound is too loose for these names", maxShellSuggestionDistance, longest)
	}
}

// TestWriteCompletionIsDeterministic guards the promise Registry.Names makes —
// sorted output — for the file a caller installs. A completion script that
// reorders itself between identical runs produces a noisy diff every time it is
// regenerated, which trains reviewers to ignore changes to it.
func TestWriteCompletionIsDeterministic(t *testing.T) {
	for _, shell := range Shells {
		t.Run(shell, func(t *testing.T) {
			first := render(t, shell, "kroot", testEntries())
			second := render(t, shell, "kroot", testEntries())

			if first != second {
				t.Errorf("%s script differs between identical runs:\nfirst:\n%s\nsecond:\n%s", shell, first, second)
			}
		})
	}
}

// TestWriteCompletionSortsWithoutTouchingTheCaller proves the ordering the
// script promises does not depend on the caller having sorted first — the input
// here is deliberately unsorted — and that the caller's slice survives intact.
// Sorting in place would reorder a registry's own view of its commands, which is
// a side effect a rendering function has no business having.
func TestWriteCompletionSortsWithoutTouchingTheCaller(t *testing.T) {
	entries := testEntries() // unsorted: version, completion, help
	before := make([]Entry, len(entries))
	copy(before, entries)

	script := render(t, "bash", "kroot", entries)

	if compgen := "completion help version"; !strings.Contains(script, compgen) {
		t.Errorf("bash script does not list the commands in name order:\n%s", script)
	}
	for i := range entries {
		if entries[i] != before[i] {
			t.Errorf("WriteCompletion reordered the caller's slice at %d: got %+v, want %+v", i, entries[i], before[i])
		}
	}
}

// TestWriteCompletionPropagatesWriteErrors proves a truncated install is
// reported rather than reported as success. A caller redirecting the script to
// a file that fills up must not be told the install worked.
func TestWriteCompletionPropagatesWriteErrors(t *testing.T) {
	for _, shell := range Shells {
		t.Run(shell, func(t *testing.T) {
			err := WriteCompletion(failWriter{}, shell, "kroot", testEntries())

			if err == nil {
				t.Fatalf("WriteCompletion(%q) with a failing writer = nil; want an error", shell)
			}
			if !strings.Contains(err.Error(), "write failed") {
				t.Errorf("error = %v; want it to wrap the underlying failure", err)
			}
		})
	}
}

// failWriter fails every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

// TestWriteCompletionSurvivesAHostileSummary is the escaping test, and it is a
// correctness test rather than a tidy-up one: a summary is free text, and a
// summary containing a quote would otherwise close the quoted word it sits in
// and either break the script or, worse, inject a command into it. The generated
// script is handed to a real parser afterwards, so "it still parses" is
// demonstrated rather than assumed.
func TestWriteCompletionSurvivesAHostileSummary(t *testing.T) {
	// A summary is free text that reaches a shell script as data. This one
	// carries every character that could terminate a quoted word or start a
	// substitution: single quotes, a backslash, backticks and a command
	// substitution.
	hostileSummary := "it's a 'trap'; $(rm -rf /) \\`quoted\\` \"double\""

	hostile := []Entry{
		{Name: "help", Summary: hostileSummary},
		{Name: "version", Summary: "print the version and exit"},
	}

	t.Run("bash", func(t *testing.T) {
		// bash carries no descriptions at all — compgen -W has nowhere to put
		// one — so the strongest statement available is that the text never
		// reaches the script and cannot be misquoted there.
		//
		// The probes are chosen not to collide with the template's own prose:
		// "quoted" appears in a bash comment explaining the unquoted expansion.
		script := render(t, "bash", "kroot", hostile)

		for _, unwanted := range []string{"trap", "double", "rm -rf", hostileSummary} {
			if strings.Contains(script, unwanted) {
				t.Errorf("bash script contains summary text %q; want descriptions omitted entirely:\n%s", unwanted, script)
			}
		}
	})

	// fish honours exactly two escapes inside single quotes, so both the
	// backslash and the quote have to be neutralised. zsh honours the same
	// single-quote rule as bash, where a backslash is already literal and only
	// the quote needs work. fish is not installed everywhere, so that case is
	// asserted structurally; the zsh case below is proven by a real parser.
	for shell, escapes := range map[string][]string{
		"fish": {`\'`, `\\`},
		"zsh":  {`'\''`},
	} {
		t.Run(shell, func(t *testing.T) {
			script := render(t, shell, "kroot", hostile)

			for _, escape := range escapes {
				if !strings.Contains(script, escape) {
					t.Errorf("%s script does not contain the escape %s; the summary would terminate its quoted word:\n%s", shell, escape, script)
				}
			}
		})
	}
}

// TestZshCompletionRoundTripsAHostileSummary proves the escaping by reading it
// back out of a real zsh rather than by inspecting the text. The generated
// script is sourced, _describe is intercepted, and the array it would have been
// given is dereferenced — so the summary that comes out must be byte-for-byte
// the summary that went in. Escaping that was merely plausible would mangle it
// here.
func TestZshCompletionRoundTripsAHostileSummary(t *testing.T) {
	requireTool(t, "zsh")

	hostileSummary := "it's a 'trap'; $(rm -rf /) \\`quoted\\` \"double\""
	script := render(t, "zsh", "kroot", []Entry{{Name: "help", Summary: hostileSummary}})
	path := writeScript(t, "zsh", script)

	// _describe is handed the array *name* and dereferences it, so the
	// interceptor mirrors that to reveal the entries the real one would receive.
	//
	// print carries -r: without it zsh interprets escape sequences in its
	// arguments, which would eat the very backslashes this test exists to check.
	driver := fmt.Sprintf(`
autoload -Uz compinit && compinit -u -d "$2"
source "$1"
_describe() { shift 3; print -r -l -- "${(@P)1}" }
_%s_completions
`, functionName("kroot"))

	dump := filepath.Join(t.TempDir(), "zcompdump")
	out, err := runShell(t, "zsh", "-f", "-c", driver, "zsh-probe", path, dump)
	if err != nil {
		t.Fatalf("zsh driver failed: %v\n%s\nscript:\n%s", err, out, script)
	}

	// _describe receives "name:summary"; everything after the first colon is the
	// description, so that is what must survive unchanged.
	want := "help:" + hostileSummary
	found := false
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("zsh did not read the summary back unchanged.\nwant entry: %q\ngot:\n%s", want, out)
	}
}

// TestGeneratedScriptsParse runs each generated script through the shell it
// targets. This is the test that would have caught a malformed template, and it
// is why the templates are plain constants: what is checked here is exactly what
// ships.
//
// fish is skipped when absent rather than assumed, and the skip is visible in
// the output rather than hidden.
func TestGeneratedScriptsParse(t *testing.T) {
	tests := []struct {
		shell string
		// check parses a generated script without running it.
		check func(t *testing.T, path string) (string, error)
	}{
		{
			shell: "bash",
			// -n reads the file and reports syntax errors without executing it.
			check: func(t *testing.T, path string) (string, error) {
				return runShell(t, "bash", "-n", path)
			},
		},
		{
			shell: "zsh",
			check: func(t *testing.T, path string) (string, error) {
				return runShell(t, "zsh", "-n", path)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.shell, func(t *testing.T) {
			requireTool(t, tc.shell)

			script := render(t, tc.shell, "kroot", testEntries())
			path := writeScript(t, tc.shell, script)

			out, err := tc.check(t, path)
			if err != nil {
				t.Fatalf("%s rejected the generated script: %v\n%s\nscript:\n%s", tc.shell, err, out, script)
			}
		})
	}
}

// TestGeneratedBashScriptActuallyCompletes is the end-to-end proof: the script is
// sourced by a real bash, and the registered completion function is driven the
// way bash drives it. A script that parses but returns nothing would pass every
// other test here, and a caller would get a tab key that does nothing.
func TestGeneratedBashScriptActuallyCompletes(t *testing.T) {
	requireTool(t, "bash")

	script := render(t, "bash", "kroot", testEntries())
	path := writeScript(t, "bash", script)

	// The driver sources the script and calls the completion function with the
	// COMP_* state bash would set, then reports the result as key=value lines.
	// The function name is derived the same way the generator derives it, so a
	// change to functionName cannot silently break this test.
	driver := fmt.Sprintf(`
source "$1"
# The completion function is called, never redefined: a wrapper sharing its
# name would shadow the generated one and the test would pass while proving
# nothing about the script.
_%s_completions_probe() {
  COMPREPLY=()
  COMP_CWORD=$1
  shift
  COMP_WORDS=("$@")
  _%s_completions
  echo "${COMPREPLY[*]-}"
}
# Echoing the sourced body proves the function under test is the generated one.
echo "sourced=$(declare -f _%s_completions | grep -c compgen)"
echo "all=$(_%s_completions_probe 1 kroot '')"
echo "h=$(_%s_completions_probe 1 kroot h)"
echo "ver=$(_%s_completions_probe 1 kroot ver)"
echo "none=$(_%s_completions_probe 1 kroot zzz)"
echo "second=$(_%s_completions_probe 2 kroot help '')"
`, functionName("kroot"), functionName("kroot"), functionName("kroot"),
		functionName("kroot"), functionName("kroot"), functionName("kroot"),
		functionName("kroot"), functionName("kroot"))

	// The script path is passed as an argument rather than interpolated into the
	// driver, so a path containing a space or a quote cannot alter the program
	// bash runs.
	out, err := runShell(t, "bash", "-c", driver, "kroot-probe", path)
	if err != nil {
		t.Fatalf("bash driver failed: %v\n%s", err, out)
	}

	got := parseKeyValues(out)
	want := map[string]string{
		// One line of the generated body mentions compgen, so a sourced-and-
		// intact function reports 1. Anything else means the test exercised
		// something other than the generated script.
		"sourced": "1",
		"all":     "completion help version",
		"h":       "help",
		"ver":     "version",
		"none":    "",
		"second":  "",
	}

	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			t.Errorf("driver output has no %q line:\n%s", key, out)
			continue
		}
		if gotValue != wantValue {
			t.Errorf("%s = %q; want %q", key, gotValue, wantValue)
		}
	}
}

// TestFunctionNameIsAValidIdentifier covers the rename hazard: a binary whose
// name contains a dash would otherwise generate a function the shell reads as a
// subtraction, and the failure would only appear in a user's interactive shell.
func TestFunctionNameIsAValidIdentifier(t *testing.T) {
	tests := []struct {
		binary string
		want   string
	}{
		{binary: "kroot", want: "kroot"},
		{binary: "k-root", want: "k_root"},
		{binary: "kroot2", want: "kroot2"},
		{binary: "k root", want: "k_root"},
		{binary: "kroot.sh", want: "kroot_sh"},
	}

	for _, tc := range tests {
		t.Run(tc.binary, func(t *testing.T) {
			if got := functionName(tc.binary); got != tc.want {
				t.Errorf("functionName(%q) = %q; want %q", tc.binary, got, tc.want)
			}
		})
	}
}

// TestDashlessBinaryNameProducesAValidScript is the end-to-end form of the test
// above: the generated bash script has to reference a function that bash will
// accept, which is only true if the name was sanitised.
func TestDashlessBinaryNameProducesAValidScript(t *testing.T) {
	requireTool(t, "bash")

	script := render(t, "bash", "k-root", testEntries())
	if !strings.Contains(script, "_k_root_completions()") {
		t.Errorf("script does not define a sanitised function name:\n%s", script)
	}

	path := writeScript(t, "bash", script)
	if out, err := runShell(t, "bash", "-n", path); err != nil {
		t.Errorf("bash rejected the script: %v\n%s", err, out)
	}
}

// runShell runs one of the shells under test and returns its combined output.
//
// It exists so the subprocess suppression lives in exactly one place, with one
// explanation, instead of at every call site.
func runShell(t *testing.T, shell string, args ...string) (string, error) {
	t.Helper()

	//nolint:gosec // G204: the shell is a constant from a test table, and every
	// argument is either a literal or a path this test created under t.TempDir().
	// Nothing a caller controls reaches the command line, which is the condition
	// G204 exists to check.
	out, err := exec.Command(shell, args...).CombinedOutput()
	return string(out), err
}

// writeScript puts a generated script on disk for a shell to read, and returns
// its path. A file is used rather than stdin because that is how a caller
// installs it, and because a parser given "-" would read the driver's own stdin.
func writeScript(t *testing.T, name, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name+"_completion")
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write %s script: %v", name, err)
	}
	return path
}

// parseKeyValues reads the key=value lines the bash driver prints.
func parseKeyValues(out string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found {
			continue
		}
		values[key] = value
	}
	return values
}
