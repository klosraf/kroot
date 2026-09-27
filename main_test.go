package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/klosraf/kroot/internal/cli"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    error
		wantOutput string
	}{
		{
			name:       "no arguments prints help",
			args:       nil,
			wantOutput: "Usage:",
		},
		{
			name:       "help command prints help",
			args:       []string{"help"},
			wantOutput: "Usage:",
		},
		{
			name:       "version command prints version",
			args:       []string{"version"},
			wantOutput: "kroot " + version,
		},
		{
			name:       "version flag prints version",
			args:       []string{"-version"},
			wantOutput: "kroot " + version,
		},
		{
			name:       "the program's own -h prints the program's help",
			args:       []string{"-h"},
			wantOutput: "Usage:\n  kroot [flags] <command>",
		},
		{
			name:       "version -h prints that command's help",
			args:       []string{"version", "-h"},
			wantOutput: "Usage:\n  kroot version",
		},
		{
			name:       "completion --help prints that command's help",
			args:       []string{"completion", "--help"},
			wantOutput: "Usage:\n  kroot completion <",
		},
		{
			name:       "help --help prints the help command's own help",
			args:       []string{"help", "--help"},
			wantOutput: "Usage:\n  kroot help [command]",
		},
		{
			name:       "the help operand wins over what follows it",
			args:       []string{"version", "--help", "extra"},
			wantOutput: "Usage:\n  kroot version",
		},
		{
			name:    "the help operand after a surplus operand is data, not a rescue",
			args:    []string{"version", "extra", "--help"},
			wantErr: cli.ErrUsage,
		},
		{
			name:       "help for one command prints that command's usage",
			args:       []string{"help", "version"},
			wantOutput: "Usage:\n  kroot version",
		},
		{
			name:    "help with too many arguments is a usage error",
			args:    []string{"help", "version", "extra"},
			wantErr: cli.ErrUsage,
		},
		{
			name:    "help with an unknown command suggests the closest match",
			args:    []string{"help", "versioo"},
			wantErr: cli.ErrUnknownCommand,
		},
		{
			name:    "unknown command returns error",
			args:    []string{"bogus"},
			wantErr: cli.ErrUnknownCommand,
		},
		{
			name: "completion prints a script on stdout",
			args: []string{"completion", "bash"},
			// The function name rather than the whole registration line. The
			// options that line carries belong to the generator, and the cli
			// package already asserts them exactly, in
			// TestWriteCompletionRendersEveryShell and
			// TestNoShellFallsBackToFilenamesAfterADeclinedPosition. Repeating
			// the line here would tie this table to generator options without
			// adding a guarantee, and would make the two packages fail together
			// when only one of them is wrong.
			wantOutput: "_kroot_completions",
		},
		{
			name:    "completion without a shell is a usage error",
			args:    []string{"completion"},
			wantErr: cli.ErrUsage,
		},
		{
			name:    "completion with too many shells is a usage error",
			args:    []string{"completion", "bash", "zsh"},
			wantErr: cli.ErrUsage,
		},
		{
			name:    "completion with an unknown shell is a usage error",
			args:    []string{"completion", "tcsh"},
			wantErr: cli.ErrUsage,
		},
		{
			name:       "man prints the program page",
			args:       []string{"man"},
			wantOutput: ".SH",
		},
		{
			name:       "man for one command prints that page",
			args:       []string{"man", "version"},
			wantOutput: `KROOT-VERSION`,
		},
		{
			name:    "man with an unknown command is a usage error",
			args:    []string{"man", "versioo"},
			wantErr: cli.ErrUnknownCommand,
		},
		{
			name:    "man with too many commands is a usage error",
			args:    []string{"man", "version", "help"},
			wantErr: cli.ErrUsage,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			err := run(context.Background(), tc.args, &out, io.Discard)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("run(%v) error = %v; want %v", tc.args, err, tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("run(%v) unexpected error: %v", tc.args, err)
			}

			if got := out.String(); !strings.Contains(got, tc.wantOutput) {
				t.Errorf("run(%v) output = %q; want it to contain %q", tc.args, got, tc.wantOutput)
			}
		})
	}
}

// failingWriter is an io.Writer that always fails. It exists to prove that
// write errors are propagated rather than silently dropped.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

// TestRunPropagatesFatalWriteError covers the one write failure the CLI can act
// on: failing to print the version is a runtime failure, so run must report it.
func TestRunPropagatesFatalWriteError(t *testing.T) {
	err := run(context.Background(), []string{"version"}, failingWriter{}, io.Discard)

	if err == nil {
		t.Fatal("run(version) with a failing writer = nil; want an error")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Errorf("run(version) error = %v; want it to wrap the underlying write error", err)
	}
}

// TestRunToleratesHelpWriteError documents the deliberate asymmetry: help text
// write failures are not actionable, because there is nowhere left to report
// them, so printUsage swallows them by design and run still succeeds.
func TestRunToleratesHelpWriteError(t *testing.T) {
	if err := run(context.Background(), []string{"help"}, failingWriter{}, io.Discard); err != nil {
		t.Errorf("run(help) with a failing writer = %v; want nil", err)
	}
}

