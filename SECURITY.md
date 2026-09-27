# Security — Kroot

> **Do not open a public issue for a suspected vulnerability.**

Email **`sr.klosraf@gmail.com`** with a subject starting with
`[kroot-security]`. The report reaches only the maintainer, and it is acted on
before anything is published.

<!--
The reporting channel is constrained by the repository's visibility:

- Private repository (current): GitHub's private vulnerability reporting is not
  available — the API returns 404 unless the account has GitHub Advanced
  Security — so email is the channel. Repository issues are not an alternative:
  although they are collaborator-only, a disclosure belongs in a private
  conversation, not in a tracker.
- Public repository: enable GitHub private vulnerability reporting
  (Settings -> Code security) and make it the primary channel, keeping the email
  as the fallback.
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

## Security best practices for operators

- Run the latest release, and apply PATCH releases within 14 days.
- Never commit `.env` files or credentials; supply configuration through
  `KROOT_*` variables or a secret manager.
- Restrict network exposure: bind to loopback or a private interface unless a
  reverse proxy with TLS and authentication sits in front.
- Monitor logs for unexpected authentication or configuration activity.

## For maintainers

- `govulncheck ./...` gates every PR that touches Go code.
- Dependencies are reviewed on addition, not only on upgrade.
- Secrets never enter the repository; leaked credentials are rotated
  immediately, never reused.
