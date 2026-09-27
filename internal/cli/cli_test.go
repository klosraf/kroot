package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// noop is the smallest valid command body, so a test can vary one field at a
// time without repeating a closure.
func noop() RunFunc { return func(_ context.Context, _ Env) error { return nil } }

// TestNewRejectsInvalidSets is the regression test for declaring a command
// surface: every rule is checked once at construction, so a mistake fails at
// startup rather than as a command that is silently unreachable or shadows
// another.
func TestNewRejectsInvalidSets(t *testing.T) {
	tests := []struct {
		name     string
		commands []Command
		want     string
	}{
		{
			name:     "empty name",
			commands: []Command{{Name: "", Summary: "s", Run: noop()}},
			want:     "invalid",
		},
		{
			name:     "upper case name",
			commands: []Command{{Name: "Version", Summary: "s", Run: noop()}},
			want:     "invalid",
		},
		{
			name:     "name with a space",
			commands: []Command{{Name: "ver sion", Summary: "s", Run: noop()}},
			want:     "invalid",
		},
		{
			name:     "name starting with a digit",
			commands: []Command{{Name: "1version", Summary: "s", Run: noop()}},
			want:     "invalid",
		},
		{
			name:     "name starting with a dash",
			commands: []Command{{Name: "-version", Summary: "s", Run: noop()}},
			want:     "invalid",
		},
		{
			name:     "missing summary",
			commands: []Command{{Name: "version", Run: noop()}},
			want:     "summary",
		},
		{
			name:     "missing Run",
			commands: []Command{{Name: "version", Summary: "s"}},
			want:     "Run",
		},
		{
			name: "duplicate name",
			commands: []Command{
				{Name: "version", Summary: "s", Run: noop()},
				{Name: "version", Summary: "other", Run: noop()},
			},
			want: "twice",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.commands...)
			if err == nil {
				t.Fatalf("New(%v) = nil error; want a rejection", tc.commands)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("New error = %q; want it to name %q", err, tc.want)
			}
		})
	}
}

// TestNewAcceptsValidNames pins the accepted shape, including the digits and
// dashes a later command such as `session-1` will need.
func TestNewAcceptsValidNames(t *testing.T) {
	for _, name := range []string{"version", "help", "completion", "session-1", "a1", "x"} {
		t.Run(name, func(t *testing.T) {
			if _, err := New(Command{Name: name, Summary: "s", Run: noop()}); err != nil {
				t.Errorf("New(%q) error = %v; want nil", name, err)
			}
		})
	}
}

// TestRegistryListsSortedNames proves rendering does not depend on declaration
// order: help, completion and the manual all read the same sorted view.
func TestRegistryListsSortedNames(t *testing.T) {
	r, err := New(
		Command{Name: "version", Summary: "v", Run: noop()},
		Command{Name: "completion", Summary: "c", Run: noop()},
		Command{Name: "help", Summary: "h", Run: noop()},
	)
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	if got, want := strings.Join(r.Names(), ","), "completion,help,version"; got != want {
		t.Errorf("Names() = %q; want %q", got, want)
	}

	commands := r.Commands()
	for i, want := range []string{"completion", "help", "version"} {
		if commands[i].Name != want {
			t.Errorf("Commands()[%d].Name = %q; want %q", i, commands[i].Name, want)
		}
	}
}

// TestLookupSeparatesFoundFromAbsent keeps the two outcomes distinguishable, so
// a caller cannot mistake "registered with no behaviour" for "not registered".
func TestLookupSeparatesFoundFromAbsent(t *testing.T) {
	r, err := New(Command{Name: "version", Summary: "v", Run: noop()})
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	if _, ok := r.Lookup("version"); !ok {
		t.Error("Lookup(version) reported absent; want present")
	}
	if _, ok := r.Lookup("Version"); ok {
		t.Error("Lookup(Version) reported present; want absent: names are case sensitive")
	}
}

// TestSuggestionsAreBoundedOrderedAndDeterministic covers the suggestion rules:
// distance decides, name breaks ties, the list is capped, and a name nothing
// resembles gets no guesses at all.
func TestSuggestionsAreBoundedOrderedAndDeterministic(t *testing.T) {
	r, err := New(
		Command{Name: "man", Summary: "m", Run: noop()},
		Command{Name: "map", Summary: "m", Run: noop()},
		Command{Name: "help", Summary: "h", Run: noop()},
		Command{Name: "version", Summary: "v", Run: noop()},
		Command{Name: "completion", Summary: "c", Run: noop()},
	)
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	tests := []struct {
		name  string
		typed string
		want  []string
	}{
		{name: "one substitution", typed: "versioo", want: []string{"version"}},
		{name: "missing letter", typed: "versin", want: []string{"version"}},
		{name: "extra letter", typed: "versions", want: []string{"version"}},
		{name: "transposition", typed: "hlep", want: []string{"help"}},
		{name: "tie broken by name", typed: "map", want: []string{"map", "man"}},
		{name: "nothing close", typed: "kubernetes", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := r.Suggestions(tc.typed)
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Errorf("Suggestions(%q) = %v; want %v", tc.typed, got, tc.want)
			}
		})
	}
}

