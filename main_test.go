package main

import (
	"bytes"
	"errors"
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

			err := run(tc.args, &out)

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
	err := run([]string{"version"}, failingWriter{})
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
	if err := run([]string{"help"}, failingWriter{}); err != nil {
		t.Errorf("run(help) with a failing writer = %v; want nil", err)
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
