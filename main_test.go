package main

import (
	"bytes"
	"context"
	"errors"
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
