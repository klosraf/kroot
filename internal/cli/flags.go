package cli

import (
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
)

// PrintFlags writes fs's flag list to w using kroot's own column rules: two
// leading spaces, the name column padded to the longest name plus two, and the
// usage text in the same column a command summary would occupy.
//
// It deliberately does not delegate to (*flag.FlagSet).PrintDefaults. That
// function is a fine implementation of its own contract and the wrong one for
// this screen: it separates the name from its usage with a literal tab and a
// four-space hanging indent, so the Flags block and the Commands block above it
// obeyed two different typographic rules inside one screen. The tab is also the
// one whitespace character that re-expands per tab-stop and survives
// reindentation, which is why it is a defect here and not a style preference.
//
// Rendering from the flag set rather than from a description string is kept: the
// set is the single source of truth, so help cannot describe a flag the binary
// does not accept. Sort order is by name, so the same flags always print in the
// same order and a diff of the help output means something changed.
//
// The set is walked rather than printed through, so the caller learns about a
// failed write instead of being told the help was printed when it was not.
func PrintFlags(w io.Writer, fs *flag.FlagSet) error {
	type entry struct{ name, usage string }

	var entries []entry
	fs.VisitAll(func(f *flag.Flag) {
		entries = append(entries, entry{name: f.Name, usage: f.Usage})
	})

	// PrintDefaults emits flags in lexicographic order and so does VisitAll, but
	// sorting here makes the guarantee local to this function rather than a
	// property of the standard library's traversal that could change.
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	// The width is measured on the rendered label — dash included — because that
	// is the string the column has to hold. Measuring the bare name instead
	// silently shortens the gap by one on every row, which is invisible with a
	// single flag and shows up the moment a second, longer one is added.
	width := 0
	for _, e := range entries {
		if n := len(flagLabel(e.name)); n > width {
			width = n
		}
	}
	width += 2 // the same gap the command list leaves between name and summary

	for _, e := range entries {
		// A flag's usage is authored prose that may itself contain a newline,
		// so it is normalised to spaces: a wrapped entry would break the column
		// for every row after it, which is the defect this function exists to
		// remove.
		usage := strings.Join(strings.Fields(e.usage), " ")
		if _, err := fmt.Fprintf(w, "  %-*s%s\n", width, flagLabel(e.name), usage); err != nil {
			return fmt.Errorf("writing flags: %w", err)
		}
	}
	return nil
}

// flagLabel renders a flag name as the caller types it. Both the width
// calculation and the row itself go through it, so the column is measured and
// written from one definition and cannot drift between them.
func flagLabel(name string) string { return "-" + name }
