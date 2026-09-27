package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

// manualSection is the man section a user-facing command belongs in.
const manualSection = "1"

// ExitCode is one line of the process exit contract, as a caller reads it.
//
// It is data rather than text so the manual and the code that returns these
// values cannot disagree: the caller supplies them from the constants it exits
// with, and this file only knows how to typeset what it is given.
type ExitCode struct {
	// Code is the numeric status the process returns.
	Code int
	// Name is the short label, e.g. "usage error".
	Name string
	// Meaning is one sentence a script author can act on.
	Meaning string
}

// ManualInfo is the provenance a generated page carries.
//
// A manual page that does not say which version it documents cannot be checked
// against the binary it came from, so the version and date are part of the page
// rather than decoration added by whoever installs it.
type ManualInfo struct {
	// Version is the version the page documents. It appears in the footer
	// beside the page name.
	Version string
	// Date is the page's date field, already formatted for display — a
	// timestamp is unhelpful here, and formatting it is the caller's job
	// because only the caller knows the build's timezone.
	Date string
	// ExitCodes documents the exit contract. When empty, the section is
	// omitted rather than printed empty.
	ExitCodes []ExitCode
}

// WriteManual writes the roff source of a manual page to w.
//
// With an empty command it documents the program: synopsis, every subcommand and
// the global flags. With a command name it documents that command alone, under
// its own page name, the way `man kroot-completion` would.
//
// The page is generated from the same registry help and completion render, so a
// command cannot be reachable but undocumented: there is no second list to keep
// in step. The cost is that a page is only as current as the binary that wrote
// it, which is the intended trade for a CLI whose commands change with releases.
//
// An unknown command name returns the registry's own rejection, so a typo in a
// manual request is answered like a typo anywhere else.
func WriteManual(w io.Writer, p *Program, command string, info ManualInfo) error {
	if command == "" {
		return programPage(w, p, info)
	}

	c, ok := p.Commands.Lookup(command)
	if !ok {
		return p.Commands.Error(command)
	}
	return commandPage(w, p, c, info)
}

// programPage writes the page for the binary as a whole.
func programPage(w io.Writer, p *Program, info ManualInfo) error {
	if err := writeHeader(w, p.Name, p.Name, info); err != nil {
		return err
	}

	if err := writeSection(w, "NAME", fmt.Sprintf("%s - %s", p.Name, p.Summary)); err != nil {
		return err
	}

	if err := writeSection(w, "SYNOPSIS", p.UsageLine); err != nil {
		return err
	}

	if err := writeSection(w, "DESCRIPTION", p.Summary); err != nil {
		return err
	}

	// The list is the registry's, sorted, so the page cannot name a command that
	// does not exist or omit one that does — the same guarantee completion
	// derives its own answer from.
	commands := p.Commands.Commands()
	if len(commands) > 0 {
		if err := writeSection(w, "COMMANDS", ""); err != nil {
			return err
		}
		for _, c := range commands {
			usage := c.Usage
			if usage == "" {
				usage = p.Name + " " + c.Name
			}
			if err := writeTerm(w, usage, c.Summary); err != nil {
				return err
			}
		}
	}

	if flags := collectFlags(p.FlagSet); len(flags) > 0 {
		if err := writeSection(w, "OPTIONS", ""); err != nil {
			return err
		}
		for _, f := range flags {
			if err := writeTerm(w, dash(f.name), f.usage); err != nil {
				return err
			}
		}
	}

	return writeFooter(w, info)
}

// commandPage writes the page for one command.
func commandPage(w io.Writer, p *Program, c Command, info ManualInfo) error {
	// "kroot completion" documents as kroot-completion(1), the naming a reader
	// would expect from `man kroot-completion`.
	if err := writeHeader(w, p.Name+"-"+c.Name, p.Name, info); err != nil {
		return err
	}

	if err := writeSection(w, "NAME", fmt.Sprintf("%s - %s", c.Name, c.Summary)); err != nil {
		return err
	}

	usage := c.Usage
	if usage == "" {
		usage = p.Name + " " + c.Name
	}
	if err := writeSection(w, "SYNOPSIS", usage); err != nil {
		return err
	}

	body := c.Long
	if body == "" {
		body = c.Summary
	}
	if err := writeSection(w, "DESCRIPTION", body); err != nil {
		return err
	}

	return writeFooter(w, info)
}

// writeFooter writes the sections every page ends with: the exit contract, and
// where the rest of the documentation lives.
func writeFooter(w io.Writer, info ManualInfo) error {
	if len(info.ExitCodes) > 0 {
		if err := writeSection(w, "EXIT STATUS", ""); err != nil {
			return err
		}
		for _, e := range info.ExitCodes {
			if err := writeTerm(w, fmt.Sprintf("%d", e.Code), fmt.Sprintf("%s: %s", e.Name, e.Meaning)); err != nil {
				return err
			}
		}
	}

	// The exit contract and the versioning rules are documented in prose, and
	// prose is where the reasoning lives. Pointing at it beats restating a
	// summary here that could disagree with the document it came from.
	//
	// The lines are kept short deliberately: roff convention is a source file
	// that stays under 80 columns, so a reader inspecting the generated page is
	// not scrolling through reflowed prose.
	return writeSection(w, "SEE ALSO",
		"The CLI contract, the exit-code table and the versioning rules are\n"+
			"documented in this repository under docs/enterprise/.")
}