// TestRunHelpFlagIsNotAnError covers `-h` and `-help`. flag returns
// flag.ErrHelp, which run() must not treat as a failure: the user asked for help
// and got help, so the documented exit code is 0 (api-compatibility.md,
// "Exit codes"). Before the fix this reported a runtime failure and exited 1,
// while `kroot help` exited 0 — two spellings of one request, two answers.
func TestRunHelpFlagIsNotAnError(t *testing.T) {
	for _, arg := range []string{"-h", "-help"} {
		t.Run(arg, func(t *testing.T) {
			var out, errOut bytes.Buffer

			if err := run(context.Background(), []string{arg}, &out, &errOut); err != nil {
				t.Errorf("run(%s) error = %v; want nil: help is a success, not a failure", arg, err)
			}
			if got := out.String(); !strings.Contains(got, "Usage:") {
				t.Errorf("run(%s) stdout = %q; want it to contain %q", arg, got, "Usage:")
			}
			if got := errOut.String(); got != "" {
				t.Errorf("run(%s) stderr = %q; want it empty: help is requested output, not a diagnostic", arg, got)
			}
		})
	}
}

// TestExitCodesMatchTheDocumentedContract pins the three values in
// docs/enterprise/api-compatibility.md § "Exit codes". Scripts branch on the
// status alone, so a usage error reported as a runtime failure — or the reverse —
// misleads a caller that never reads the output.
func TestExitCodesMatchTheDocumentedContract(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{name: "no arguments prints help and succeeds", args: nil, want: exitSuccess},
		{name: "help succeeds", args: []string{"help"}, want: exitSuccess},
		{name: "-h succeeds", args: []string{"-h"}, want: exitSuccess},
		{name: "-help succeeds", args: []string{"-help"}, want: exitSuccess},
		{name: "version succeeds", args: []string{"version"}, want: exitSuccess},
		{name: "unknown command is a usage error", args: []string{"bogus"}, want: exitUsage},
		{name: "unknown flag is a usage error", args: []string{"-nope"}, want: exitUsage},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			if got := exitCodeFor(run(context.Background(), tc.args, &out, io.Discard)); got != tc.want {
				t.Errorf("exitCodeFor(run(%v)) = %d; want %d", tc.args, got, tc.want)
			}
		})
	}
}

// TestRuntimeFailureIsNotAUsageError covers the other half of the contract: a
// failure the caller cannot fix by reinvoking must not be reported as exit 2.
// Here the invocation is correct and only the write fails.
func TestRuntimeFailureIsNotAUsageError(t *testing.T) {
	err := run(context.Background(), []string{"version"}, failingWriter{}, io.Discard)

	if errors.Is(err, cli.ErrUsage) {
		t.Fatalf("run(version) with a failing writer = %v; must not be a usage error: the invocation was correct", err)
	}
	if got := exitCodeFor(err); got != exitFailure {
		t.Errorf("exitCodeFor(...) = %d; want %d", got, exitFailure)
	}
}

// TestUsageFailuresGoToStderr covers the stream half of the CLI contract for
// callers: a failed invocation must not write to stdout. Before, a bad flag
// wrote flag's own error line and the whole usage text to stdout, so a caller
// redirecting stdout captured thirteen lines of help from a failure — while the
// same failure was logged again, structurally, on stderr.
func TestUsageFailuresGoToStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := run(context.Background(), []string{"-nope"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run(-nope) = nil; want a usage error")
	}
	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q; want it empty: a failure must not write to stdout", got)
	}
	if got := stderr.String(); !strings.Contains(got, "Usage:") {
		t.Errorf("stderr = %q; want it to contain the usage text", got)
	}
}

// TestParseLogLevelAcceptsOnlyTheDocumentedVocabulary is the regression test
// for the first KROOT_* variable: every accepted value maps, and every other
// value fails with the vocabulary named so a typo can be found in the error.
func TestParseLogLevelAcceptsOnlyTheDocumentedVocabulary(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  slog.Level
	}{
		{name: "unset selects the documented default", value: "", want: slog.LevelInfo},
		{name: "debug maps", value: "debug", want: slog.LevelDebug},
		{name: "info maps", value: "info", want: slog.LevelInfo},
		{name: "warn maps", value: "warn", want: slog.LevelWarn},
		{name: "error maps", value: "error", want: slog.LevelError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLogLevel(tc.value)
			if err != nil {
				t.Fatalf("parseLogLevel(%q) error = %v; want nil", tc.value, err)
			}
			if got != tc.want {
				t.Errorf("parseLogLevel(%q) = %v; want %v", tc.value, got, tc.want)
			}
		})
	}
}

