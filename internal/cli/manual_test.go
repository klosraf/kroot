package cli

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// maxRoffColumn is the source width a roff file is expected to stay under.
// mandoc reports a longer line as a STYLE warning, and a generated file that
// trips the linter its own ecosystem uses is a file people stop trusting.
const maxRoffColumn = 80

// testInfo is provenance with a real, parseable date, so a page under test
// renders the way a released binary's would rather than through the "unknown"
// fallback that a plain `go build` produces.
func testInfo() ManualInfo {
	return ManualInfo{
		Version: "1.2.3",
		Date:    "2026-09-27",
		ExitCodes: []ExitCode{
			{Code: 0, Name: "success", Meaning: "the invocation did what it was asked to do"},
			{Code: 2, Name: "usage error", Meaning: "unknown command or bad argument"},
		},
	}
}

// manualProgram builds a program with a representative surface: a flag set, a
// command with a long description, and commands that exercise every rendering
// branch.
func manualProgram(t *testing.T) *Program {
	t.Helper()

	registry, err := New(
		Command{Name: "help", Summary: "print this help", Usage: "kroot help [command]", Long: "Print the command list.", Run: noop()},
		Command{Name: "version", Summary: "print the version", Run: noop()},
		Command{Name: "run", Summary: "run the thing", Run: noop()},
	)
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.Bool("version", false, "print the version and exit")

	return &Program{
		Name:      "kroot",
		Summary:   "an application",
		UsageLine: "kroot [flags] <command>",
		Commands:  registry,
		FlagSet:   fs,
	}
}

// renderManual generates a page into a buffer.
func renderManual(t *testing.T, p *Program, command string, info ManualInfo) string {
	t.Helper()

	var buf bytes.Buffer
	if err := WriteManual(&buf, p, command, info); err != nil {
		t.Fatalf("WriteManual(%q) error = %v; want nil", command, err)
	}
	return buf.String()
}

// TestProgramPageDocumentsTheWholeSurface is the base table for the program page.
// Every command, the flag and the exit contract must appear: a manual missing a
// command that help lists is worse than no manual, because it reads as
// authoritative.
func TestProgramPageDocumentsTheWholeSurface(t *testing.T) {
	got := renderManual(t, manualProgram(t), "", testInfo())

	for _, want := range []string{
		`.TH "KROOT" "1" "2026-09-27" "kroot 1.2.3" "kroot Manual"`,
		".SH",
		// The hyphen is escaped in the source: escapeRoff writes \- so the
		// formatter renders a hyphen rather than a minus.
		`kroot \- an application`,
		"kroot [flags] <command>",
		"kroot help [command]",
		"print the version and exit",
		"run the thing",
		"0",
		"usage error",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("program page does not contain %q:\n%s", want, got)
		}
	}

	// A command with no Usage falls back to the program name and its own name,
	// so the page never shows a bare name with no way to invoke it.
	if !strings.Contains(got, "kroot version") {
		t.Errorf("program page does not fall back to a synthesised usage line:\n%s", got)
	}
}

// TestCommandPageIsNamedAfterItsCommand covers the man naming convention:
// `kroot completion` documents as kroot-completion(1), so a reader who types
// `man kroot-completion` finds the page they expect.
func TestCommandPageIsNamedAfterItsCommand(t *testing.T) {
	tests := []struct {
		command string
		wantTH  string
	}{
		{command: "help", wantTH: `.TH "KROOT-HELP" "1"`},
		{command: "version", wantTH: `.TH "KROOT-VERSION" "1"`},
		{command: "run", wantTH: `.TH "KROOT-RUN" "1"`},
	}

	for _, tc := range tests {
		t.Run(tc.command, func(t *testing.T) {
			got := renderManual(t, manualProgram(t), tc.command, testInfo())

			if !strings.Contains(got, tc.wantTH) {
				t.Errorf("page for %q does not start with %s:\n%s", tc.command, tc.wantTH, got)
			}
			if !strings.Contains(got, ".SH "+`"DESCRIPTION"`) {
				t.Errorf("page for %q has no DESCRIPTION section:\n%s", tc.command, got)
			}
		})
	}
}

