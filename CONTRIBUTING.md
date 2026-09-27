# Contributing to Kroot

Thanks for your interest in Kroot. This document describes how to propose,
review and land a change. It is the operational companion to
[`AGENTS.md`](./AGENTS.md) (the engineering charter) and
[`docs/enterprise/`](./docs/enterprise/) (the policies).

## Before you start

- Read [`AGENTS.md`](./AGENTS.md). It is short and it is binding.
- Set up the toolchain:

  ```sh
  make tools      # install pinned dev tools into ./bin
  make ci         # confirm the tree is green before you touch it
  ```

## What lands easily

- Bug fixes with a regression test.
- Performance work with before/after measurements.
- Security fixes (report privately first — see [`SECURITY.md`](./SECURITY.md)).

## What needs a discussion first

Open an issue **before** writing code for:

- New features, new CLI flags, new `KROOT_*` variables, new endpoints.
- New dependencies.
- Refactors that touch more than a few files.
- Anything that changes behaviour users already depend on.

In the issue, explain the problem rather than the solution, why it matters, how
it will be used, and how it will be tested. Bonus points for a draft of the
documentation you would expect to ship with it.

## What is unlikely to be accepted

- Breaking changes to a released CLI, API or environment variable without a
  MAJOR bump.
- New dependencies that duplicate the standard library.
- Changes that add long-term maintenance burden for a short-term gain.
- Unbounded configuration knobs that exist for a single caller.

## Development workflow

```sh
git switch -c feat/my-change        # from an up-to-date main
# ... work ...
make ci                              # fmt-check + vet + lint + test-race + build
git commit                           # Conventional Commits, see below
```

`main` is protected. Never push to it directly, and never merge `main` into your
branch — rebase.

## Commit messages

Conventional Commits, with the affected package or area as the scope:

```
<type>(<scope>): <short description>
```

| Type | Use |
|---|---|
| `feat` | new user-visible capability |
| `fix` | bug fix |
| `docs` | documentation only |
| `style` | formatting only, no change in meaning |
| `refactor` | behaviour-preserving restructuring |
| `test` | tests only |
| `perf` | measurable performance work |
| `ci` | CI/CD configuration |
| `build` | build system or packaging |
| `chore` | tooling, dependencies, housekeeping |
| `revert` | revert of an earlier commit |

The description is imperative, lower-case, and has no trailing period.

```
feat(server): add /healthz readiness probe
fix(cli): stop swallowing the config parse error
docs(enterprise): record the persistence ADR
```

### Where messages are validated

Two validators run, and they must agree — a message accepted locally that CI
rejects is a defect in this repository, not in your commit.

| Where | What it enforces | Effect |
|---|---|---|
| Local `commit-msg` hook (`~/.config/git/hooks`) | ≤ 72-character subject; Conventional Commits shape | blocks on length, warns on format |
| `pnpm exec commitlint --edit <file>` | everything in `commitlint.config.mjs` | blocks |
| CI `commits` job | everything in `commitlint.config.mjs` | blocks the PR |

**CI is authoritative.** The local hook is deliberately lenient so that a
work-in-progress commit is not blocked mid-task; CI is where the convention is
actually enforced. To check a message before committing:

```sh
git commit -m "feat(cli): add a thing" --dry-run   # or:
printf 'feat(cli): add a thing\n' | pnpm exec commitlint
```

## Pull requests

Every PR includes:

1. **What and why** — the problem, not only the diff.
2. **Test evidence** — the exact command you ran and its result.
3. **Docs** — updated in the same PR when the CLI, API or env vars change.
4. **Changelog** — an entry under `## [Unreleased]` for user-visible changes.

Reviewers check that CI is green, that tests cover the new behaviour, and that
the change respects the charter. Address review comments with new commits; the
PR is squash-merged, so the branch history does not matter.

## Questions

Open an issue. There is no such thing as a question that is too basic — a
misunderstood requirement is more expensive than a short conversation.
