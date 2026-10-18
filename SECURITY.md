# Security policy

## Reporting a vulnerability

Report vulnerabilities privately through GitHub:
[Report a vulnerability](https://github.com/heartleo/zlib/security/advisories/new).
Do not open a public issue, discussion, or comment with the details.

Include the affected version (`zlib version`) or commit, the steps to
reproduce, and the impact you observed. Use fake credentials in reproductions.
Never send real session cookies (`remix_userid`, `remix_userkey`), passwords,
or SMTP credentials; redact them from logs and config files.

Reports are handled by the maintainer on a best-effort basis. You will get a
reply in the advisory thread, and a fix is coordinated there before the
details are published.

## Supported versions

Fixes land on `main` and ship in the next release. Only the latest release is
supported; update with `zlib update`.

## Scope

In scope, for example:

- Session cookies or other credentials sent to a host outside the domain
  allowlist, including through redirects.
- Downloaded files written outside the chosen directory, or existing files
  overwritten or deleted unexpectedly.
- Secrets in `~/.config/zlib/` exposed by the CLI, logs, or `--json` output.
- `zlib update` installing something other than an official release.
- The agent plugin skills running commands the user did not ask for.

Out of scope:

- The Z-Library service, its mirrors, and their availability.
- Bypassing DiamWall or other bot protection.
- Attacks that require control of the user's account or local machine.
