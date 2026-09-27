// Command kroot is the entry point of the kroot application.
//
// main() stays intentionally tiny: flag parsing, wiring and command dispatch
// live in run() so the whole program is testable without spawning a process.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/klosraf/kroot/internal/cli"
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

// Usage errors are the framework's to define, since it is the layer that decides
// what a well-formed invocation is; package main only maps them to process exit
// codes below.
func main() {
	os.Exit(realMain(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

// realMain wires configuration into the process and returns the exit code. It
// reads KROOT_LOG_LEVEL itself so the wiring — not only the parsing — is
// covered, and so a typo in the variable name fails a test rather than in
// production. main() stays one statement: parse, wire, hand off, exit.
//
// Logging is configured before anything can fail, so every later failure is
// reported by the logger this function just built, never by a default handler
// that was never chosen.
func realMain(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	level, err := parseLogLevel(os.Getenv("KROOT_LOG_LEVEL"))
	slog.SetDefault(newLogger(stderr, isTerminal(stderr), level))

	if err != nil {
		slog.Error("kroot failed", "err", err)
		return exitCodeFor(err)
	}
	if err := run(ctx, args, stdout, stderr); err != nil {
		slog.Error("kroot failed", "err", err)
		return exitCodeFor(err)
	}
	return exitSuccess
}

// exitCodeFor maps an error from run to the exit code the contract promises. A
// failure the caller can fix by reinvoking is a usage error; everything else is a
// runtime failure, because no reinvocation changes the outcome.
func exitCodeFor(err error) int {
	switch {
	case err == nil:
		return exitSuccess
	case errors.Is(err, cli.ErrUsage):
		return exitUsage
	default:
		return exitFailure
	}
}

// allowedLogLevels is the full accepted vocabulary for KROOT_LOG_LEVEL, in the
// order the manual and the error message present it. A single definition means
// neither can drift from the parser.
var allowedLogLevels = []string{"debug", "info", "warn", "error"}

// parseLogLevel maps KROOT_LOG_LEVEL to a slog level. Empty means unset and
// selects the documented default (info). An unknown value is rejected rather
// than coerced, so a typo fails loudly with the accepted values named.
//
// The failure returns LevelError, not LevelInfo: no configured level may
// suppress the record that explains why the configuration was rejected.
func parseLogLevel(v string) (slog.Level, error) {
	switch v {
	case "":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		// "configuration rejected" is the wording api-compatibility.md ties to
		// exit code 1, so the diagnostic and the contract use one phrase.
		return slog.LevelError, fmt.Errorf(
			"configuration rejected: KROOT_LOG_LEVEL %q: want one of %s",
			v, strings.Join(allowedLogLevels, "|"),
		)
	}
}

// newLogger builds the process logger. Terminals get human-readable text; any
// other destination gets JSON, which stays machine-readable when piped. The
// structured keys are identical either way — only the encoding differs, and the
// encoding is not part of the compatibility contract.
func newLogger(out io.Writer, tty bool, level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	if tty {
		return slog.New(slog.NewTextHandler(out, opts))
	}
	return slog.New(slog.NewJSONHandler(out, opts))
}

// statser is the subset of *os.File that isTerminal needs. Taking the
// interface rather than *os.File keeps isTerminal testable with an in-memory
// writer and lets a closed or unstatable descriptor be handled instead of
// panicking.
type statser interface {
	Stat() (fs.FileInfo, error)
}

// isTerminal reports whether w is a terminal. Terminals are character devices;
// pipes, files and other redirections are not. It reports false when w cannot
// describe itself, because the safe answer for an unknown destination is the
// machine-readable one.
//
// The test is deliberately ModeCharDevice rather than a pty probe: /dev/null is
// a character device and therefore answers true, which is the one known
// misclassification. Its consequence is only log encoding into a destination
// that discards everything it receives, so nothing can be harmed by it.
func isTerminal(w io.Writer) bool {
	s, ok := w.(statser)
	if !ok {
		return false
	}
	fi, err := s.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// run executes the kroot CLI and returns the first fatal error.
//
// stdout carries what the caller asked for — help text on request, the version
// line. stderr carries diagnostics. A caller that redirects stdout must not
// receive usage text produced by a failure, and a caller that discards stdout
// must still be able to tell what went wrong. See
// docs/enterprise/api-compatibility.md § "Streams".
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	// A cancelled context means shutdown is already underway: fail fast instead
	// of doing work nobody will observe. Long-running commands must additionally
	// select on ctx while they run.
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("run cancelled: %w", err)
	}

	fs := flag.NewFlagSet("kroot", flag.ContinueOnError)
	var usage bytes.Buffer

	// The program is built before parsing because flag needs the help text for
	// its own -h handling, and that handling is the same in both directions: a
	// requested -h is the requested output, a parse failure is a diagnostic.
	program, err := newProgram(fs)
	if err != nil {
		return err
	}

	// flag writes its own error line and then calls fs.Usage, and it does both
	// for an explicit -h (a success, stdout) and for a bad flag (a failure,
	// stderr). The callback cannot tell them apart, so it renders into a buffer
	// and run() decides where the text belongs. Writing to a bytes.Buffer cannot
	// fail, which is why the render error is dropped here and reported by the
	// paths that write to a stream.
	fs.SetOutput(io.Discard)
	fs.Usage = func() { _ = program.GeneralHelp(&usage) }

	showVersion := fs.Bool("version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = usage.WriteTo(stdout)
			return nil
		}
		// Every other parse failure is a usage error: no amount of retrying makes
		// -nope a valid flag. The usage text goes to stderr so that redirecting
		// stdout does not capture help text from a failure.
		_, _ = usage.WriteTo(stderr)
		return fmt.Errorf("%w: parsing flags: %w", cli.ErrUsage, err)
	}

	if *showVersion {
		return printVersion(stdout)
	}

	// No command is the same request as `help`: one spelling, one answer. The
	// operands are sliced only when there are any: slicing an empty slice from
	// index 1 panics, and "no arguments" is a supported invocation.
	name := "help"
	var operands []string
	if args := fs.Args(); len(args) > 0 {
		name, operands = args[0], args[1:]
	}

	command, ok := program.Commands.Lookup(name)
	if !ok {
		return program.Commands.Error(name)
	}

	return command.Run(ctx, cli.Env{Stdout: stdout, Stderr: stderr, Args: operands})
}

// newProgram declares the command surface. Every command is constructed here so
// the registry can be validated once, at startup, instead of being discovered
// one missing summary at a time.
func newProgram(fs *flag.FlagSet) (*cli.Program, error) {
	program := &cli.Program{
		Name:      "kroot",
		Summary:   "application skeleton",
		UsageLine: "kroot [flags] <command>",
		Flags: func(w io.Writer) error {
			return cli.PrintFlags(w, fs)
		},
	}

	registry, err := cli.New(
		cli.Command{
			Name:    "completion",
			Summary: "print a shell completion script",
			Usage:   "kroot completion <" + strings.Join(cli.Shells, "|") + ">",
			Long: "Print the completion script for one shell on standard output, ready to install.\n\n" +
				"The command list is written into the script, so regenerate it after upgrading kroot:\n\n" +
				"  kroot completion " + strings.Join(cli.Shells, "|") + " > <path>",
			Run: func(_ context.Context, env cli.Env) error {
				return runCompletion(program, env)
			},
		},
		cli.Command{
			Name:    "help",
			Summary: "print this help, or the help of one command",
			Usage:   "kroot help [command]",
			Long:    "Print the command list, or the detail of one command when a name is given.",
			Run: func(_ context.Context, env cli.Env) error {
				return runHelp(program, env)
			},
		},
		cli.Command{
			Name:    "version",
			Summary: "print the version and exit",
			Usage:   "kroot version",
			Long:    "Print the version, the revision it was built from and the toolchain that produced it.",
			Run: func(_ context.Context, env cli.Env) error {
				return printVersion(env.Stdout)
			},
		},
	)
	if err != nil {
		return nil, err
	}

	program.Commands = registry
	return program, nil
}

// runHelp implements the help command.
//
// A help write failure is deliberately tolerated: the text already targets
// stdout and there is nowhere left to report the failure, so failing would turn
// an unreadable help page into a non-zero exit for no benefit. Every other
// failure — an unknown name, too many operands — is actionable and is returned.
func runHelp(program *cli.Program, env cli.Env) error {
	switch {
	case len(env.Args) == 0:
		_ = program.GeneralHelp(env.Stdout)
		return nil
	case len(env.Args) > 1:
		// Rejecting rather than ignoring: silently dropping the extra operand
		// would hide a real mistake, and api-compatibility.md ties a bad
		// argument to exit 2.
		return fmt.Errorf("%w: help takes at most one command name, got %d", cli.ErrUsage, len(env.Args))
	}

	command, ok := program.Commands.Lookup(env.Args[0])
	if !ok {
		return program.Commands.Error(env.Args[0])
	}

	_ = program.CommandHelp(env.Stdout, command)
	return nil
}

// runCompletion implements the completion command.
//
// The script is what the caller asked for, so it goes to stdout and can be
// installed by redirecting: `kroot completion zsh > _kroot`. Both arity failures
// are returned rather than papered over — a missing or extra shell name is
// actionable by reinvoking, which is exactly what api-compatibility.md ties to
// exit code 2, and a bare "usage error" with no vocabulary would send the
// reader to the manual for something the error can answer.
func runCompletion(program *cli.Program, env cli.Env) error {
	shells := strings.Join(cli.Shells, "|")

	switch {
	case len(env.Args) == 0:
		return fmt.Errorf("%w: completion needs a shell: want one of %s", cli.ErrUsage, shells)
	case len(env.Args) > 1:
		return fmt.Errorf("%w: completion takes one shell, got %d: want one of %s", cli.ErrUsage, len(env.Args), shells)
	}

	// The registry is the single source of truth for what can be typed, so the
	// script cannot offer a command that does not exist or miss one that does.
	// Commands() is sorted, which is what makes the generated file stable
	// across regenerations and reviewable as a diff.
	commands := program.Commands.Commands()
	entries := make([]cli.Entry, 0, len(commands))
	for _, c := range commands {
		entries = append(entries, cli.Entry{Name: c.Name, Summary: c.Summary})
	}

	return cli.WriteCompletion(env.Stdout, env.Args[0], program.Name, entries)
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