// writeHeader writes the .TH line that gives a page its title and footers.
//
// The title is upper-cased because every man convention does it, and a hyphen in
// a title has to be escaped: in roff an unescaped hyphen is a hyphen glyph
// rather than the minus a reader expects in a name like KROOT-COMPLETION.
func writeHeader(w io.Writer, page, binary string, info ManualInfo) error {
	_, err := fmt.Fprintf(w, ".TH %s %s %s %s %s\n",
		quoteRoff(strings.ToUpper(page)),
		quoteRoff(manualSection),
		quoteRoff(info.Date),
		quoteRoff(binary+" "+info.Version),
		quoteRoff(binary+" Manual"),
	)
	if err != nil {
		return fmt.Errorf("writing manual header: %w", err)
	}
	return nil
}

// writeSection writes a headed section. An empty body still produces the heading
// and a blank line, which is what roff expects before a run of .TP entries.
func writeSection(w io.Writer, name, body string) error {
	if _, err := fmt.Fprintf(w, ".SH %s\n", quoteRoff(name)); err != nil {
		return fmt.Errorf("writing manual section %s: %w", name, err)
	}
	if body == "" {
		return nil
	}
	if err := writeText(w, body); err != nil {
		return fmt.Errorf("writing manual section %s: %w", name, err)
	}
	return nil
}

// writeTerm writes one .TP entry: an indented term followed by its definition.
// This is how a man page lists a command, a flag or an exit code, and indenting
// is what makes the rendered page a list rather than a paragraph.
//
// The term is escaped as text, not quoted as a macro argument, because .TP takes
// no arguments: the term is the next non-blank line, so quotes around it would be
// rendered as part of the term.
func writeTerm(w io.Writer, term, definition string) error {
	if _, err := fmt.Fprintf(w, ".TP\n%s\n", escapeRoff(term)); err != nil {
		return fmt.Errorf("writing manual term: %w", err)
	}
	return writeText(w, definition)
}

// flagNames is one global flag as the manual needs it: the name a reader types
// and the description it was declared with.
type flagNames struct {
	name  string
	usage string
}

// collectFlags reads the global flags out of the flag set.
//
// It walks the set rather than reusing the help renderer because the two want
// different shapes: help prints the FlagSet's own layout, while a man page needs
// each flag's name and description as separate fields.
//
// VisitAll walks in lexicographical order, so the page is stable for a given set
// of flags regardless of the order they were declared in. A nil set yields no
// flags rather than a panic, so a program with no global flags needs no special
// case at the call site.
func collectFlags(fs *flag.FlagSet) []flagNames {
	if fs == nil {
		return nil
	}

	var flags []flagNames
	fs.VisitAll(func(f *flag.Flag) {
		flags = append(flags, flagNames{name: f.Name, usage: f.Usage})
	})
	return flags
}

// dash renders a flag name the way a reader writes it. The set is then wrapped so
// the definition starts on its own line, which is what .TP expects.
func dash(name string) string {
	return "-" + name
}

// writeText writes a block of body text, one output line per input line.
//
// roff joins consecutive input lines into one filled paragraph, so the lines are
// written verbatim and left for it to fill. Every line is escaped, because a
// summary or a long description is free text and roff gives a leading dot
// control over the whole line.
func writeText(w io.Writer, text string) error {
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if _, err := fmt.Fprintf(w, "%s\n", escapeRoff(line)); err != nil {
			return fmt.Errorf("writing manual text: %w", err)
		}
	}
	return nil
}

// escapeRoff makes s safe to place in a roff file.
//
// Three characters matter, and they interact, so this is a single pass rather
// than three replacements: an earlier replacement would escape the backslashes a
// later one introduces.
//
//   - A backslash starts an escape sequence, so a literal one is written \e.
//   - A hyphen is a hyphen glyph, not a minus; \- asks for the character a
//     reader expects in a name like kroot-completion.
//   - A line whose first non-blank character is . or ' is a control line, so it
//     is prefixed with \& — a zero-width character that suppresses the
//     interpretation while printing nothing.
//   - A tab becomes a space, because roff does not honour a tab in filled text.
func escapeRoff(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)

	// Blank lines are tracked separately from the escape state: roff's rule is
	// about the first character on the line, and a tab counts as blank there.
	atLineStart := true

	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\e`)
			atLineStart = false
		case r == '-':
			b.WriteString(`\-`)
			atLineStart = false
		case r == '\n':
			b.WriteRune(r)
			atLineStart = true
		case r == '\t':
			// A tab is not honoured in filled text — mandoc warns about it and
			// the formatter has no glyph for it — so it becomes the space the
			// author meant. Converting keeps the page renderable instead of
			// leaving a warning in every install of it.
			b.WriteRune(' ')
			// Blank does not end the run of leading blanks, so a line that opens
			// with a tab and then a dot is still a control line.
		case r == ' ':
			b.WriteRune(r)
			// As above: a leading blank does not stop the next character from
			// being the one that decides whether this is a control line.
		case (r == '.' || r == '\'') && atLineStart:
			b.WriteString(`\&`)
			b.WriteRune(r)
			atLineStart = false
		default:
			b.WriteRune(r)
			atLineStart = false
		}
	}
	return b.String()
}

// quoteRoff renders s as one roff macro argument.
//
// Macro arguments are not body text, and treating them as such breaks the page
// in two ways this function exists to prevent:
//
//   - An argument containing a space is read as several arguments, so a footer
//     like "kroot 1.2.3" silently truncates at the space and the rest is
//     discarded as excess. Every argument is therefore quoted, which is why the
//     footers survive at all.
//   - Hyphens are deliberately left alone. A backslash-escaped hyphen is
//     correct in a name like kroot\-completion but destroys a date, which a
//     formatter has to parse literally — an escaped "2026-09-27" is not a date
//     any formatter recognises.
//
// A literal double quote is written doubled inside a quoted argument, and a
// literal backslash as \e.
func quoteRoff(s string) string {
	s = strings.ReplaceAll(s, `\`, `\e`)
	s = strings.ReplaceAll(s, `"`, `""`)
	return `"` + s + `"`
}
