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

// Exit codes are part of the public CLI contract and scripts branch on them; see
// docs/enterprise/api-compatibility.md § "Exit codes". They are named here rather
// than written at the os.Exit call sites so that a typo cannot silently redefine
// what a caller observes.
const (
	exitSuccess = 0 // the invocation did what it was asked to do
	exitFailure = 1 // runtime failure: I/O, network, dependency, configuration
	exitUsage   = 2 // usage error: unknown flag, unknown command, bad argument
)

// ErrUsage marks a failure caused by how the program was invoked rather than by
// what it was asked to do. It is the sentinel behind exit code 2, so a caller can
// tell "you called me wrong" from "it broke while running".
var ErrUsage = errors.New("usage error")

// ErrUnknownCommand is returned when the user passes an unrecognised command.
var ErrUnknownCommand = errors.New("unknown command")

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		slog.Error("kroot failed", "err", err)
		os.Exit(exitCodeFor(err))
	}
}

// exitCodeFor maps an error from run to the exit code the contract promises. A
// failure the caller can fix by reinvoking is a usage error; everything else is a
// runtime failure, because no reinvocation changes the outcome.
func exitCodeFor(err error) int {
	switch {
	case err == nil:
		return exitSuccess
	case errors.Is(err, ErrUsage):
		return exitUsage
	default:
		return exitFailure
	}
}

// run executes the kroot CLI and returns the first fatal error.
func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.Usage = func() { printUsage(out, fs) }

	showVersion := fs.Bool("version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		// flag.ErrHelp means the user asked for help and got it: printing usage
		// and reporting success is what the contract says, and it is what
		// `kroot help` already did. `-h` failing while `help` succeeded was two
		// spellings of one request with two different answers.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		// Every other parse failure is a usage error: no amount of retrying
		// makes -nope a valid flag.
		return fmt.Errorf("%w: parsing flags: %w", ErrUsage, err)
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
		// Wrapped in ErrUsage as well: an unrecognised command is the caller's
		// mistake and maps to exit 2, while ErrUnknownCommand still answers
		// "which command?" for tests and callers inspecting the chain.
		return fmt.Errorf("%w: %w: %q", ErrUsage, ErrUnknownCommand, command)
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