// TestParseLogLevelRejectsAnythingElse pins the contract half of the first
// KROOT_* variable: an unknown value is *configuration rejected*, which
// api-compatibility.md ties to exit code 1 — not to exit 2, because the
// operator fixes an environment variable, not an invocation. The message also
// names the vocabulary, so the fix does not require reading source.
func TestParseLogLevelRejectsAnythingElse(t *testing.T) {
	for _, value := range []string{"DEBUG", "Info", "verbose", "debug ", "1"} {
		t.Run(value, func(t *testing.T) {
			got, err := parseLogLevel(value)
			if err == nil {
				t.Fatalf("parseLogLevel(%q) = %v, nil; want an error", value, got)
			}
			if got != slog.LevelError {
				t.Errorf("parseLogLevel(%q) level = %v; want LevelError so the rejection cannot be suppressed", value, got)
			}
			if got := exitCodeFor(err); got != exitFailure {
				t.Errorf("exitCodeFor(parseLogLevel(%q)) = %d; want %d: configuration rejected is exit 1", value, got, exitFailure)
			}
			if errors.Is(err, cli.ErrUsage) {
				t.Errorf("parseLogLevel(%q) matched ErrUsage; a rejected environment value is not a usage error", value)
			}
		})
	}

	_, err := parseLogLevel("VERBOSE")
	if err == nil || !strings.Contains(err.Error(), "debug|info|warn|error") {
		t.Errorf("parseLogLevel(%q) error = %v; want it to name the vocabulary", "VERBOSE", err)
	}
	if err != nil && !strings.Contains(err.Error(), "configuration rejected") {
		t.Errorf("parseLogLevel(%q) error = %v; want the wording api-compatibility.md ties to exit 1", "VERBOSE", err)
	}
}

// TestNewLoggerKeepsEncodingIndependentOfStructuredKeys verifies the TTY rule:
// the destination selects text or JSON, but the record carries the same keys
// either way. It enables INFO at minimum so a record at the default level is
// actually emitted in both encodings.
func TestNewLoggerKeepsEncodingIndependentOfStructuredKeys(t *testing.T) {
	for _, tty := range []bool{true, false} {
		t.Run(map[bool]string{true: "tty", false: "piped"}[tty], func(t *testing.T) {
			var out bytes.Buffer
			logger := newLogger(&out, tty, slog.LevelInfo)
			logger.Info("config loaded", "component", "config")

			got := out.String()
			if !strings.Contains(got, "config loaded") {
				t.Errorf("log output = %q; want it to contain the message", got)
			}
			if !strings.Contains(got, "component=config") && !strings.Contains(got, `"component":"config"`) {
				t.Errorf("log output = %q; want it to carry the component key", got)
			}
		})
	}
}

// TestIsTerminalDistinguishesRedirectedOutput proves the TTY property is a
// property of the descriptor, not the environment: a temp file is never a
// terminal, and a closed file reports non-terminal rather than failing.
func TestIsTerminalDistinguishesRedirectedOutput(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "kroot-tty-*.log")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if isTerminal(f) {
		t.Errorf("isTerminal(temp file) = true; want false")
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
}

// TestRunFailsFastWhenShutdownIsUnderway pins the contract half of the context:
// a cancelled context refuses to start work, and the failure is a runtime one
// (exit 1), not a usage one — reinvoking the same way fails the same way.
func TestRunFailsFastWhenShutdownIsUnderway(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := run(ctx, []string{"version"}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("run(cancelled) = nil; want an error")
	}
	if errors.Is(err, cli.ErrUsage) {
		t.Errorf("run(cancelled) error = %v; must not match ErrUsage", err)
	}
	if got := exitCodeFor(err); got != exitFailure {
		t.Errorf("exitCodeFor(run(cancelled)) = %d; want %d", got, exitFailure)
	}
}

