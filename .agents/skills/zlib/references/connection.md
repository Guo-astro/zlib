# Login and connection recovery

Use this reference after a login, session, domain or network error. Use CLI error
messages and exit status; keep session cookies and passwords out of tool output.

## Missing or expired session

Run `zlib doctor --json` to obtain available candidates. It neither logs in nor
changes the saved domain. Its JSON uses `status: "healthy"`; the human table
labels those entries `Available`. An empty array means no available candidate
was found on this network.

Give the user a concrete login command using a returned domain:

```sh
zlib login --domain https://z-lib.gd
```

The address above is an example, not a guaranteed working domain. Ask the user to
complete the interactive login in their own terminal, then resume with
`zlib profile --json`. The CLI saves the session and domain; the agent does not
need to read `session.json` or receive the password.

If login returns `Authorization failed`, explain that some accounts require
importing an existing browser session. The user can export their own Netscape
Cookie file and run this locally, using the matching domain:

```sh
zlib login --cookies ./cookies.txt --domain https://z-lib.gd
```

Keep the exported file on the user's machine. Do not ask for its contents in chat.
For `Incorrect email or password`, have the user recheck credentials in the
interactive prompt. Do not retry a refused login across every domain.

## Domain or network failure

Run `zlib doctor --json` once. If the user has supplied a proxy, test it with
`zlib doctor --proxy "proxy-url" --json`; avoid placing proxy credentials in a
command or chat. Do not replace the configured domain merely because another
candidate is available.

Domain configuration precedence is process environment, working-directory `.env`,
then `~/.config/zlib/.env`. Authenticated commands use the saved session domain
when `ZLIB_DOMAIN` is absent or blank. Login's `--domain` overrides the domain for
that login and updates the global `.env`, but a stale higher-priority assignment
can still affect later commands.

If configuration needs editing, identify the applicable file and explain the
change. Read or edit only the needed assignment; never dump `.env` or session
contents. Preserve other settings. When no candidate works, report the failed
connectivity check and suggest checking the network or configured proxy.

## Missing command or JSON flag

Inspect `zlib version` and `zlib <command> --help`. Explain that the installed
binary needs an update if it lacks the requested command. Do not fall back to
HTML scraping, raw authenticated requests, or bare interactive commands.
Features in a development checkout may require building that checkout rather
than installing the latest published release.
