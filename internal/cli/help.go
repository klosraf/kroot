package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

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
	// FlagSet holds the program's global flags. Help and the manual both render
	// from it rather than from a pre-rendered string, so neither can describe a
	// flag the binary does not accept. When nil, both omit the section rather
	// than printing an empty heading.
	//
	// The set is held rather than a rendering function because the two
	// renderers need it in different shapes: help wants the FlagSet's own
	// layout, while the manual wants each flag's name and usage separately.
	FlagSet *flag.FlagSet
}

// GeneralHelp writes what the program is, how it is invoked, the commands it
// offers, the flags it accepts and where to go next.
func (p *Program) GeneralHelp(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s - %s\n\nUsage:\n  %s\n\nCommands:\n", p.Name, p.Summary, p.UsageLine); err != nil {
		return fmt.Errorf("writing help: %w", err)
	}

	if err := p.writeCommandList(w); err != nil {
		return err
	}

	if p.FlagSet != nil {
		if _, err := io.WriteString(w, "\nFlags:\n"); err != nil {
			return fmt.Errorf("writing help: %w", err)
		}
		if err := PrintFlags(w, p.FlagSet); err != nil {
			return err
		}
	}

	return p.writeNextSteps(w)
}

// writeNextSteps closes the screen by naming the two things a reader who has just
// seen the command list cannot guess: how to read about one command, and where
// the full contract lives.
//
// It existed because the help screen ended at the flag list. Every command was
// therefore discoverable but none was explorable: the affordance that made the
// other three commands reachable — the hint that they take a name — was the one
// thing absent from the screen. A first run is a request for orientation, and
// orientation without a next step leaves the reader to guess.
//
// The lines are built from p.Name, so a program that renames itself renames them
// too. The block is omitted rather than printed empty when the commands it names
// are not registered, which is the same rule every other section follows: no
// heading over nothing.
func (p *Program) writeNextSteps(w io.Writer) error {
	var lines []string

	if _, ok := p.Commands.Lookup("help"); ok {
		lines = append(lines, fmt.Sprintf("Run %q for detail on one command.", p.Name+" help <command>"))
	}
	if _, ok := p.Commands.Lookup("man"); ok {
		lines = append(lines, fmt.Sprintf("Run %q for the full manual, including exit codes and environment.",
			"man "+p.Name))
	}

	if len(lines) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(w, "\n%s\n", strings.Join(lines, "\n")); err != nil {
		return fmt.Errorf("writing help: %w", err)
	}
	return nil
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

// UnknownCommand reports a command name the program does not have.
//
// The registry's rejection is the whole answer when it offered a near miss,
// because that suggestion is itself the recovery. When nothing was close, the
// caller is told what they typed and left with no next step at all, so the
// program's own command list is named instead — the one failure a reader cannot
// act on is the one with nothing to try next.
//
// Only the program knows how it is invoked, which is why this lives here and
// not in the registry.
func (p *Program) UnknownCommand(name string) error {
	err := p.Commands.Error(name)
	if len(p.Commands.Suggestions(name)) > 0 {
		return err
	}
	return fmt.Errorf("%w; see %q for the command list", err, p.Name+" help")
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
