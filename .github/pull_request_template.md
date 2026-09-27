<!--
PR titles follow Conventional Commits, e.g.
  feat(server): add readiness probe
  fix(cli): stop swallowing the config parse error
Use a scope. See docs/enterprise/branching-strategy.md.
-->

## What

<!-- One paragraph a reviewer can read without opening the diff. -->

## Why

<!-- The problem being solved. Link the issue: Closes #123 -->

## How

<!-- The approach, and any alternative you rejected and why. -->

## Test evidence

<!--
Paste the exact commands you ran and their results. For a bug fix, confirm the
regression test fails without the fix and passes with it.
-->

```
$ make ci
```

## Checklist

- [ ] `make ci` is green locally (fmt-check, vet, lint, test-race, vuln, build)
- [ ] New or changed behaviour is covered by tests
- [ ] Bug fixes include a regression test that failed before the fix
- [ ] Documentation updated if CLI, HTTP API or `KROOT_*` variables changed
- [ ] `CHANGELOG.md` updated under `## [Unreleased]` for user-visible changes
- [ ] Public behaviour change is backwards compatible, or the versioning policy
      was applied (`docs/enterprise/versioning-policy.md`)
- [ ] An ADR was added for any decision with long-term consequences
- [ ] No secrets, credentials or `.env` values in the diff
- [ ] Diff stays reviewable (aim for under ~500 changed lines; explain if larger)

## Out of scope

<!-- Related work deliberately deferred, so reviewers know it was considered. -->
