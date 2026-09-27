# Branching Strategy — Kroot

## Model: trunk-based with short-lived branches

- `main` is always releasable and protected: required CI, linear history,
  squash-merge only.
- Work happens in short-lived branches cut from an up-to-date `main`.
- Branches live for days, not weeks. A long-lived branch is a merge conflict
  that has not happened yet.

| Prefix | Use | Example |
|---|---|---|
| `feat/` | new capability | `feat/healthz-probe` |
| `fix/` | bug fix | `fix/config-parse-error` |
| `chore/` | tooling, dependencies, CI | `chore/pin-golangci-lint` |
| `docs/` | documentation only | `docs/enterprise-policies` |
| `test/` | tests only | `test/cover-stream-decoder` |
| `refactor/` | behaviour-preserving restructuring | `refactor/split-http-layer` |
| `release/` | release preparation | `release/v0.1.0` |
| `hotfix/` | emergency production fix | `hotfix/v0.1.1` |

## Pull request rules

1. **One concern per PR.** A reviewer should be able to describe the change in
   one sentence.
2. **Reviewable size.** Aim for under ~500 changed lines. A larger diff needs a
   stated reason in the description.
3. **Conventional title**, scoped to the affected area:
   `feat(server): add readiness probe`, `fix(cli): stop swallowing parse errors`.
4. **Complete PR body**: what and why, test evidence, docs updated when the CLI,
   HTTP API or `KROOT_*` variables change, changelog entry for user-visible
   changes.
5. **Green CI.** Every gate in `AGENTS.md` §7 must pass. A red gate is not
   "probably fine".
6. **At least one approval** from a code owner, on the final revision.
7. **Squash-merge.** The PR history does not need to be clean; the merged commit
   does. The squash subject becomes the permanent record.
8. **Never merge `main` into a feature branch.** Rebase instead, so history stays
   linear and bisectable.

## Protected branch settings for `main`

| Setting | Value |
|---|---|
| Require pull request | yes, at least 1 approval |
| Dismiss stale approvals on push | yes |
| Require status checks | `ci` workflow (all jobs) |
| Require linear history | yes |
| Allow force push | no |
| Allow deletion | no |

## Hotfixes

1. Cut `hotfix/vX.Y.Z` from the tag of the affected release.
2. Fix **and** add the regression test in the same PR.
3. Merge into `main`, tag `vX.Y.Z`, publish.
4. Cherry-pick forward into the current development line if it diverged.
5. Post-mortem within five business days: what changed, what the guard is now.
