package cli

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

// failingWriter fails every write, so a rendering path can be shown to report
// its failure rather than to pretend it succeeded.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

// testProgram builds a program with the given global flags, so help rendering
// can be asserted against a real flag set rather than a stand-in for one.
func testProgram(t *testing.T, flags *flag.FlagSet) *Program {
	t.Helper()

	registry, err := New(
		Command{Name: "help", Summary: "print this help", Usage: "kroot help", Long: "Print help.", Run: noop()},
		Command{Name: "version", Summary: "print the version", Run: noop()},
	)
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	return &Program{
		Name:      "kroot",
		Summary:   "application skeleton",
		UsageLine: "kroot [flags] <command>",
		Commands:  registry,
		FlagSet:   flags,
	}
}

// TestGeneralHelpListsCommandsSortedAndAligned pins what a caller reads first:
// the commands appear in sorted order with their summaries aligned, and the
// alignment is computed rather than hard-coded.
func TestGeneralHelpListsCommandsSortedAndAligned(t *testing.T) {
	var out strings.Builder

	if err := testProgram(t, nil).GeneralHelp(&out); err != nil {
		t.Fatalf("GeneralHelp() error = %v; want nil", err)
	}

	got := out.String()
	for _, want := range []string{
		"kroot - application skeleton",
		"Usage:\n  kroot [flags] <command>",
		"Commands:\n",
		"  help     print this help\n",
		"  version  print the version\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("GeneralHelp() output = %q; want it to contain %q", got, want)
		}
	}

	if strings.Index(got, "  help ") > strings.Index(got, "  version ") {
		t.Errorf("GeneralHelp() listed version before help; want sorted order:\n%s", got)
	}
}

// TestGeneralHelpRendersTheFlagsSection is the regression test for the defect
// this package was created with: the binary printed a "Flags:" heading and
// nothing under it, because flag's own output was pointed at io.Discard.
func TestGeneralHelpRendersTheFlagsSection(t *testing.T) {
	var out strings.Builder

	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.Bool("version", false, "print the version and exit")

	if err := testProgram(t, fs).GeneralHelp(&out); err != nil {
		t.Fatalf("GeneralHelp() error = %v; want nil", err)
	}

	got := out.String()
	heading := strings.Index(got, "Flags:\n")
	if heading < 0 {
		t.Fatalf("GeneralHelp() output = %q; want a Flags section", got)
	}
	if !strings.Contains(got[heading:], "-version") {
		t.Errorf("GeneralHelp() output = %q; want the flag list after the heading, not an empty section", got)
	}
}

// TestGeneralHelpOmitsAnEmptyFlagsSection keeps help honest: with nothing to
// declare, it does not print a heading that promises content.
func TestGeneralHelpOmitsAnEmptyFlagsSection(t *testing.T) {
	var out strings.Builder

	if err := testProgram(t, nil).GeneralHelp(&out); err != nil {
		t.Fatalf("GeneralHelp() error = %v; want nil", err)
	}

	if strings.Contains(out.String(), "Flags:") {
		t.Errorf("GeneralHelp() output = %q; want no Flags section when there are no flags", out.String())
	}
}

// TestCommandHelpPrefersLongAndFallsBack proves both halves of the contract:
// the detailed text wins when present, and the summary is used when it is not.
func TestCommandHelpPrefersLongAndFallsBack(t *testing.T) {
	program := testProgram(t, nil)

	helpCommand, ok := program.Commands.Lookup("help")
	if !ok {
		t.Fatal("Lookup(help) reported absent; want present")
	}
	versionCommand, ok := program.Commands.Lookup("version")
	if !ok {
		t.Fatal("Lookup(version) reported absent; want present")
	}

	var detailed strings.Builder
	if err := program.CommandHelp(&detailed, helpCommand); err != nil {
		t.Fatalf("CommandHelp(help) error = %v; want nil", err)
	}
	for _, want := range []string{"Usage:\n  kroot help", "Print help."} {
		if !strings.Contains(detailed.String(), want) {
			t.Errorf("CommandHelp(help) = %q; want it to contain %q", detailed.String(), want)
		}
	}

	var summaryOnly strings.Builder
	if err := program.CommandHelp(&summaryOnly, versionCommand); err != nil {
		t.Fatalf("CommandHelp(version) error = %v; want nil", err)
	}
	if want := "Usage:\n  kroot version\n\nprint the version\n"; summaryOnly.String() != want {
		t.Errorf("CommandHelp(version) = %q; want %q: an absent Usage and Long fall back to the program name and the summary", summaryOnly.String(), want)
	}
}

// TestUnknownCommandNamesTheCommandListWhenNothingIsClose covers the one
// rejection a caller cannot act on by themselves. A near miss is answered with
// the command it meant, which is a recovery in itself; a name that resembles
// nothing would otherwise leave the caller holding a rejection with no next step
// at all, so the command list is named instead. The suggestion suppresses the
// pointer on purpose — two hints in one line bury the one that matters.
func TestUnknownCommandNamesTheCommandListWhenNothingIsClose(t *testing.T) {
	program := testProgram(t, nil)

	near := program.UnknownCommand("versioo")
	if !errors.Is(near, ErrUnknownCommand) {
		t.Errorf("UnknownCommand(versioo) = %v; want it to wrap ErrUnknownCommand", near)
	}
	if want := `did you mean "version"?`; !strings.Contains(near.Error(), want) {
		t.Errorf("UnknownCommand(versioo) = %q; want it to contain %q", near, want)
	}
	if strings.Contains(near.Error(), "command list") {
		t.Errorf("UnknownCommand(versioo) = %q; want no pointer to the command list when the nearest command is already named", near)
	}

	far := program.UnknownCommand("kubernetes")
	if !errors.Is(far, ErrUsage) {
		t.Errorf("UnknownCommand(kubernetes) = %v; want it to wrap ErrUsage", far)
	}
	if want := `"kroot help"`; !strings.Contains(far.Error(), want) {
		t.Errorf("UnknownCommand(kubernetes) = %q; want it to contain %q so the caller has a next step", far, want)
	}
}

// TestRenderingReportsWriteFailures proves help does not claim success when the
// destination refused the text; the caller decides what a broken pipe means.
func TestRenderingReportsWriteFailures(t *testing.T) {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.Bool("version", false, "print the version and exit")

	program := testProgram(t, fs)

	if err := program.GeneralHelp(failingWriter{}); err == nil {
		t.Error("GeneralHelp(failingWriter) = nil; want a write failure")
	}

	command, ok := program.Commands.Lookup("help")
	if !ok {
		t.Fatal("Lookup(help) reported absent; want present")
	}
	if err := program.CommandHelp(failingWriter{}, command); err == nil {
		t.Error("CommandHelp(failingWriter) = nil; want a write failure")
	}

	if err := program.GeneralHelp(&strings.Builder{}); err != nil {
		t.Errorf("GeneralHelp(strings.Builder) error = %v; want nil", err)
	}
}
