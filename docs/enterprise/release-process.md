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

Steps 6–8 are **not automated yet**: this repository has no release workflow and
has never cut a tag, so signing the tag, building the artifacts and publishing the
GitHub Release are manual. Until that is automated (a change that needs its own
ADR), this section is a checklist rather than a description of tooling. Every
other step is a human responsibility, recorded in the release PR description.

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
