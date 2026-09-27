package cli

import (
	"flag"
	"fmt"
	"io"
)

// PrintFlags writes fs's flag list to w, indented the way (*flag.FlagSet)
// does it.
//
// It exists because PrintDefaults writes to the flag set's output, and kroot
// deliberately points that at io.Discard so flag's own chatter never reaches a
// stream. Rendering through this helper sends the list to the caller's writer
// instead — without it, help ends at a "Flags:" heading with nothing under it,
// which is what the binary did before this package existed.
func PrintFlags(w io.Writer, fs *flag.FlagSet) error {
	previous := fs.Output()
	defer fs.SetOutput(previous)

	// PrintDefaults reports nothing: it writes and moves on. An errorRecorder
	// captures the first write failure so the caller still learns about it
	// rather than being told the help was printed when it was not.
	recorder := &errorRecorder{w: w}
	fs.SetOutput(recorder)
	fs.PrintDefaults()

	if recorder.err != nil {
		return fmt.Errorf("writing flags: %w", recorder.err)
	}
	return nil
}

// errorRecorder forwards writes and remembers the first failure.
type errorRecorder struct {
	w   io.Writer
	err error
}

func (r *errorRecorder) Write(p []byte) (int, error) {
	n, err := r.w.Write(p)
	if err != nil && r.err == nil {
		r.err = err
	}
	return n, err
}