// TestCommandPageFallsBackToTheSummary proves a command with no long description
// still gets a DESCRIPTION rather than an empty section, matching what
// CommandHelp does for `kroot help <cmd>`.
func TestCommandPageFallsBackToTheSummary(t *testing.T) {
	got := renderManual(t, manualProgram(t), "run", testInfo())

	if !strings.Contains(got, "run the thing") {
		t.Errorf("page for run does not fall back to the summary:\n%s", got)
	}
}

// TestWriteManualRejectsAnUnknownCommand reuses the registry's own rejection, so
// a typo in a manual request is answered exactly like a typo anywhere else —
// including the suggestion.
func TestWriteManualRejectsAnUnknownCommand(t *testing.T) {
	var buf bytes.Buffer

	err := WriteManual(&buf, manualProgram(t), "hlep", testInfo())

	if !errors.Is(err, ErrUsage) {
		t.Fatalf("WriteManual(hlep) error = %v; want it to wrap ErrUsage", err)
	}
	if !errors.Is(err, ErrUnknownCommand) {
		t.Errorf("WriteManual(hlep) error = %v; want it to wrap ErrUnknownCommand", err)
	}
	if !strings.Contains(err.Error(), `did you mean "help"`) {
		t.Errorf("WriteManual(hlep) error = %v; want it to suggest the nearest command", err)
	}
	if got := buf.String(); got != "" {
		t.Errorf("WriteManual(hlep) wrote %q; want nothing: a rejection is not output", got)
	}
}

// TestManualStaysWithinTheRoffColumnLimit turns the convention into something
// enforced. The generator does not reflow authored prose — roff fills paragraphs
// at render time, and rewrapping here would fight the author — so the width of
// the page is a property of the text that was written. A test is what catches the
// next long line, and it says which one.
func TestManualStaysWithinTheRoffColumnLimit(t *testing.T) {
	p := manualProgram(t)

	commands := append([]string{""}, p.Commands.Names()...)
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			page := renderManual(t, p, command, testInfo())

			for i, line := range strings.Split(page, "\n") {
				if len(line) > maxRoffColumn {
					t.Errorf("page line %d is %d columns, over the %d limit: %q", i+1, len(line), maxRoffColumn, line)
				}
			}
		})
	}
}

