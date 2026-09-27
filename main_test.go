package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
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
			name:    "unknown command returns error",
			args:    []string{"bogus"},
			wantErr: ErrUnknownCommand,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer

			err := run(tc.args, &out, io.Discard)

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
	err := run([]string{"version"}, failingWriter{}, io.Discard)

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
	if err := run([]string{"help"}, failingWriter{}, io.Discard); err != nil {
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

			if err := run([]string{arg}, &out, &errOut); err != nil {
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

			if got := exitCodeFor(run(tc.args, &out, io.Discard)); got != tc.want {
				t.Errorf("exitCodeFor(run(%v)) = %d; want %d", tc.args, got, tc.want)
			}
		})
	}
}

// TestRuntimeFailureIsNotAUsageError covers the other half of the contract: a
// failure the caller cannot fix by reinvoking must not be reported as exit 2.
// Here the invocation is correct and only the write fails.
func TestRuntimeFailureIsNotAUsageError(t *testing.T) {
	err := run([]string{"version"}, failingWriter{}, io.Discard)

	if errors.Is(err, ErrUsage) {
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

	err := run([]string{"-nope"}, &stdout, &stderr)
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
	} {
		if !strings.Contains(got, want) {
			t.Errorf("printVersion() output = %q; want it to contain %s %q", got, name, want)
		}
	}
}
