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

// It asserts the two facts a reader needs rather than a whole sentence, so the
// wording stays editable without the test becoming the reason it is not.
func TestGeneralHelpNamesTheNextStep(t *testing.T) {
	registry, err := New(
		Command{Name: "help", Summary: "print this help", Usage: "kroot help", Long: "Print help.", Run: noop()},
		Command{Name: "man", Summary: "print the manual", Run: noop()},
	)
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	program := &Program{
		Name:      "kroot",
		Summary:   "application skeleton",
		UsageLine: "kroot [flags] <command>",
		Commands:  registry,
	}

	var out strings.Builder
	if err := program.GeneralHelp(&out); err != nil {
		t.Fatalf("GeneralHelp() error = %v; want nil", err)
	}

	got := out.String()
	for _, want := range []string{`"kroot help <command>"`, `"man kroot"`} {
		if !strings.Contains(got, want) {
			t.Errorf("GeneralHelp() output = %q; want it to point at %s", got, want)
		}
	}
}

// TestGeneralHelpOmitsNextStepsItCannotName keeps the block honest in the other
// direction: a program without a man command must not advertise one. The rule is
// the same one the empty Flags section follows — no text over nothing.
func TestGeneralHelpOmitsNextStepsItCannotName(t *testing.T) {
	var out strings.Builder

	// testProgram registers help but not man, so the man pointer must be absent
	// while the help pointer remains.
	if err := testProgram(t, nil).GeneralHelp(&out); err != nil {
		t.Fatalf("GeneralHelp() error = %v; want nil", err)
	}

	got := out.String()
	if strings.Contains(got, "man kroot") {
		t.Errorf("GeneralHelp() output = %q; want no manual pointer when no man command exists", got)
	}
	if !strings.Contains(got, `"kroot help <command>"`) {
		t.Errorf("GeneralHelp() output = %q; want the help pointer, which does exist", got)
	}
}

// TestPrintFlagsUsesTheCommandColumnRule is the regression test for the tab. The
// Flags block used to be rendered by (*flag.FlagSet).PrintDefaults, which
// separates a name from its usage with a tab and a four-space hanging indent,
// while the Commands block directly above it used a computed column. The screen
// therefore carried two typographic rules at once.
//
// Two spaces is asserted, not one: measuring the column on the bare flag name
// instead of the rendered "-name" shortens the gap on every row, which is
// invisible while a single flag exists and appears the moment a longer one is
// added. That is why the width is computed from the rendered label.
func TestPrintFlagsUsesTheCommandColumnRule(t *testing.T) {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.Bool("version", false, "print the version and exit")
	fs.Bool("output", false, "choose an output format")

	var out strings.Builder
	if err := PrintFlags(&out, fs); err != nil {
		t.Fatalf("PrintFlags() error = %v; want nil", err)
	}

	got := out.String()
	if strings.ContainsRune(got, '\t') {
		t.Errorf("PrintFlags() = %q; want no tab in human-facing output", got)
	}
	if !strings.Contains(got, "  -version  print the version and exit\n") {
		t.Errorf("PrintFlags() = %q; want the longest label padded to a two-space gap", got)
	}
}

// TestPrintFlagsNormalisesMultilineUsage guards the column against authored prose.
// A usage string containing a newline would wrap onto an unindented second line
// and break the alignment of every row after it, which is the defect this
// renderer exists to remove.
func TestPrintFlagsNormalisesMultilineUsage(t *testing.T) {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.String("mode", "", "choose a mode\nand a format")

	var out strings.Builder
	if err := PrintFlags(&out, fs); err != nil {
		t.Fatalf("PrintFlags() error = %v; want nil", err)
	}

	got := out.String()
	if lines := strings.Count(strings.TrimSuffix(got, "\n"), "\n"); lines != 0 {
		t.Errorf("PrintFlags() = %q; want a single row, got %d lines", got, lines+1)
	}
	if !strings.Contains(got, "-mode  choose a mode and a format\n") {
		t.Errorf("PrintFlags() = %q; want the newline folded into a space", got)
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

// TestCommandListStaysWithinTheDesignWidth renders the help for a command set
// built to the edge of the budget and measures the bytes that come out.
//
// The registry already rejects a wider set, and this checks the half it cannot:
// that the number New enforces and the layout writeCommandList produces are the
// same number. If the padding or the formula changes on one side only, the
// rendered line is what notices — which is the whole argument for measuring
// output rather than trusting two places that happen to agree today.
func TestCommandListStaysWithinTheDesignWidth(t *testing.T) {
	// Sized to the budget by arithmetic rather than written out, so the test
	// cannot rot into checking a set that never reached the boundary.
	summary := "print the completion script for one shell"
	name := strings.Repeat("c", designWidth-4-len(summary))

	registry, err := New(
		Command{Name: name, Summary: summary, Run: noop()},
		Command{Name: "version", Summary: "print the version", Run: noop()},
	)
	if err != nil {
		t.Fatalf("New() error = %v; want a set at the budget to be accepted", err)
	}

	program := &Program{
		Name:      "kroot",
		Summary:   "an application",
		UsageLine: "kroot [flags] <command>",
		Commands:  registry,
	}
	var out strings.Builder
	if err := program.GeneralHelp(&out); err != nil {
		t.Fatalf("GeneralHelp() error = %v; want nil", err)
	}

	rendered := out.String()
	for _, line := range strings.Split(rendered, "\n") {
		if len(line) > designWidth {
			t.Errorf("a rendered line is %d columns, over the design width of %d:\n%s",
				len(line), designWidth, line)
		}
	}
	// The widest line has to reach the budget, or the loop above would pass on a
	// set that never came close to it.
	if !strings.Contains(rendered, summary) {
		t.Errorf("the rendered list does not contain the summary it was built for:\n%s", rendered)
	}
}

// TestGeneralHelpReportsAFailureAtEveryWrite walks the destination's failure
// through every write a full help output takes, and requires an error each time.
//
// The other write-failure test fails the very first write, which proves the
// header is checked and nothing else. Help writes in pieces — a header, one line
// per command, the flags heading, the flag list — and each of those is a place
// where a truncated help could be reported as success. The manual's generator
// already walks its own writes the same way; this is the same obligation on the
// other renderer, and the same bug it catches: a stage that swallows its failure
// and lets the caller believe it received a whole document.
func TestGeneralHelpReportsAFailureAtEveryWrite(t *testing.T) {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.Bool("version", false, "print the version and exit")
	program := testProgram(t, fs)

	// Count the writes a successful rendering makes, so the walk below covers
	// every stage rather than a guessed number of them.
	var counter countWriter
	if err := program.GeneralHelp(&counter); err != nil {
		t.Fatalf("GeneralHelp() error = %v; want nil", err)
	}
	if counter.writes == 0 {
		t.Fatal("GeneralHelp() wrote nothing; the walk below would prove nothing")
	}

	for n := range counter.writes {
		err := program.GeneralHelp(&failAfterWriter{remaining: n})
		if err == nil {
			t.Errorf("GeneralHelp() = nil when the destination failed on write %d of %d: a truncated help must be reported",
				n+1, counter.writes)
			continue
		}
		if !strings.Contains(err.Error(), "write failed") {
			t.Errorf("GeneralHelp() error = %v; want it to wrap the underlying failure", err)
		}
	}
}

// countWriter is shared with the manual's tests: both renderers write in
// pieces, and both are walked write by write to prove a truncated document is
// reported rather than returned as success.

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
