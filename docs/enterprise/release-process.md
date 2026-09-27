# Release Process — Kroot

## Cadence

- **PATCH** releases on demand, for fixes and security issues.
- **MINOR** releases when a coherent set of capabilities is complete and
  documented.
- **MAJOR** releases are planned work with a migration guide, never a surprise.

No release ships without a green `main` and a completed checklist.

## Version determination

Pick the bump per [`versioning-policy.md`](./versioning-policy.md) by inspecting
the merged commits since the last tag. If a commit is ambiguous, treat it as
user-visible.

```sh
git log --oneline "$(git describe --tags --abbrev=0)"..HEAD
```

> This command does not run today: `git describe` cannot reach the `v0.1.0` tag
> from any branch. See *The `v0.1.0` tag does not describe this history* below for
> the command to use instead, and for why the tag is not simply moved onto the
> commit that actually reached `main`.

## Release checklist

1. **Scope** — confirm the release contains only reviewed, merged, green commits.
2. **Freeze** — cut `release/vX.Y.Z` from `main`; no new features land in the
   branch.
3. **Changelog** — promote `## [Unreleased]` to `## [vX.Y.Z] - YYYY-MM-DD` in
   `CHANGELOG.md`, grouping entries under Added / Changed / Deprecated / Removed /
   Fixed / Security. Open a fresh empty `## [Unreleased]` section above it.
4. **Docs** — every CLI, HTTP API and `KROOT_*` change is documented in this
   repository, in the same release.
5. **Verification** — `make ci` plus `make vuln` green on the release branch, and
   the integration suite run once manually.
6. **Tag** — annotated, signed tag on the release-branch head:
   `git tag -s vX.Y.Z -m "kroot vX.Y.Z"`.
7. **Artifacts** — CI builds the release binaries, attaches checksums
   (`SHA256SUMS`), an SBOM (`spdx-json`) and the changelog excerpt.
8. **Publish** — GitHub Release, marked pre-release for `rc`/`alpha`/`beta`.
9. **Announce** — release notes link the changelog entry and the migration guide
   when the release is MAJOR.
10. **Merge back** — the release branch merges into `main`; bump the development
    version if the project tracks one.

Steps 6–8 are **not automated yet**: this repository has no release workflow, so
signing the tag, building the artifacts and publishing the GitHub Release are
manual. Until that is automated (a change that needs its own ADR), this section is
a checklist rather than a description of tooling. Every other step is a human
responsibility, recorded in the release PR description.

That was written when no release had been cut. `v0.1.0` has since been tagged,
built and published as a GitHub Release, by hand, following this list — so the
absence of automation is established by a release that shipped without it.

## The `v0.1.0` tag does not describe this history

`v0.1.0` points at `044d233` (*chore(release): prepare v0.1.0*), a commit that is
**not an ancestor of any branch**. The same work reached `main` as `285971c`
(*chore(release): prepare v0.1.0 (#21)*), the squash-merge of the release pull
request, and the tag was not moved onto it.

Three things follow, and all three are still true until the next release:

- `git describe --tags` fails on every branch with *"No tags can describe …"*, so
  the documented step 1 below cannot be run as written.
- The `VERSION` the `Makefile` injects falls back to the abbreviated commit — a
  build is labelled `b1332f0-dirty` rather than a version — so `kroot version`
  reports a commit hash as if it were a release.
- A release cut from this history would be the first whose `git describe` output
  is meaningful, because the new tag *would* be reachable.

**The tag is published, and it stays.** The rollback rules below forbid deleting
or moving a published tag, and rewriting it to repair tooling would turn a
bookkeeping mistake into a history rewrite for anyone who fetched it. The repair
is forward: the next release cuts a correct tag on a release-branch head, which
gives `git describe` a reachable anchor, and the step below is amended to say so
while the gap is open.

**Until then, use this instead of the command in step 1**, which needs no tag:

```sh
git log --oneline "$(git merge-base HEAD main)"..HEAD   # commits on this branch
git log --oneline -1 main                             # what is merged
```

## Artifacts

| Artifact | Purpose |
|---|---|
| `kroot_<version>_<os>_<arch>` binaries | direct download |
| `SHA256SUMS` | integrity verification |
| `sbom.spdx.json` | software bill of materials |
| `CHANGELOG` excerpt | release notes body |

Builds are reproducible: fixed toolchain version, pinned dependencies, no
network access at build time beyond the module proxy.

## Rollback

- **Before publish** — delete the tag, fix, re-run the checklist. Nothing shipped.
- **After publish** — never delete a published tag or re-release the same
  version. Ship `vX.Y.Z+1` that reverts the change, and mark the broken release
  as such in its GitHub Release notes.
- **Security regression** — follow [`security-policy.md`](./security-policy.md)
  for coordinated disclosure and the advisory.