// TestEscapeRoffNeutralisesEverythingThatWouldChangeThePage is the escaping
// test, and it is a correctness test rather than a tidy-up one. Every case here
// is a way free text can take over a roff file: a backslash starts an escape, a
// leading dot or apostrophe makes a control line, and an unescaped hyphen is the
// wrong glyph. A summary or description is author-supplied text, so it has to
// survive being typeset.
func TestEscapeRoffNeutralisesEverythingThatWouldChangeThePage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain text is untouched", in: "print the version", want: "print the version"},
		{name: "backslash becomes an escaped backslash", in: `a\b`, want: `a\eb`},
		{name: "hyphen becomes a real hyphen", in: "kroot-completion", want: `kroot\-completion`},
		{name: "leading dot is defused", in: ".SH NAME", want: `\&.SH NAME`},
		{name: "leading apostrophe is defused", in: "' quoted", want: `\&' quoted`},
		{name: "a dot mid-line needs no defusing", in: "version 1.2.3", want: "version 1.2.3"},
		{name: "a tab becomes a space, which roff can actually typeset", in: "a\tb", want: "a b"},
		{name: "a second line is treated as a new line", in: "ok\n.SH NAME", want: "ok\n" + `\&.SH NAME`},
		{name: "a dot after leading blanks is still a control line", in: "  .SH NAME", want: "  " + `\&.SH NAME`},
		{name: "empty input is empty", in: "", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := escapeRoff(tc.in); got != tc.want {
				t.Errorf("escapeRoff(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestEscapeRoffIsIdempotentUnderReescaping guards the interaction that makes
// this a single pass rather than three replacements. Escaping an already-escaped
// string must not be a fixed point, but escaping twice must never produce a
// *valid* roff construct out of text that was not one: a doubled `\&` is inert,
// whereas a lone one is what defuses a control line.
func TestEscapeRoffDoesNotCreateAControlLine(t *testing.T) {
	// A summary that is nothing but a macro call. Escaped, it must not be able to
	// become a heading, no matter how many times it passes through.
	hostile := ".SH EVIL"

	once := escapeRoff(hostile)
	if strings.HasPrefix(once, ".") {
		t.Errorf("escapeRoff(%q) = %q; want it not to start with a dot", hostile, once)
	}

	twice := escapeRoff(once)
	if strings.HasPrefix(twice, ".") {
		t.Errorf("escapeRoff(escapeRoff(%q)) = %q; want it not to start with a dot", hostile, twice)
	}
}

// TestQuoteRoffProtectsMacroArguments covers the other half of the escaping
// problem. A macro argument containing a space is read as several arguments, so
// the footers "kroot 1.2.3" and "kroot Manual" would otherwise truncate at the
// space and the remainder be discarded as excess — which is exactly the defect
// mandoc reported on the first draft of this generator.
func TestQuoteRoffProtectsMacroArguments(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "a space is protected", in: "kroot 1.2.3", want: `"kroot 1.2.3"`},
		{name: "a quote is doubled", in: `say "hi"`, want: `"say ""hi"""`},
		{name: "a backslash is escaped", in: `a\b`, want: `"a\eb"`},
		{name: "hyphens are left literal so dates stay parseable", in: "2026-09-27", want: `"2026-09-27"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := quoteRoff(tc.in); got != tc.want {
				t.Errorf("quoteRoff(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestManualSurvivesHostileText proves the escaping end to end, by rendering a
// page whose command names and text are chosen to break it, and reading the
// result back out of a real roff formatter. Assertions on the generated string
// can only show that the text looks escaped; only the formatter can show that
// the page still means what it said.
func TestManualSurvivesHostileText(t *testing.T) {
	requireTool(t, "mandoc")

	hostile := "it's a 'trap'; $(id) \\ `x` \"double\" -dash\n.SH INJECTED\n\ttabbed .dot"

	registry, err := New(
		Command{Name: "help", Summary: hostile, Run: noop()},
	)
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	p := &Program{Name: "kroot", Summary: hostile, UsageLine: "kroot", Commands: registry}

	var page bytes.Buffer
	if err := WriteManual(&page, p, "", testInfo()); err != nil {
		t.Fatalf("WriteManual() error = %v; want nil", err)
	}

	// Lint rejects a page whose macros are malformed, which is the failure a
	// broken escape produces.
	path := writePage(t, page.String())
	if out, err := runMandoc(t, "-T", "lint", path); err != nil {
		t.Fatalf("mandoc rejected the page: %v\n%s\npage:\n%s", err, out, page.String())
	}

	// Read the page back. If the escaping failed, ".SH INJECTED" became a real
	// section, and mandoc's parse tree shows every section it recognised — so the
	// section count is the assertion. Checking only that the text survives would
	// pass whether it was typeset as prose or promoted to a heading, which is the
	// whole question.
	tree, err := runMandoc(t, "-T", "tree", path)
	if err != nil {
		t.Fatalf("mandoc could not parse the page: %v\n%s", err, tree)
	}

	const wantSections = 6 // NAME, SYNOPSIS, DESCRIPTION, COMMANDS, EXIT STATUS, SEE ALSO
	if got := strings.Count(tree, "SH (head)"); got != wantSections {
		t.Errorf("mandoc found %d sections; want %d: the injected .SH was honoured as a real section\n%s", got, wantSections, tree)
	}

	// The text must also still be there: escaping that dropped content would be as
	// wrong as escaping that promoted it.
	if !strings.Contains(tree, "INJECTED") {
		t.Errorf("the injected text is missing from the parse; escaping dropped content:\n%s", tree)
	}
}

// TestGeneratedPagesPassARealRoffLinter hands the pages to mandoc, which is what
// macOS's own man uses, and asks it to render them. Linting proves the macros are
// well formed; rendering proves a reader gets a page back rather than an error.
//
// mandoc is not present on every machine, so the test skips visibly instead of
// assuming — the structural tests above run regardless.
func TestGeneratedPagesPassARealRoffLinter(t *testing.T) {
	requireTool(t, "mandoc")

	p := manualProgram(t)
	info := testInfo()

	commands := append([]string{""}, p.Commands.Names()...)
	for _, command := range commands {
		name := command
		if name == "" {
			name = "program"
		}
		t.Run(name, func(t *testing.T) {
			page := renderManual(t, p, command, info)
			path := writePage(t, page)

			if out, err := runMandoc(t, "-T", "lint", path); err != nil {
				t.Fatalf("mandoc -T lint rejected the page for %q: %v\n%s\npage:\n%s", command, err, out, page)
			}

			tree, err := runMandoc(t, "-T", "tree", path)
			if err != nil {
				t.Fatalf("mandoc could not parse the page for %q: %v\n%s", command, err, tree)
			}
			// The parse tree is the assertion surface rather than rendered text:
			// a formatter overstrikes headings, so "SEE ALSO" arrives as
			// "S EE E A L LS O", and it fills paragraphs at a width the author
			// never chose. The tree reports what was understood, which is what
			// correctness of a generated page actually means.
			if !strings.Contains(tree, "NAME") || !strings.Contains(tree, "SYNOPSIS") {
				t.Errorf("mandoc did not recognise the page structure for %q:\n%s", command, tree)
			}
		})
	}
}

// TestProgramPageIsUnderstoodByARealRoffFormatter reads the page back out of
// mandoc and checks it says what the binary does, and that the provenance
// survived. Every other test inspects the generated source; this one inspects
// what a formatter made of it, which is the only form the guarantee is about.
//
// The parsed date is asserted separately because it is the field a formatter
// either understands or rejects: an escaped hyphen in "2026-09-27" renders as a
// perfectly good-looking date that no formatter can parse, which is a defect
// visible only here.
func TestProgramPageIsUnderstoodByARealRoffFormatter(t *testing.T) {
	requireTool(t, "mandoc")

	page := renderManual(t, manualProgram(t), "", testInfo())
	tree, err := runMandoc(t, "-T", "tree", writePage(t, page))
	if err != nil {
		t.Fatalf("mandoc could not parse the page: %v\n%s", err, tree)
	}

	for _, want := range []string{
		// The tree reports the source line, so the escaped hyphen is visible
		// here. That it renders as a real hyphen rather than a minus is the
		// formatter's business, and is what the \- is for.
		`kroot \- an application`, // NAME
		"kroot help [command]",    // a command's usage
		"print the version",       // a command's summary
		"-version",                // an option
		"usage error",             // the exit contract
		"SEE ALSO",
	} {
		if !strings.Contains(tree, want) {
			t.Errorf("parsed page does not contain %q:\n%s", want, tree)
		}
	}

	// The .TH fields are echoed back parsed, so this proves the date is a date.
	if !strings.Contains(tree, `date  = "2026-09-27"`) {
		t.Errorf("mandoc did not parse the page date; an escaped hyphen would do this:\n%s", tree)
	}
	if !strings.Contains(tree, `title = "KROOT"`) {
		t.Errorf("parsed page has no title; the .TH line is malformed:\n%s", tree)
	}
}

// countWriter counts the writes a successful generation makes, so a test can
// learn how many stages there are to fail.
type countWriter struct {
	writes int
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.writes++
	return len(p), nil
}

// failAfterWriter succeeds for its first n writes and fails on every write after.
// Failing the very first write only ever exercises one error path; failing a
// chosen one reaches the branch belonging to that stage of the generator.
type failAfterWriter struct {
	remaining int
}

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, errors.New("write failed")
	}
	w.remaining--
	return len(p), nil
}

// TestWriteManualReportsAFailureAtEveryStage walks the destination's failure
// through every write a full page takes, and requires an error each time.
//
// A generator that writes a page in a dozen pieces has a dozen places to get
// wrong, and the bug this catches is the quiet one: a stage that swallows its
// write error returns success, so a caller redirecting to a disk that filled up
// is told the install worked and is left with half a man page. Failing only the
// first write would have passed while eleven of those paths went unproven.
func TestWriteManualReportsAFailureAtEveryStage(t *testing.T) {
	p := manualProgram(t)
	info := testInfo()

	counter := &countWriter{}
	if err := WriteManual(counter, p, "", info); err != nil {
		t.Fatalf("WriteManual() error = %v; want nil", err)
	}
	if counter.writes == 0 {
		t.Fatal("WriteManual() wrote nothing; the stage walk below would prove nothing")
	}

	// Every command page too, since a shorter page exercises a different subset
	// of the same stages.
	for _, command := range append([]string{""}, p.Commands.Names()...) {
		name := command
		if name == "" {
			name = "program"
		}
		t.Run(name, func(t *testing.T) {
			count := &countWriter{}
			if err := WriteManual(count, p, command, info); err != nil {
				t.Fatalf("WriteManual(%q) error = %v; want nil", command, err)
			}

			// remaining = n lets writes 1..n through and fails write n+1, so the
			// range covers a failure at every write the page actually makes.
			for n := range count.writes {
				w := &failAfterWriter{remaining: n}

				err := WriteManual(w, p, command, info)

				if err == nil {
					t.Fatalf("WriteManual(%q) = nil when the destination failed on write %d of %d; a truncated page must be reported",
						command, n+1, count.writes)
				}
				if !strings.Contains(err.Error(), "write failed") {
					t.Errorf("WriteManual(%q) error = %v; want it to wrap the underlying failure", command, err)
				}
			}
		})
	}
}

// TestManualOmitsSectionsItHasNothingFor keeps the page honest: a program with no
// global flags gets no OPTIONS heading, and provenance with no exit contract
// gets no EXIT STATUS. A heading that promises nothing is the same defect help
// was fixed for.
func TestManualOmitsSectionsItHasNothingFor(t *testing.T) {
	registry, err := New(Command{Name: "help", Summary: "print this help", Run: noop()})
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	bare := &Program{Name: "kroot", Summary: "an application", UsageLine: "kroot", Commands: registry}

	var buf bytes.Buffer
	if err := WriteManual(&buf, bare, "", ManualInfo{Version: "dev", Date: "2026-09-27"}); err != nil {
		t.Fatalf("WriteManual() error = %v; want nil", err)
	}

	got := buf.String()
	for _, unwanted := range []string{"OPTIONS", "EXIT STATUS"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("page contains a %s section with nothing in it:\n%s", unwanted, got)
		}
	}
	// The sections that do have content must still be there.
	for _, want := range []string{"COMMANDS", "SEE ALSO"} {
		if !strings.Contains(got, want) {
			t.Errorf("page is missing the %s section:\n%s", want, got)
		}
	}
}

// runMandoc runs mandoc over a page and returns its combined output.
//
// mandoc is the formatter macOS's own man uses, so asking it to lint and render
// is the only way to know the generated macros are well formed rather than merely
// plausible.
func runMandoc(t *testing.T, args ...string) (string, error) {
	t.Helper()

	//nolint:gosec // G204: mandoc is a constant and the only variable argument is
	// a path this test created under t.TempDir(). Nothing a caller controls
	// reaches the command line, which is the condition G204 checks.
	out, err := exec.Command("mandoc", args...).CombinedOutput()
	return string(out), err
}

// writePage puts a generated page on disk for a roff tool to read, and returns
// its path. A file is used rather than stdin because that is how a page is
// installed, and because some tools treat "-" as their own stdin.
func writePage(t *testing.T, page string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "kroot.1")
	if err := os.WriteFile(path, []byte(page), 0o600); err != nil {
		t.Fatalf("write page: %v", err)
	}
	return path
}
