# Versioning Policy — Kroot

## Scheme

`vMAJOR.MINOR.PATCH` — [Semantic Versioning 2.0.0](https://semver.org/).

While `MAJOR` is `0`, the public surface may change in a MINOR release. From
`v1.0.0` onward, the guarantees below are binding.

| Bump | When |
|---|---|
| **MAJOR** | breaking change to a released CLI flag, HTTP endpoint, `KROOT_*` variable, on-disk format or supported platform |
| **MINOR** | new backwards-compatible capability: new command, new endpoint, new optional flag, new configuration variable with a safe default |
| **PATCH** | backwards-compatible fix: bug fix, security fix, documentation-only change, dependency bump with no behaviour change |

A single breaking change forces a MAJOR bump. There is no "small" breaking
change.

## Compatibility guarantees

Once `v1.0.0` ships:

| Surface | Guarantee |
|---|---|
| **CLI** | flags and subcommands are additive within a MAJOR; removed flags go through a deprecation MINOR and print a warning before the next MAJOR |
| **HTTP API** | additive within a MAJOR; deprecations announced one MINOR ahead with `Deprecation` and `Sunset` headers |
| **Environment variables** | never renamed or repurposed within a MAJOR; new variables always ship with a default |
| **Exit codes** | stable within a MAJOR; `0` success, `1` runtime failure, `2` usage error |
| **Log records** | `log/slog` message text is not a stable interface; the structured keys used by operators are |
| **On-disk state** (`~/.kroot`) | migrations are automatic and forward-compatible; a breaking migration requires a MAJOR and a documented rollback |

This policy is elaborated in [`api-compatibility.md`](./api-compatibility.md).

## Pre-releases

- `vX.Y.Z-rc.N` for release candidates.
- `vX.Y.Z-alpha.N` / `vX.Y.Z-beta.N` for earlier previews.
- Pre-releases publish as GitHub pre-releases and are never selected by default
  by an installer; they require explicit pinning.

## Build metadata

The binary reports the exact revision it was built from:

```sh
kroot version        # e.g. "kroot 0.1.0 (commit 9a1d3f2, built 2026-09-26T19:00:00Z, go1.23.12)"
```

Version, commit and build time are injected at link time
(`-ldflags "-X main.version=… -X main.commit=… -X main.buildTime=…"`), never
hardcoded.
