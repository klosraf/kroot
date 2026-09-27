# 0002 — Keep macOS verification off pull requests

- **Status:** accepted
- **Date:** 2026-09-27
- **Deciders:** repository maintainers

## Context

The `test` job ran the matrix `[ubuntu-latest, macos-latest]` on every event,
including every push to every pull request.

Two facts made that unsustainable:

1. **macOS runners bill at a 10x rate multiplier** on private repositories, and
   kroot is private. A macOS job that takes three minutes costs the equivalent of
   thirty Linux minutes.
2. **The account ran out of included Actions minutes.** Measured through the
   billing API for January–September 2026, the account consumed 40,009 Linux,
   6,794 Windows and 5,801 macOS 3-core minutes — roughly 111,600
   billable-equivalent minutes, about 12,400 per month against the 3,000 per month
   included with GitHub Pro. kroot's own usage was a rounding error in that total,
   but it was consuming the same exhausted pool.

The consequence was not a slow job or a warning. GitHub stopped starting jobs at
all, in every repository on the account, and the failure it produced is
indistinguishable from a code failure in the API:

```text
The job was not started because recent account payments have failed or your
spending limit needs to be increased. Please check the 'Billing & plans'
section in your settings
```

Every check in the repository went red for twelve consecutive runs while the code
was unchanged — the same commit that passed at 03:05 failed at 03:07 — and no
gate, log or annotation pointed at the actual cause. Diagnosing that cost more
than the minutes the macOS job was saving.

The documented policy is unaffected by the platform question:
`docs/enterprise/testing-strategy.md` requires `go vet`, `gofmt -l`,
`go test -race -cover` and `govulncheck` on **every pull request**, and none of
those requirements is platform-specific. macOS was in the matrix because of a
single inline comment ("the project targets it for local use"), not because any
policy demanded it per pull request.

## Decision

1. **Run Linux on every event and macOS only on a push to `main`.** Merging *is*
   a push, so the platform is still verified before release; it is verified once
   per merge instead of once per commit.
2. **Keep the job name stable.** `test (ubuntu-latest)` continues to be reported
   on pull requests, so the required status check keeps its exact context.
3. **Remove `test (macos-latest)` from the required status checks of `main`.**
   A required check that a pull request never reports does not block *the check*,
   it blocks *the pull request* — forever. `docs/enterprise/branching-strategy.md`
   records the same trap for the `commits` job, which is why that job reports
   success instead of being skipped on paths where linting does not apply.
4. **Record the decision here and update the policy documents** so the required
   check list quoted in `branching-strategy.md` matches what is actually enforced.

## Consequences

**Positive**

- Per-pull-request cost drops by the entire macOS share: the largest multiplier
  applied to the event that happens most often.
- One platform's availability can no longer stop every gate in the repository for
  an entire account, at least not through this job.
- The macOS job is unchanged — same steps, same command, same runner. Only the set
  of events that triggers it shrank, which keeps one definition of the test steps
  instead of duplicating them into a second job.

**Negative / costs**

- A regression that affects only macOS is now found at the merge to `main`
  instead of in the pull request that introduced it. The blast radius is one
  revert, not a release.
- Two CI configurations now exist by construction (PR vs push), so the matrix
  expression is the single place where cross-platform behaviour can drift.
- The intended behaviour of the matrix expression cannot be verified locally; it
  is confirmed the first time a pull request and a merge run it.

**Neutral**

- Nothing about the toolchain, the Go version or the test suite changes. The
  `ubuntu-latest` job still runs `gofmt`, `go vet`, `go test -race` and
  `govulncheck` on every pull request, as `testing-strategy.md` requires.

## Alternatives rejected

- **`paths-ignore` so documentation-only pull requests skip CI.** The same trap
  in a different costume: a skipped required check leaves the pull request
  blocked, so a docs-only PR could never merge.
- **Dropping macOS entirely.** The project targets macOS for local use and
  publishes a desktop build from this repository, so the platform has to be
  verified somewhere. Merges are the cheapest place that still counts.
- **Self-hosted macOS runners.** Removes the per-minute cost but adds a machine,
  a security boundary and an availability dependency that a single-maintainer
  project cannot justify yet.
- **Making the gate advisory instead of removing it from the required set.**
  Configured-but-required checks fail silently: the UI reports "Expected" with no
  explanation, which is the same diagnosis tax that motivated this ADR.

## Notes

- Revisit when `web/` lands: a Vite build has more platform-visible behaviour
  than a standard-library CLI, and the trade-off may tilt back toward per-PR
  verification.
- Revisit when a second maintainer joins and review capacity allows pair review of
  platform-specific changes.
- This ADR does not weaken the compatibility guarantees in
  `docs/enterprise/api-compatibility.md`; it changes where a check runs, not what
  a release is allowed to contain.