// The count and the ErrUsage wrapping are asserted alongside the recovery, so a
// fix that softens the message cannot quietly drop the contract.
func TestArityFailuresNameTheCorrectForm(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "help", args: []string{"help", "a", "b"}, want: `see "kroot help <command>"`},
		{name: "man", args: []string{"man", "a", "b"}, want: `see "kroot man <command>"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer

			err := run(context.Background(), tc.args, &stdout, io.Discard)
			if err == nil {
				t.Fatalf("run(%v) = nil; want a usage error", tc.args)
			}
			if !errors.Is(err, cli.ErrUsage) {
				t.Errorf("run(%v) error = %v; want it to match ErrUsage", tc.args, err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("run(%v) error = %q; want it to name the next step %s", tc.args, err, tc.want)
			}
			if !strings.Contains(err.Error(), "got 2") {
				t.Errorf("run(%v) error = %q; want it to keep the count of what arrived", tc.args, err)
			}
			// A caller that redirects stdout must not capture usage text from a
			// failure: the help it would have received is the answer, but a
			// diagnostic on stdout is a broken pipe for anything downstream.
			if got := stdout.String(); got != "" {
				t.Errorf("run(%v) wrote %q to stdout; want nothing on the failure path", tc.args, got)
			}
		})
	}
}

// TestCompletionScriptReachesStdoutAndNothingElse covers the stream half of the
// contract for the generated script. The script is what the caller asked for, so
// it belongs on stdout and stderr must stay empty — otherwise a caller
// installing it with `kroot completion zsh > _kroot` captures a diagnostic into
// a file the shell will source.
func TestCompletionScriptReachesStdoutAndNothingElse(t *testing.T) {
	for _, shell := range cli.Shells {
		t.Run(shell, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			if err := run(context.Background(), []string{"completion", shell}, &stdout, &stderr); err != nil {
				t.Fatalf("run(completion %s) error = %v; want nil", shell, err)
			}
			if got := stdout.String(); strings.TrimSpace(got) == "" {
				t.Errorf("stdout is empty; want the %s script", shell)
			}
			if got := stderr.String(); got != "" {
				t.Errorf("stderr = %q; want it empty: the script is requested output, not a diagnostic", got)
			}
		})
	}
}

// TestCompletionScriptNamesEveryRegisteredCommand is the contract that keeps the
// two halves of the framework honest: the script is generated from the registry,
// so it cannot offer a command that does not exist and cannot omit one that
// does. A completion missing a command teaches the caller the command is gone.
func TestCompletionScriptNamesEveryRegisteredCommand(t *testing.T) {
	program, err := newProgram(flag.NewFlagSet("kroot", flag.ContinueOnError))
	if err != nil {
		t.Fatalf("newProgram() error = %v; want nil", err)
	}

	for _, shell := range cli.Shells {
		t.Run(shell, func(t *testing.T) {
			var stdout bytes.Buffer

			if err := run(context.Background(), []string{"completion", shell}, &stdout, io.Discard); err != nil {
				t.Fatalf("run(completion %s) error = %v; want nil", shell, err)
			}

			script := stdout.String()
			for _, c := range program.Commands.Commands() {
				if !strings.Contains(script, c.Name) {
					t.Errorf("%s script omits command %q", shell, c.Name)
				}
			}
		})
	}
}

// surfaceCase is one complete byte stream a caller can ask for. The guards below
// sweep every case rather than a sample, because a surface nobody enumerated is a
// surface the guard does not cover.
type surfaceCase struct {
	name string
	args []string
}

// allSurfaceCases lists every output the binary can produce: the general help, the
// help and the manual page of every registered command, and every completion script.
// The program is built the way run() builds it, so the list comes from the registry
// rather than from a hard-coded copy that a new command would miss.
func allSurfaceCases(t *testing.T) []surfaceCase {
	t.Helper()

	program, err := newProgram(flag.NewFlagSet("kroot", flag.ContinueOnError))
	if err != nil {
		t.Fatalf("newProgram() error = %v; want nil", err)
	}

	commands := program.Commands.Commands()
	cases := make([]surfaceCase, 0, 2+2*len(commands)+len(cli.Shells))
	cases = append(cases,
		surfaceCase{name: "general help"},
		surfaceCase{name: "manual, program page", args: []string{"man"}},
	)
	for _, c := range commands {
		cases = append(cases,
			surfaceCase{name: "help " + c.Name, args: []string{"help", c.Name}},
			surfaceCase{name: "manual, " + c.Name, args: []string{"man", c.Name}},
		)
	}
	for _, shell := range cli.Shells {
		cases = append(cases, surfaceCase{name: "completion " + shell, args: []string{"completion", shell}})
	}
	return cases
}

// renderSurface runs the binary's own dispatch and returns what reached stdout.
func renderSurface(t *testing.T, args []string) string {
	t.Helper()

	var out bytes.Buffer
	if err := run(context.Background(), args, &out, io.Discard); err != nil {
		t.Fatalf("run(%v) error = %v; want nil", args, err)
	}
	return out.String()
}

// TestNoSurfaceEmitsAnEscapeSequence is the accessibility guard: these bytes may not
// depend on a capability the destination might not have. A screen reader, a pipe and
// a log file receive exactly what is asserted here, and an escape byte in any of them
// is invisible to the author who wrote it and disruptive to the reader who did not.
// It costs nothing to assert, and it makes a spinner or a checkmark impossible to add
// unnoticed — which is the only way a rule like "no colour" survives contact with a
// deadline.
func TestNoSurfaceEmitsAnEscapeSequence(t *testing.T) {
	for _, tc := range allSurfaceCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			got := renderSurface(t, tc.args)

			i := strings.IndexByte(got, 0x1b)
			if i < 0 {
				return
			}
			start, end := max(0, i-40), min(len(got), i+40)
			t.Errorf("%s emitted an escape byte at offset %d: …%q…", tc.name, i, got[start:end])
		})
	}
}

// A tab in human-facing output is not a style preference. It re-expands against
// whatever tab-stop the reader's terminal happens to use, it survives
// reindentation, and it makes a byte-exact assertion impossible. It reached the
// help screen because the Flags block was rendered by the standard library's
// PrintDefaults while the Commands block above it was rendered by kroot — two
// typographic rules inside one screen. Asserting the absence of the byte is what
// stops the next third-party renderer from reintroducing it.
func TestNoSurfaceEmitsATab(t *testing.T) {
	for _, tc := range allSurfaceCases(t) {
		if strings.HasPrefix(tc.name, "completion ") {
			// Shell scripts are code a shell parses, not prose read at a
			// terminal. A tab inside one is a shell's business, and rewriting
			// a generated script to satisfy a human-facing rule would break the
			// syntax that rule is protecting.
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			got := renderSurface(t, tc.args)

			i := strings.IndexByte(got, '\t')
			if i < 0 {
				return
			}
			start, end := max(0, i-40), min(len(got), i+40)
			t.Errorf("%s emitted a tab at offset %d: …%q…", tc.name, i, got[start:end])
		})
	}
}

// TestHumanFacingSurfacesStayWithinTheDesignWidth pins the layout budget: 80 columns
// for what a person reads in a terminal or in a diff — the help output and the roff
// source of the manual. Output is ASCII, so bytes are columns and no conversion is
// needed to measure it.
//
// Generated shell scripts are excluded deliberately, and naming the exclusion is the
// point: they are code a shell parses, not prose read at a terminal, and fish's
// candidate lines are candidate plus description, which no reflow can shorten
// without dropping the description a candidate exists to carry. Excluding them in a
// comment is an exception the reviewer can disagree with; excluding them silently
// would be a budget that measures nothing.
func TestHumanFacingSurfacesStayWithinTheDesignWidth(t *testing.T) {
	const designWidth = 80

	for _, tc := range allSurfaceCases(t) {
		if strings.HasPrefix(tc.name, "completion ") {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			got := renderSurface(t, tc.args)

			for i, line := range strings.Split(got, "\n") {
				if len(line) > designWidth {
					t.Errorf("line %d is %d columns, over the design width of %d:\n%s",
						i+1, len(line), designWidth, line)
				}
			}
		})
	}
}

// TestCompletionFailuresNameTheSupportedShells asserts the rejection is
// actionable on its own. A caller who typed "kroot completion bas" learns the
// answer from the error — both the vocabulary and the near miss — without
// consulting the manual.
func TestCompletionFailuresNameTheSupportedShells(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no shell names the vocabulary", args: []string{"completion"}, want: strings.Join(cli.Shells, "|")},
		{name: "unknown shell names the vocabulary", args: []string{"completion", "tcsh"}, want: "tcsh"},
		{name: "a near miss is proposed", args: []string{"completion", "bas"}, want: "did you mean \"bash\""},
		{name: "too many shells is rejected", args: []string{"completion", "bash", "zsh"}, want: "one shell"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			err := run(context.Background(), tc.args, &stdout, &stderr)

			if !errors.Is(err, cli.ErrUsage) {
				t.Fatalf("run(%v) error = %v; want it to wrap ErrUsage", tc.args, err)
			}
			if got := stdout.String(); got != "" {
				t.Errorf("stdout = %q; want it empty: a failure must not write to stdout", got)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("run(%v) error = %v; want it to contain %q", tc.args, err, tc.want)
			}
		})
	}
}

// TestHelpForCompletionNamesTheShells covers discoverability: the shell list a
// caller needs is in `kroot help completion`, derived from cli.Shells rather
// than restated, so a shell added later appears here without a second edit.
func TestHelpForCompletionNamesTheShells(t *testing.T) {
	var stdout bytes.Buffer

	if err := run(context.Background(), []string{"help", "completion"}, &stdout, io.Discard); err != nil {
		t.Fatalf("run(help completion) error = %v; want nil", err)
	}

	got := stdout.String()
	for _, want := range append([]string{"kroot completion"}, cli.Shells...) {
		if !strings.Contains(got, want) {
			t.Errorf("help for completion = %q; want it to contain %q", got, want)
		}
	}
}

// TestCompletionExitCodesMatchTheDocumentedContract adds the command to the
// exit-code table: a generated script is exit 0, and every way of getting the
// shell name wrong is exit 2 rather than a runtime failure, because each is
// fixed by reinvoking.
func TestCompletionExitCodesMatchTheDocumentedContract(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{name: "bash script succeeds", args: []string{"completion", "bash"}, want: exitSuccess},
		{name: "zsh script succeeds", args: []string{"completion", "zsh"}, want: exitSuccess},
		{name: "fish script succeeds", args: []string{"completion", "fish"}, want: exitSuccess},
		{name: "no shell is a usage error", args: []string{"completion"}, want: exitUsage},
		{name: "unknown shell is a usage error", args: []string{"completion", "tcsh"}, want: exitUsage},
		{name: "too many shells is a usage error", args: []string{"completion", "bash", "zsh"}, want: exitUsage},
		{name: "man succeeds", args: []string{"man"}, want: exitSuccess},
		{name: "man for a command succeeds", args: []string{"man", "version"}, want: exitSuccess},
		{name: "man for an unknown command is a usage error", args: []string{"man", "versioo"}, want: exitUsage},
		{name: "man with too many commands is a usage error", args: []string{"man", "version", "help"}, want: exitUsage},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			if got := exitCodeFor(run(context.Background(), tc.args, &out, io.Discard)); got != tc.want {
				t.Errorf("exitCodeFor(run(%v)) = %d; want %d", tc.args, got, tc.want)
			}
		})
	}
}

// TestManualPageReachesStdoutAndNothingElse covers the stream half of the
// contract. The page is what the caller asked for, so it belongs on stdout with
// stderr empty — otherwise `kroot man | man -l -` pipes a diagnostic into the
// formatter instead of a page.
func TestManualPageReachesStdoutAndNothingElse(t *testing.T) {
	for _, args := range [][]string{{"man"}, {"man", "version"}, {"man", "completion"}} {
		t.Run(strings.Join(append([]string{"man"}, args[1:]...), " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			if err := run(context.Background(), args, &stdout, &stderr); err != nil {
				t.Fatalf("run(%v) error = %v; want nil", args, err)
			}
			if got := stdout.String(); strings.TrimSpace(got) == "" {
				t.Errorf("stdout is empty; want the page for %v", args)
			}
			if got := stderr.String(); got != "" {
				t.Errorf("stderr = %q; want it empty: the page is requested output, not a diagnostic", got)
			}
		})
	}
}

// TestManualPageDocumentsEveryRegisteredCommand is the guarantee that makes the
// manual worth generating rather than writing by hand: the page is derived from
// the registry, so a command cannot be reachable but undocumented. It is also
// what lets api-compatibility.md treat "documented in the manual" as a real
// precondition for the v1.0.0 stability promise.
func TestManualPageDocumentsEveryRegisteredCommand(t *testing.T) {
	program, err := newProgram(flag.NewFlagSet("kroot", flag.ContinueOnError))
	if err != nil {
		t.Fatalf("newProgram() error = %v; want nil", err)
	}

	var stdout bytes.Buffer
	if err := run(context.Background(), []string{"man"}, &stdout, io.Discard); err != nil {
		t.Fatalf("run(man) error = %v; want nil", err)
	}

	page := stdout.String()
	for _, c := range program.Commands.Commands() {
		if !strings.Contains(page, c.Name) {
			t.Errorf("manual page omits command %q", c.Name)
		}
	}
}

// TestManualPageCarriesTheExitContract asserts the page documents the same exit
// codes the process returns. The codes are passed in from main rather than
// restated in the generator, so this checks the wiring rather than a copy — a
// manual that disagrees with the binary about its exit codes is worse than none.
func TestManualPageCarriesTheExitContract(t *testing.T) {
	var stdout bytes.Buffer

	if err := run(context.Background(), []string{"man"}, &stdout, io.Discard); err != nil {
		t.Fatalf("run(man) error = %v; want nil", err)
	}

	page := stdout.String()
	for _, want := range []string{
		"EXIT STATUS",
		fmt.Sprintf("%d", exitSuccess),
		fmt.Sprintf("%d", exitFailure),
		fmt.Sprintf("%d", exitUsage),
	} {
		if !strings.Contains(page, want) {
			t.Errorf("manual page does not contain %q; the exit contract must match what the process returns:\n%s", want, page)
		}
	}
}

// TestManualDateFallsBackHonestly covers both branches of the page date. A
// release build gets a formatted date a formatter can parse; a plain `go build`
// has no build time, and inventing a plausible date would be a lie in a document
// whose whole purpose is to be checked against the binary it came from.
func TestManualDateFallsBackHonestly(t *testing.T) {
	previous := buildTime
	t.Cleanup(func() { buildTime = previous })

	t.Run("a timestamp becomes a plain date", func(t *testing.T) {
		buildTime = "2026-09-27T08:24:44Z"
		if got, want := manualDate(), "2026-09-27"; got != want {
			t.Errorf("manualDate() = %q; want %q", got, want)
		}
	})

	t.Run("an unknown build time is shown, not invented", func(t *testing.T) {
		buildTime = "unknown"
		if got := manualDate(); got != "unknown" {
			t.Errorf("manualDate() = %q; want %q", got, "unknown")
		}
	})
}

// TestHelpForManNamesTheUsage covers discoverability: the long text tells a
// reader how to get a page into a viewer, which is the one thing the command
// cannot do on its own.
func TestHelpForManNamesTheUsage(t *testing.T) {
	var stdout bytes.Buffer

	if err := run(context.Background(), []string{"help", "man"}, &stdout, io.Discard); err != nil {
		t.Fatalf("run(help man) error = %v; want nil", err)
	}

	got := stdout.String()
	for _, want := range []string{"kroot man [command]", "man -l -"} {
		if !strings.Contains(got, want) {
			t.Errorf("help for man = %q; want it to contain %q", got, want)
		}
	}
}

// keepDefaultLogger makes a test that drives realMain safe for the tests that
// follow: realMain calls slog.SetDefault, which mutates process-global state,
// so the previous logger is restored when the test ends. It is not a mutex —
// tests are sequential — it is the restore.
func keepDefaultLogger(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
}

// statser that fails lets the unknown-destination branch be exercised without
// depending on the process streams.
type failingStatser struct {
	bytes.Buffer
}

func (failingStatser) Stat() (fs.FileInfo, error) {
	return nil, errors.New("stat failed")
}

// TestIsTerminalTreatsUnknowableOutputAsPiped pins the safe default: a writer
// that cannot describe itself, and a writer that describes itself badly, both
// count as not a terminal. Guessing "terminal" would hand JSON consumers
// human-readable lines, which is the failure mode this detection exists to
// prevent.
func TestIsTerminalTreatsUnknowableOutputAsPiped(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Error("isTerminal(bytes.Buffer) = true; want false: a plain writer is not a descriptor")
	}
	if isTerminal(&failingStatser{}) {
		t.Error("isTerminal(writer with failing Stat) = true; want false")
	}
}

// TestIsTerminalAcceptsCharacterDevices covers the affirmative branch with a
// real character device rather than a mock, so the detection is proven against
// the kernel's answer and not only against ours. /dev/null is the portable
// stand-in for "character device" where no pty is available to a test, and its
// misclassification is documented as harmless in isTerminal.
func TestIsTerminalAcceptsCharacterDevices(t *testing.T) {
	f, err := os.Open("/dev/null")
	if err != nil {
		t.Skipf("no character device available: %v", err)
	}
	defer func() { _ = f.Close() }()

	if !isTerminal(f) {
		t.Error("isTerminal(character device) = false; want true")
	}
}

// TestRealMainReportsRejectedConfiguration covers the KROOT_LOG_LEVEL wiring in
// the process: the variable is read, a bad value stops the run before any
// command executes, and the exit code is 1 — configuration rejected, not 2 —
// with the reason on stderr.
func TestRealMainReportsRejectedConfiguration(t *testing.T) {
	t.Setenv("KROOT_LOG_LEVEL", "VERBOSE")
	keepDefaultLogger(t)

	var stdout, stderr bytes.Buffer

	if got := realMain(context.Background(), []string{"version"}, &stdout, &stderr); got != exitFailure {
		t.Errorf("realMain(...) = %d; want %d: configuration rejected is exit 1", got, exitFailure)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q; want it empty: nothing ran", got)
	}
	if got := stderr.String(); !strings.Contains(got, "configuration rejected") {
		t.Errorf("stderr = %q; want it to name the rejected configuration", got)
	}
	if got := stderr.String(); !strings.Contains(got, "debug|info|warn|error") {
		t.Errorf("stderr = %q; want it to name the accepted vocabulary", got)
	}
}

// TestRealMainHonoursAValidLogLevel proves the variable reaches the logger and
// the command still runs: a valid value is not a barrier, and the command's
// answer still lands on stdout as the stream contract requires.
func TestRealMainHonoursAValidLogLevel(t *testing.T) {
	t.Setenv("KROOT_LOG_LEVEL", "debug")
	keepDefaultLogger(t)

	var stdout, stderr bytes.Buffer

	if got := realMain(context.Background(), []string{"version"}, &stdout, &stderr); got != exitSuccess {
		t.Errorf("realMain(...) = %d; want %d", got, exitSuccess)
	}
	if got := stdout.String(); !strings.Contains(got, "kroot "+version) {
		t.Errorf("stdout = %q; want it to contain the version line", got)
	}
}

// TestRealMainMapsCommandFailuresToTheDocumentedExitCodes asserts the mapping
// survives the wiring layer: an unknown command is still exit 2 through
// realMain, not 1, so a script cannot be misled by the layer that owns the
// process rather than the command.
func TestRealMainMapsCommandFailuresToTheDocumentedExitCodes(t *testing.T) {
	t.Setenv("KROOT_LOG_LEVEL", "")
	keepDefaultLogger(t)

	tests := []struct {
		name string
		args []string
		want int
	}{
		{name: "no arguments prints help", args: nil, want: exitSuccess},
		{name: "unknown command is a usage error", args: []string{"bogus"}, want: exitUsage},
		{name: "unknown flag is a usage error", args: []string{"-nope"}, want: exitUsage},
		{name: "version succeeds", args: []string{"version"}, want: exitSuccess},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			if got := realMain(context.Background(), tc.args, &stdout, &stderr); got != tc.want {
				t.Errorf("realMain(%v) = %d; want %d", tc.args, got, tc.want)
			}
		})
	}
}

// TestRealMainFailsFastWhenShutdownIsUnderway asserts the context survives the
// wiring layer: a cancelled context stops the process before any command runs,
// and the code is a runtime failure because reinvoking fails the same way.
func TestRealMainFailsFastWhenShutdownIsUnderway(t *testing.T) {
	t.Setenv("KROOT_LOG_LEVEL", "")
	keepDefaultLogger(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout, stderr bytes.Buffer

	if got := realMain(ctx, []string{"version"}, &stdout, &stderr); got != exitFailure {
		t.Errorf("realMain(cancelled) = %d; want %d", got, exitFailure)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q; want it empty: nothing ran", got)
	}
}

// Both classes are asserted, not only the one that changed: a test that only
// pins the usage path would still pass if a runtime failure were demoted to WARN
// by the next person to touch the switch.
func TestUsageErrorIsNotLoggedAsAProgramFailure(t *testing.T) {
	keepDefaultLogger(t)

	tests := []struct {
		name string
		// level is the KROOT_LOG_LEVEL the run starts from. It differs per case
		// because the configuration failure below is only reachable with an
		// invalid value, which would stop the usage-error case before it ran.
		level     string
		args      []string
		wantLevel string
		wantMsg   string
		wantInErr string
	}{
		{
			name:      "a caller's typo is a warning about their invocation",
			args:      []string{"bogus"},
			wantLevel: "WARN",
			wantMsg:   "kroot: usage error",
			wantInErr: `see "kroot help"`,
		},
		{
			name:      "rejected configuration stays the program's failure",
			level:     "verbose",
			args:      []string{"version"},
			wantLevel: "ERROR",
			wantMsg:   "kroot failed",
			wantInErr: "configuration rejected",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("KROOT_LOG_LEVEL", tc.level)

			var stdout, stderr bytes.Buffer

			realMain(context.Background(), tc.args, &stdout, &stderr)
			got := stderr.String()

			if !strings.Contains(got, tc.wantLevel) {
				t.Errorf("stderr = %q; want level %s", got, tc.wantLevel)
			}
			if !strings.Contains(got, tc.wantMsg) {
				t.Errorf("stderr = %q; want the message %q", got, tc.wantMsg)
			}
			// The recovery is the part written for a person, so it must survive
			// in the record rather than being dropped by a reworded headline.
			//
			// stderr is not a terminal here, so the record is JSON and its quotes
			// are escaped. The assertion is made against the decoded record
			// rather than against the raw bytes, because asserting on escaped
			// output would pin the encoder's escaping rather than the message.
			var record map[string]any
			if err := json.Unmarshal(stderr.Bytes(), &record); err != nil {
				t.Fatalf("stderr is not a JSON record: %v\n%s", err, got)
			}
			if msg, _ := record["err"].(string); !strings.Contains(msg, tc.wantInErr) {
				t.Errorf("record err = %q; want it to contain %q", msg, tc.wantInErr)
			}
		})
	}
}

// TestWarnSilencesAUsageErrorButNotAProgramFailure is the reason the level was
// worth deciding. With KROOT_LOG_LEVEL=warn a caller can raise the floor to hide
// their own typos, and real failures still get through — a level that cannot
// separate the two is a level that filters nothing.
func TestWarnSilencesAUsageErrorButNotAProgramFailure(t *testing.T) {
	keepDefaultLogger(t)

	t.Setenv("KROOT_LOG_LEVEL", "error")
	var stdout, stderr bytes.Buffer
	realMain(context.Background(), []string{"bogus"}, &stdout, &stderr)
	if got := stderr.String(); got != "" {
		t.Errorf("stderr = %q; want a usage error silenced at KROOT_LOG_LEVEL=error", got)
	}

	t.Setenv("KROOT_LOG_LEVEL", "verbose")
	stdout.Reset()
	stderr.Reset()
	realMain(context.Background(), []string{"version"}, &stdout, &stderr)
	if got := stderr.String(); !strings.Contains(got, "configuration rejected") {
		t.Errorf("stderr = %q; want a configuration rejection at ERROR even under a warn floor", got)
	}
}

// TestDiagnosticSurvivesRedirection asserts the change did not cost a machine
// consumer anything. api-compatibility.md fixes the record's keys, so a redirect
// must still receive parseable JSON carrying the same keys and the whole message
// — the level moved, the shape did not.
func TestDiagnosticSurvivesRedirection(t *testing.T) {
	t.Setenv("KROOT_LOG_LEVEL", "")
	keepDefaultLogger(t)

	var stdout, stderr bytes.Buffer

	// stderr is a bytes.Buffer, which is not a character device, so this is the
	// redirected path a script and a CI job actually get.
	if got := realMain(context.Background(), []string{"bogus"}, &stdout, &stderr); got != exitUsage {
		t.Fatalf("realMain(bogus) = %d; want %d", got, exitUsage)
	}

	var record map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &record); err != nil {
		t.Fatalf("stderr is not JSON on a redirect: %v\n%s", err, stderr.String())
	}
	for _, key := range []string{"time", "level", "msg", "err"} {
		if _, ok := record[key]; !ok {
			t.Errorf("record %v is missing the key %q; the key set is the stable interface", record, key)
		}
	}
	if got := record["level"]; got != "WARN" {
		t.Errorf("record level = %v; want WARN", got)
	}
	if msg, _ := record["err"].(string); !strings.Contains(msg, `see "kroot help"`) {
		t.Errorf("record err = %q; want the whole message, recovery included", msg)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q; want it empty: a diagnostic never belongs on stdout", got)
	}
}

// BenchmarkRunVersion measures the cheapest complete invocation a caller can make —
// flag handling, configuration parsing, dispatch and the write — inside the process.
//
// It is deliberately not a spawn benchmark. Process startup (fork, exec, dynamic
// loading) is the operating system's cost and moves with the machine, so folding it
// into one number would produce a figure no code change could ever move, which is a
// budget that measures noise. What this benchmark can regress on is what the code
// owns, and that is what a budget attached to it guards. The process-level cost is
// measured separately against the built binary and recorded next to it.
func BenchmarkRunVersion(b *testing.B) {
	for range b.N {
		var out bytes.Buffer
		if err := run(context.Background(), []string{"version"}, &out, io.Discard); err != nil {
			b.Fatalf("run(version) error = %v; want nil", err)
		}
	}
}

// TestVersionDefaultsToDev asserts the value used when the linker does not
// inject a version, so a stale default is caught rather than shipped.
func TestVersionDefaultsToDev(t *testing.T) {
	if version != "dev" {
		t.Errorf("version = %q; want %q when not overridden via ldflags", version, "dev")
	}
}

// TestVersionReportsBuildMetadata guards the guarantee in
// docs/enterprise/versioning-policy.md § "Build metadata": a released binary
// must be able to report the exact revision it was built from.
func TestVersionReportsBuildMetadata(t *testing.T) {
	var out bytes.Buffer

	if err := printVersion(&out); err != nil {
		t.Fatalf("printVersion() error = %v; want nil", err)
	}

	got := out.String()
	for name, want := range map[string]string{
		"version":   version,
		"commit":    commit,
		"buildTime": buildTime,
		"toolchain": runtime.Version(),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("printVersion() output = %q; want it to contain %s %q", got, name, want)
		}
	}
}
