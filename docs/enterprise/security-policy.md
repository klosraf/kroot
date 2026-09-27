# Security Policy — Kroot

Canonical contact and disclosure process: [`SECURITY.md`](../../SECURITY.md).
This document is the operator-facing view: what the threat model is, and how to
deploy Kroot hardened.

## Threat model

Kroot is a locally executed service. The assets worth protecting are the
credentials in its configuration, the data it stores under `~/.kroot`, and the
integrity of whatever it is wired to.

| # | Threat | Mitigation |
|---|---|---|
| T1 | Credentials leak through source control or logs | `KROOT_*` variables and mounted files only; `.env` git-ignored; secrets never logged |
| T2 | Service exposed beyond the intended network | bind to loopback by default; exposure requires an explicit `KROOT_HOST` change |
| T3 | Untrusted input crosses a boundary unvalidated | validate and bound every external input; reject rather than coerce |
| T4 | Injection into SQL, shell or templates | parameterised queries; no shell interpolation; no dynamic template evaluation |
| T5 | Dependency compromise | pinned versions, `govulncheck` in CI, dependency review on addition |
| T6 | Path traversal via user-controlled paths | resolve and confine to an allow-listed root; never trust a relative path from a request |
| T7 | Denial of service through unbounded work | request size and timeout limits; bounded worker pools; rate limiting on exposed endpoints |
| T8 | Supply-chain tampering with release artifacts | signed tags, checksums and an SBOM attached to every release |

## Handling secrets

- Secrets arrive through environment variables or files mounted at runtime.
- Never commit a `.env`, a key or a token. `.env*` is git-ignored, and the
  exception is only `.env.example` with obvious placeholder values.
- Never log a secret, a token, an authorization header or a full request body in
  production. Logging those is a defect, not a debugging aid.
- A leaked credential is rotated immediately and never reused.

## Input handling

- Validate at the boundary, once, and pass typed values inward.
- Bound everything a caller controls: length, count, range, nesting depth.
- Reject unknown fields rather than ignoring them where a schema exists.
- Treat all external data as hostile: HTTP bodies, headers, files, environment.

## Hardening checklist (production)

- [ ] Bind to loopback or a private interface; never expose the service directly to the internet
- [ ] Terminate TLS at a reverse proxy and require authentication in front
- [ ] Supply configuration from a secret manager, not from a file in the repository
- [ ] Run as a non-root user, with a read-only root filesystem where possible
- [ ] Restrict `~/.kroot` to the service account (`0700`) and back it up encrypted
- [ ] Set request size, timeout and concurrency limits to values you can defend
- [ ] Ship logs to a retained, access-controlled destination
- [ ] Track releases and apply PATCH updates within 14 days

## CI security gates

| Gate | Scope |
|---|---|
| `govulncheck ./...` | every PR touching Go code |
| `golangci-lint` with `gosec` | every PR |
| Dependency review | every PR that modifies `go.mod` / `go.sum` |
| SBOM (`spdx-json`) | every GitHub Release |
| Signed tag | every GitHub Release |

## Secure defaults

A default must be the safe option. If an insecure setting is required for a
legitimate use case, it is opt-in, named explicitly, and documented with its
consequences.
