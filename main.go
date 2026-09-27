// Command kroot is the entry point of the kroot application.
//
// main() stays intentionally tiny: flag parsing, wiring and command dispatch
// live in run() so the whole program is testable without spawning a process.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
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
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
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
//
// stdout carries what the caller asked for — help text on request, the version
// line. stderr carries diagnostics. A caller that redirects stdout must not
// receive usage text produced by a failure, and a caller that discards stdout
// must still be able to tell what went wrong. See
// docs/enterprise/api-compatibility.md § "Streams".
func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)

	// flag calls fs.Usage in two situations: an explicit -h, which is a success
	// and belongs on stdout, and a parse failure, which is a diagnostic and
	// belongs on stderr. The callback cannot tell them apart, so it renders into
	// a buffer and run() decides where the text goes. flag's own error line is
	// discarded because main logs the failure once, structurally, instead of
	// twice in two different formats on two different streams.
	var usage bytes.Buffer
	fs.SetOutput(io.Discard)
	fs.Usage = func() { printUsage(&usage, fs) }

	showVersion := fs.Bool("version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// The user asked for help and got it, so the answer is the requested
			// output: exit 0, text on stdout. `-h` failing while `help` succeeded
			// was two spellings of one request with two different answers.
			_, _ = usage.WriteTo(stdout)
			return nil
		}
		// Every other parse failure is a usage error: no amount of retrying makes
		// -nope a valid flag. The usage text goes to stderr so that redirecting
		// stdout does not capture help text from a failure. A write failure here
		// is swallowed for the same reason printUsage swallows them: there is
		// nowhere left to report it.
		_, _ = usage.WriteTo(stderr)
		return fmt.Errorf("%w: parsing flags: %w", ErrUsage, err)
	}

	if *showVersion {
		return printVersion(stdout)
	}

	command := "help"
	if fs.NArg() > 0 {
		command = fs.Arg(0)
	}

	switch command {
	case "help":
		printUsage(stdout, fs)
		return nil
	case "version":
		return printVersion(stdout)
	default:
		// Wrapped in ErrUsage as well: an unrecognised command is the caller's
		// mistake and maps to exit 2, while ErrUnknownCommand still answers
		// "which command?" for tests and callers inspecting the chain.
		return fmt.Errorf("%w: %w: %q", ErrUsage, ErrUnknownCommand, command)
	}
}

// printVersion writes the application version and its build metadata to out.
//
// The toolchain comes from the runtime rather than from -ldflags, so it always
// names the compiler that actually produced this binary. Reporting the revision
// without the compiler would repeat the blind spot that let a pinned dev tool be
// silently built by an undeclared Go version: an artifact that says where it came
// from but not what made it. See docs/enterprise/versioning-policy.md § "Build
// metadata".
func printVersion(out io.Writer) error {
	if _, err := fmt.Fprintf(out, "kroot %s (commit %s, built %s, %s)\n", version, commit, buildTime, runtime.Version()); err != nil {
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