// TestSuggestionsRespectTheCap proves a short name cannot produce a wall of
// guesses: the cap is what keeps the real error readable.
func TestSuggestionsRespectTheCap(t *testing.T) {
	r, err := New(
		Command{Name: "aaa", Summary: "s", Run: noop()},
		Command{Name: "aab", Summary: "s", Run: noop()},
		Command{Name: "aac", Summary: "s", Run: noop()},
		Command{Name: "aad", Summary: "s", Run: noop()},
	)
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	if got := len(r.Suggestions("aaa")); got > maxSuggestions {
		t.Errorf("len(Suggestions()) = %d; want at most %d", got, maxSuggestions)
	}
}

// TestNewRejectsContradictoryOperands covers the one operand declaration a registry
// cannot make sense of: a first position that accepts a closed vocabulary *and* the
// command names. It is a programmer error, and New is the only place early enough to
// say so — by the time a shell script exists, the contradiction is baked into a file
// someone installed, and a shell is a poor place to report it.
func TestNewRejectsContradictoryOperands(t *testing.T) {
	_, err := New(
		Command{
			Name:     "confused",
			Summary:  "s",
			Operands: Operands{Values: []string{"a"}, CommandNames: true},
			Run:      noop(),
		},
	)
	if err == nil {
		t.Fatal("New() with both an operand vocabulary and the command names = nil; want a rejection")
	}
	if want := "one or the other"; !strings.Contains(err.Error(), want) {
		t.Errorf("New() error = %v; want it to contain %q", err, want)
	}
}

// TestErrorKeepsTheChainAndAddsTheGuess pins the diagnostic contract: the
// message stays matchable with errors.Is on both sentinels, and the suggestion
// is appended rather than replacing the facts.
func TestErrorKeepsTheChainAndAddsTheGuess(t *testing.T) {
	r, err := New(
		Command{Name: "version", Summary: "v", Run: noop()},
		Command{Name: "man", Summary: "m", Run: noop()},
		Command{Name: "map", Summary: "m", Run: noop()},
	)
	if err != nil {
		t.Fatalf("New() error = %v; want nil", err)
	}

	withGuess := r.Error("versioo")
	if !errors.Is(withGuess, ErrUsage) {
		t.Errorf("Error(versioo) = %v; want it to match ErrUsage", withGuess)
	}
	if !errors.Is(withGuess, ErrUnknownCommand) {
		t.Errorf("Error(versioo) = %v; want it to match ErrUnknownCommand", withGuess)
	}
	for _, want := range []string{`"versioo"`, `did you mean "version"?`} {
		if !strings.Contains(withGuess.Error(), want) {
			t.Errorf("Error(versioo) = %q; want it to contain %q", withGuess, want)
		}
	}

	withoutGuess := r.Error("kubernetes")
	if !errors.Is(withoutGuess, ErrUsage) {
		t.Errorf("Error(kubernetes) = %v; want it to match ErrUsage", withoutGuess)
	}
	if strings.Contains(withoutGuess.Error(), "did you mean") {
		t.Errorf("Error(kubernetes) = %q; want no guess when nothing is close", withoutGuess)
	}

	several := r.Error("ma")
	if !strings.Contains(several.Error(), `did you mean one of "man"`) {
		t.Errorf("Error(ma) = %q; want the plural form when several are close", several)
	}
}

// TestEditDistanceMeasuresRunesNotBytes keeps the measure honest for text that
// is not ASCII: a one-rune difference is one edit, whatever its encoding.
func TestEditDistanceMeasuresRunesNotBytes(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{name: "identical", a: "help", b: "help", want: 0},
		{name: "one substitution", a: "help", b: "held", want: 1},
		{name: "one insertion", a: "hel", b: "help", want: 1},
		{name: "one deletion", a: "helpp", b: "help", want: 1},
		{name: "two substitutions", a: "help", b: "hxlx", want: 2},
		{name: "far apart", a: "a", b: "abcdefgh", want: maxSuggestionDistance + 1},
		{name: "accented letter counts once", a: "sesión", b: "sesion", want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := editDistance(tc.a, tc.b, maxSuggestionDistance); got != tc.want {
				t.Errorf("editDistance(%q, %q) = %d; want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
