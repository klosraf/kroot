# Security — Kroot

> **Do not open a public issue for a suspected vulnerability.**

Email **`sr.klosraf@gmail.com`** with a subject starting with
`[kroot-security]`. The report reaches only the maintainer, and it is acted on
before anything is published.

<!--
The reporting channel is constrained by the repository's visibility.

This repository is **public**. GitHub's private vulnerability reporting is
therefore available — the API answers with `{"enabled": false}` rather than the
404 a private repository returns — but it is **not currently enabled**, so email
is the working channel today. Enabling it (Settings -> Code security ->
"Enable private vulnerability reporting") is the intended change, and email stays
as the fallback either way.

Repository issues are not an alternative at any visibility: although they are
collaborator-only, a disclosure belongs in a private conversation, not a tracker.
-->

Include the same information listed below, whichever channel you use.

## Reporting a vulnerability

Include as much of the following as you can:

- A description of the vulnerability and the affected component.
- Exact steps to reproduce, ideally a minimal proof of concept.
- Your assessment of the impact and the likely attack path.
- Any mitigations or workarounds you have identified.
- Affected version (`kroot version`) and platform (`uname -a`).

What to expect:

| Stage | Target |
|---|---|
| Acknowledgement of your report | 2 business days |
| Initial assessment and severity triage | 5 business days |
| Fix for critical and high findings | next PATCH release |
| Public disclosure | coordinated with you, after the fix ships |

We credit reporters in the release notes unless you prefer otherwise.

## Supported versions

Only the latest MINOR of the current MAJOR receives security fixes. Older MINORs
are upgraded rather than patched.

## Security characteristics of kroot

Stated so that operators reason about the actual program rather than about a
server. kroot is a terminal-first CLI
([ADR-0003](https://github.com/klosraf/kroot/blob/main/docs/adr/0003-cli-first-terminal-application.md)),
so several risks that are routine for a networked service do not apply to it:

- **No network surface.** The binary opens no sockets and speaks no protocol. It
  is not a server and must not be run as one.
- **No authentication or authorisation**, because it has no remote peer to
  authenticate. There is nothing to protect with a credential.
- **No persistence yet.** Nothing is written under `~/.kroot`, so there is no
  on-disk state to protect, migrate or roll back. This changes when persistence
  lands, under its own ADR.
- **Configuration is environment-only.** `KROOT_*` variables, never files in the
  working tree, so a checkout carries no credentials to leak.

## Security best practices for operators

- Run the latest release, and apply PATCH releases within 14 days.
- Verify a downloaded binary against the `SHA256SUMS` published with the release.
- Supply configuration through `KROOT_*` variables or a secret manager. Never
  commit `.env` files or credentials — they are git-ignored, but a committed
  secret is a public one for the life of the tag.
- Prefer `KROOT_LOG_LEVEL=error` in automation. Log records go to **stderr**,
  human-readable on a terminal and JSON when redirected, so a supervisor capturing
  stderr is capturing structured records; treat that stream as potentially
  sensitive if you add configuration values to it later.
- An unrecognised `KROOT_LOG_LEVEL` is **rejected, not coerced**: the process
  stops with exit `1` before any command runs. A supervisor that swallows that
  exit code will hide a misconfiguration, so check it.
- Branch on the exit code — `0` success, `1` runtime failure, `2` usage error.
  It is the only part of the interface guaranteed stable within a MAJOR, and it
  distinguishes "you invoked me wrong" from "it broke while running".

## For maintainers

- `govulncheck ./...` runs in CI on **every** pull request, including
  documentation-only changes: the `vulnerabilities` job has no path filter. It
  also runs as part of `make ci` locally.
- Dependencies are reviewed on addition, not only on upgrade. kroot currently has
  **no third-party dependencies**; the standard library is the dependency.
- Secrets never enter the repository; leaked credentials are rotated
  immediately, never reused.

