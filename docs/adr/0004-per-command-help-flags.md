# 0004 — A command answers `-h` and `--help` with its own help

- **Status:** accepted
- **Date:** 2026-09-27
- **Deciders:** repository maintainers

## Context

Three spellings already mean "tell me about this command", and they do not agree:

- `kroot help version` — the documented spelling, and the only one that answers
  with the command's help.
- `kroot version -h` — answered with the **program's** help, because the global
  `flag.FlagSet` owns `-h` and prints its own usage.
- `kroot version --help` — before the surplus-operand fix it printed the **version**
  and exited `0`; after it, it is a usage error.

A caller who reaches for `-h` after a command is doing what every other tool does.
Answering with a different tool's help, or with an error, costs a reader a trip to
the manual for something the binary could have said. The surface is still cheap to
change while `MAJOR` is `0`, but the behaviour must be decided rather than
inherited: `api-compatibility.md` makes subcommand names and flags documented in the
manual stable from `v1.0.0`, and help wording and layout explicitly not.

## Decision

1. **`-h` and `--help`, as the first operand of a command, print that command's
   help and exit `0`.** They are routed through the help command rather than a
   second renderer, so `kroot version --help` and `kroot help version` print the same
   bytes, tolerate a failed write the same way, and exit alike.
2. **Only the first operand counts.** Callers write flags before operands, so
   `kroot version extra --help` stays a usage error: after the first operand, `-h` is
   data, and silently accepting data is exactly what the surplus-operand rule
   closed.
3. **Anything after the help operand is not examined.** `kroot version --help extra`
   prints the command's help, mirroring the global flag set, which answers `-h`
   before it looks at the rest of the command line.
4. **The program keeps its own `-h`.** `kroot -h` and `kroot --help` continue to
   print the program's help through `flag.ErrHelp`; the global flag set is
   unchanged.
5. **No new flag names, and no manual change.** The canonical spelling
   `kroot help <command>` is already documented in the manual; this is an alias for
   it, recorded in the README and here. Adding `-h` to the manual as a per-command
   flag would promise a per-command flag set, which does not exist.

## Consequences

**Positive**

- Every spelling a reader already tries gives the same answer, from the same code
  path, with the same exit code.
- The alias costs one branch: dispatch has already resolved the command, and the
  help command does the rest, so the alias cannot drift from `kroot help <command>`.
- Help stays exit `0`: the caller asked for text and received it.

**Negative / costs**

- `kroot version --help extra` stops reporting the extra operand, because the help
  request wins. That is a deliberate, documented exception to "nothing is silently
  accepted": no command ran, so no command accepted anything.

**Neutral**

- No dependency and no new exported surface: the helper lives in `main`, where the
  dispatch does.

## Alternatives rejected

- **Per-command `flag.FlagSet`s.** The right answer for a product with real command
  options; kroot has none, and a flag set each would be machinery with one user.
- **Rejecting `--help` after a command.** It answers a reasonable request with an
  error, and the error would have to explain a spelling the caller chose
  deliberately.
- **Supporting only `-h` or only `--help`.** The two differ in habit alone, and
  accepting one and not the other is a trap with no payoff.
- **Adding the alias to every `Command.Usage`.** The usage line documents the
  invocation the command owns; the alias belongs to the framework, and four copies
  are four places to keep in step.

## Notes

- Behaviour is pinned in `TestRun`: `-h`, `--help`, the help operand followed by
  more operands, and the help operand in a non-first position.
- Revisit if a command ever accepts flags of its own: the precedence question then
  becomes real and needs its own record.
