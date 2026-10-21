# Troubleshooting

[README](../README.md) · [Configuration](configuration.md)

## No available domains

```sh
zlib doctor
```

`doctor` probes known domains using EAPI and lists usable results. When none is
reachable it exits with an error (and `--json` prints `[]`); check your network
or retry through a proxy as shown below. The focused
row shows your position; `Current domain` marks the configured domain. It does
not log in or change your configured domain. Availability
depends on your network and can change; use the current result rather than a
saved domain list. The website and EAPI may have different availability.

If your network uses a proxy, test through it:

```sh
zlib doctor --proxy http://127.0.0.1:7890
```

If that works, add `ZLIB_PROXY=http://127.0.0.1:7890` to your
[global `.env`](configuration.md#create-or-edit-the-global-file) so other commands
use the same proxy. Select an available domain and run `zlib login --domain URL`.

HTTP 517, redirect loops and HTML challenge responses can indicate anti-bot
protection. The CLI does not solve browser challenges. If no domain works, try
another network route or retry later.

## Login returns `Authorization failed`

Some accounts have been observed to receive this error from EAPI login while
an existing session still works for search and profile requests. It does not,
by itself, establish that your password is wrong.

Log in through your browser and export your own session cookies in Netscape/Mozilla
format. Import the file for the matching domain:

```sh
zlib login --cookies ./cookies.txt --domain https://z-lib.gd
```

Replace the example domain with the one matching your exported cookies. Import
accepts unexpired root-path cookies that match the selected domain. Keep both the
export and saved session private.

If the service reports `Incorrect email or password`, recheck your credentials.
Use the interactive login prompt so shell quoting cannot alter the password.

## Commands fail after login succeeds

A stale `ZLIB_DOMAIN` can override the domain saved by login. Check these locations
in order:

1. The process environment, including exports in your shell startup files.
2. `.env` in the directory where you run `zlib`.
3. `~/.config/zlib/.env`.

Update or remove stale assignments in your editor. Login updates only the global
`.env`; higher-priority settings still win. With no nonblank `ZLIB_DOMAIN`,
authenticated commands use the saved session domain.

Run `zlib doctor` again if the domain stopped working. HTTP 503 can be temporary;
a saved login does not guarantee that a domain remains reachable.

## Not logged in or session expired

Commands exit with code 3 and say whether there is no session or the server
rejected the saved one (`session expired or was revoked`). Run
`zlib login --domain URL` in your terminal, choosing a domain from `doctor`.
If password login is refused, use the [Cookie import procedure](#login-returns-authorization-failed).

Agents need access to the same user environment and session. See
[agent setup](agents.md#prepare-the-agents-environment) if the CLI works in your
terminal but fails inside an agent.

## Downloads fail

Use the complete `id:hash` reference from current search or history results.
Old short IDs from earlier HTML-based versions are not EAPI references.

Check `zlib profile` for the remaining daily quota and confirm that the download
directory exists and is writable. Downloads default to `ZLIB_DOWNLOAD_DIR` when
set, otherwise the current directory; `--dir` overrides both.

For interrupted connections and HTTP 502/503/504, the downloader retries up to
`ZLIB_DOWNLOAD_RETRIES` times (default `3`). If failures continue, check the domain
and network before retrying the download. For mail failures, see [Kindle delivery](kindle.md#if-delivery-fails).

## Missing command or flag

Run `zlib version` and `zlib --help`. Upgrade using your
[installation method](installation.md#verify-and-upgrade). Features documented on
a development branch may require building that branch from source.

For a bug report, include the version, OS, command and error message in an
[issue](https://github.com/heartleo/zlib/issues). Remove passwords, Cookie headers,
session files and signed download URLs from the report.
