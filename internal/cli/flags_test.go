package cli

import (
	"flag"
	"io"
	"strings"
	"testing"
)

// TestPrintFlagsRendersTheList is the direct regression test for the defect:
// the flag set's output is io.Discard in the real CLI, so a renderer that does
// not redirect would produce an empty list and help would end at a bare
// "Flags:" heading.
func TestPrintFlagsRendersTheList(t *testing.T) {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Bool("version", false, "print the version and exit")

	var out strings.Builder
	if err := PrintFlags(&out, fs); err != nil {
		t.Fatalf("PrintFlags() error = %v; want nil", err)
	}

	got := out.String()
	for _, want := range []string{"-version", "print the version and exit"} {
		if !strings.Contains(got, want) {
			t.Errorf("PrintFlags() = %q; want it to contain %q", got, want)
		}
	}
}

// TestPrintFlagsLeavesTheFlagSetOutputAlone proves the helper is a view, not a
// mutation: a flag set used again after rendering must still send its own
// chatter where its owner put it.
func TestPrintFlagsLeavesTheFlagSetOutputAlone(t *testing.T) {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	original := &strings.Builder{}
	fs.SetOutput(original)

	var out strings.Builder
	if err := PrintFlags(&out, fs); err != nil {
		t.Fatalf("PrintFlags() error = %v; want nil", err)
	}

	if fs.Output() != original {
		t.Error("PrintFlags() left the flag set pointing at a different writer")
	}
}

// TestPrintFlagsReportsWriteFailures covers the reason the helper returns an
// error at all: PrintDefaults reports nothing, so without the recorder a broken
// destination would be indistinguishable from a printed list.
func TestPrintFlagsReportsWriteFailures(t *testing.T) {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.Bool("version", false, "print the version and exit")

	if err := PrintFlags(failingWriter{}, fs); err == nil {
		t.Error("PrintFlags(failingWriter) = nil; want a write failure")
	}
}
