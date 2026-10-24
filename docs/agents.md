# Agents and scripts

[README](../README.md) · [Plugin installation](../README.md#agent-setup)

The shared [zlib](../.agents/skills/zlib/SKILL.md)
and [Kindle](../.agents/skills/kindle/SKILL.md) skills live in `.agents/skills/`.
The Claude Code and Codex plugins bundle these skills. They call the locally
installed CLI through the agent's shell; there is no separate API server to start.

Codex [discovers `.agents/skills/`](https://learn.chatgpt.com/docs/build-skills#where-codex-loads-local-skills)
when working in this repository. Other clients
can load or install the skill folders using their supported Agent Skills mechanism;
the [format specification](https://agentskills.io/specification) does not prescribe
one shared discovery path. The plugin
installation in the README remains available for use outside this checkout.

## Prepare the agent's environment

If `zlib` is not on the agent's `PATH`, the skills install it with winget,
Homebrew or the install script. Complete [login](../README.md#quick-start) in
your own terminal. Check:

```sh
zlib version
zlib profile --json
```

If the agent runs in a container, remote machine or another OS user account, it
needs its own accessible CLI and session. The plugins do not bundle the binary or
copy your session into another environment. Keep passwords and exported cookies
out of chat; complete authentication locally.

After [installing the plugin](../README.md#agent-setup), try:

> Use zlib to find The Pragmatic Programmer in EPUB. Show the title, author,
> year and file size before downloading.

For Kindle delivery, complete [SMTP setup](kindle.md) first, then provide the local
file path when asking the agent to send it.

## Use non-interactive commands

Use these non-interactive forms:

```sh
zlib search "the pragmatic programmer" --json --count 10
zlib favorites --json --page 1
zlib history --json --page 1
zlib recommended --json
zlib most-popular --json
zlib profile --json
zlib doctor --json
```

Pass a query to `search --json`. Start searches with `--count 10` to limit the
response size and use `--page 2` for more results. Recommendations and popular
books do not have pagination flags.

Parse the complete JSON document. The search response has `books`, `page` and
`total_pages`; each book's `id` is the full reference accepted by `download`.
Do not truncate raw JSON or parse the display table.

Saved books use the same paginated shape as search. History uses `items` instead
of `books`; recommendations and popular books return arrays. An empty page is
still JSON (`"items": []` for history), so report any output that does not parse
as a failure.

Profile's `quota_known` indicates whether the quota fields are available. When it
is false, a zero `daily_remaining` does not mean the account has exhausted its quota.

PowerShell:

```powershell
$result = zlib search "dune" --json --count 10 | ConvertFrom-Json
$result.books | Select-Object id, name, authors, year, extension, size
```

Bash with `jq` installed:

```bash
zlib search "dune" --json --count 10 |
  jq '.books[] | {id, name, authors, year, extension, size}'
```

Copy the selected book's `id` without dropping its hash. Replace `id:hash` below
with that value and use an existing destination directory:

```sh
zlib download "id:hash" --dir ./books --json
```

`download --json` prints one document, `{"id","name","path","size"}`, with the
absolute `path` of the saved file; add `--send-to-kindle` and it gains a
`kindle` object (`sent`, `to` or `error`). Progress lines go to stderr, so
read stdout only.

Branch on the [exit code](usage.md#exit-codes) rather than on message text: `3`
means the session needs `zlib login`, `4` a network or mirror problem
(`zlib doctor`), `5` a saved book whose Kindle delivery failed, `130` a
cancellation.

`zlib tui`, `zlib search` without a query, bare `history`, and `kindle send`
without a file open interactive prompts. Bare `zlib` prints help. Use the JSON
forms above for lists and an explicit file path for delivery:

```sh
zlib kindle send ./books/book.epub
```

`profile --json` contains personal account information, though it omits
authentication keys. Handle that output as private account data.

## Model Context Protocol (MCP) server

`zlib mcp` runs a stdio MCP server exposing Z-Library tools with structured
inputs and outputs. It uses the same local session (`~/.config/zlib/session.json`)
and `.env` settings as the CLI.

### MCP quick start

1. Confirm `zlib` is installed and signed in:

   ```sh
   zlib version
   zlib mcp --help
   zlib profile --json
   ```

2. Register and verify the server in Claude Code or Codex:

   ```sh
   # Claude Code
   claude mcp add zlib -- zlib mcp
   claude mcp get zlib

   # Codex
   codex mcp add zlib -- zlib mcp
   codex mcp get zlib
   ```

   For read-only access (disables `download_book`, `delete_download_history`, and
   `send_to_kindle`), pass `--read-only`:

   ```sh
   claude mcp add zlib -- zlib mcp --read-only
   codex mcp add zlib -- zlib mcp --read-only
   ```

   To allow downloads without email or history deletion, register with
   `zlib mcp --disable-kindle --disable-history-delete` instead. The separate
   `--disable-download` flag hides only the download tool. `--disable-kindle`
   also blocks `download_book.send_to_kindle`; `--read-only` overrides all write
   capabilities.

   To keep downloads and Kindle attachments inside one folder, add
   `--allowed-dir ~/Books`. Paths outside it, including through symlinks or
   Windows junctions, are rejected. Downloads without a `dir` go to
   `ZLIB_DOWNLOAD_DIR` when it lies inside that folder, otherwise to the folder
   itself. `--log-level` (`off`, `error`, `warn`
   by default, `info`, `debug`) writes tool names, durations and failures to
   stderr, never arguments.

   You can also configure Codex in `~/.codex/config.toml`:

   ```toml
   [mcp_servers.zlib]
   command = "zlib"
   args = ["mcp"]
   ```

3. Use it in an interactive session or headless run:

   > Use the zlib MCP tools to check my remaining daily downloads, search for
   > "Dune" in EPUB, and show the first 5 matches with IDs and sizes.

   Non-interactive runs:

   ```sh
   codex exec "Use zlib MCP to search for 'Dune' in EPUB and list the top 3 IDs."
   claude -p "Use mcp__zlib__search_books to find 'Dune' in EPUB and list the top 3 IDs." --allowedTools "mcp__zlib__search_books"
   ```

### Permissions and results

Downloads can replace same-named files. Kindle delivery reads a local file and
emails it to the configured recipient. Require client approval for these actions
and for history deletion; tool annotations are hints, not authorization.
Read-only mode still exposes account details and history. Without
`--allowed-dir`, the server can write and read wherever the local user can.
`check_domains` takes no proxy argument; it uses `ZLIB_PROXY` when set.

MCP search accepts `count` from 1 to 50 (default 10) and pages starting at 1.
Year filters accept 1000 through 9999, or 0 to leave a bound unset. Invalid
arguments are tool errors that name the argument and its allowed values.
History metadata failures return an error instead of an incomplete filtered
list. Restart the server after changing environment or proxy settings; account
sessions are reread on each call.

Book lists return `id`, `name`, `authors`, `publisher`, `year`, `language`,
`extension`, `size` and `rating`; `get_book` adds ISBNs, edition, pages,
categories and description. Error messages show only the scheme and host of any
URL, so signed download links and proxy credentials stay out of transcripts.

Cancelling a tool stops its HTTP and SMTP work. A completed file remains saved,
and mail already accepted by the SMTP server cannot be recalled. A failed Kindle
step after a download retains the file path in the result; do not download again
just to retry delivery.

### Available tools

| Tool | Mode | Description |
| --- | --- | --- |
| `search_books` | Read | Search books with format, language, year, and order filters |
| `get_book` | Read | Fetch book metadata and description by `id:hash` |
| `get_recommended_books` | Read | Personalized recommendations for the signed-in account |
| `get_most_popular_books` | Read | Most popular books on Z-Library |
| `get_saved_books` | Read | Paged favorite books |
| `get_download_history` | Read | Paged download history with optional format filter |
| `get_profile` | Read | Account profile and daily quota |
| `check_domains` | Read | Probe known mirrors to find reachable domains |
| `download_book` | Write | Download a book by `id:hash` with MCP progress notifications |
| `delete_download_history` | Write | Remove an entry from remote history by ID or `id:hash` |
| `send_to_kindle` | Write | Send a local ebook file to Kindle via SMTP |

## Maintain the skills

Edit `.agents/skills/<name>/SKILL.md` and its `references/` files. Run
`node scripts/sync-plugin-skills.mjs` to generate `plugins/zlib/skills/` for both
plugin packages. Only the plugin skills listed in the script (`zlib`, `kindle`)
are copied; the other folders in `.agents/skills/` are repository-level skills
for maintaining this repo and are not part of the plugin. `node scripts/sync-plugin-skills.mjs --check` verifies the bundle
without changing files; CI runs that check on every pull request.

The generated copies are committed so plugin installations work without Node.js
or a build step. Avoid editing them directly. If a source file is removed, delete
the corresponding obsolete bundle file before synchronizing.

## Writing

Use the `no-ai-slop` skill for every repository document and user-facing
description, including READMEs, skill instructions, CLI help, and TUI copy.
Name commands, data, and behavior directly. Check claims against the current
code and say when something is unknown. Cut filler, hype, repeated explanations,
and decorative headings. Apply the same standard to English and Chinese text.
