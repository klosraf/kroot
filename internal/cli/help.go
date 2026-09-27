package cli

import (
	"fmt"
	"io"
)

// FlagsWriter renders the program's global flags, indented, without a heading.
//
// It is a function rather than a pre-rendered string so the text is produced
// from the flag set itself — the single source of truth — and so it is only
// ever rendered when help is actually asked for.
type FlagsWriter func(w io.Writer) error

// Program describes the command surface of one binary.
type Program struct {
	// Name is the binary as the caller types it, e.g. "kroot".
	Name string
	// Summary is one sentence describing the program, used in the help header
	// and by the manual.
	Summary string
	// UsageLine is the top-level invocation, e.g. "kroot [flags] <command>".
	UsageLine string
	// Commands is the validated set of subcommands.
	Commands *Registry
	// Flags renders the global flag list. When nil, help omits the section
	// rather than printing an empty heading.
	Flags FlagsWriter
}

// GeneralHelp writes what the program is, how it is invoked, the commands it
// offers and the flags it accepts.
func (p *Program) GeneralHelp(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s - %s\n\nUsage:\n  %s\n\nCommands:\n", p.Name, p.Summary, p.UsageLine); err != nil {
		return fmt.Errorf("writing help: %w", err)
	}

	if err := p.writeCommandList(w); err != nil {
		return err
	}

	if p.Flags == nil {
		return nil
	}
	if _, err := io.WriteString(w, "\nFlags:\n"); err != nil {
		return fmt.Errorf("writing help: %w", err)
	}
	return p.Flags(w)
}

// CommandHelp writes the help of one command, which the caller has already
// resolved: the framework renders, main decides what an unknown name means.
func (p *Program) CommandHelp(w io.Writer, c Command) error {
	usage := c.Usage
	if usage == "" {
		usage = p.Name + " " + c.Name
	}

	body := c.Long
	if body == "" {
		body = c.Summary
	}

	if _, err := fmt.Fprintf(w, "Usage:\n  %s\n\n%s\n", usage, body); err != nil {
		return fmt.Errorf("writing help: %w", err)
	}
	return nil
}

// writeCommandList writes one line per command, sorted by name and padded so
// the summaries line up. The padding is computed from the longest name rather
// than hard-coded, so adding a command cannot misalign the list.
func (p *Program) writeCommandList(w io.Writer) error {
	commands := p.Commands.Commands()

	width := 0
	for _, c := range commands {
		if len(c.Name) > width {
			width = len(c.Name)
		}
	}
	width += 2 // at least two spaces between a name and its summary

	for _, c := range commands {
		if _, err := fmt.Fprintf(w, "  %-*s%s\n", width, c.Name, c.Summary); err != nil {
			return fmt.Errorf("writing help: %w", err)
		}
	}
	return nil
}
