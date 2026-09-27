// Command kroot is the entry point of the kroot application.
//
// main() stays intentionally tiny: flag parsing, wiring and command dispatch
// live in run() so the whole program is testable without spawning a process.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Build metadata. All three values are injected at link time:
//
//	go build -ldflags "-X main.version=v1.0.0 -X main.commit=abc1234 -X main.buildTime=2026-09-26T19:00:00Z"
//
// They are variables rather than constants because a variable is the only form
// the linker can patch.
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

// ErrUnknownCommand is returned when the user passes an unrecognised command.
var ErrUnknownCommand = errors.New("unknown command")

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		slog.Error("kroot failed", "err", err)
		os.Exit(1)
	}
}

// run executes the kroot CLI and returns the first fatal error.
func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.Usage = func() { printUsage(out, fs) }

	showVersion := fs.Bool("version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	if *showVersion {
		return printVersion(out)
	}

	command := "help"
	if fs.NArg() > 0 {
		command = fs.Arg(0)
	}

	switch command {
	case "help":
		printUsage(out, fs)
		return nil
	case "version":
		return printVersion(out)
	default:
		return fmt.Errorf("%w: %q", ErrUnknownCommand, command)
	}
}

// printVersion writes the application version and its build metadata to out.
func printVersion(out io.Writer) error {
	if _, err := fmt.Fprintf(out, "kroot %s (commit %s, built %s)\n", version, commit, buildTime); err != nil {
		return fmt.Errorf("writing version: %w", err)
	}
	return nil
}

// usageTemplate is the static half of the help text; the flag list is appended
// by (*flag.FlagSet).PrintDefaults.
const usageTemplate = `kroot - application skeleton

Usage:
  kroot [flags] <command>

Commands:
  help       print this help and exit
  version    print the version and exit

Flags:
`

// printUsage writes the CLI help text to out.
func printUsage(out io.Writer, fs *flag.FlagSet) {
	// A write failure is not actionable: the help text targets the process
	// stdout/stderr and there is no caller able to recover from it.
	_, _ = io.WriteString(out, usageTemplate)
	fs.PrintDefaults()
}
