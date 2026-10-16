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

Install `zlib` on the `PATH` used by your agent and complete
[login](../README.md#quick-start) in your own terminal. Check:

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
of `books`; recommendations and popular books return arrays. Current empty history
pages print `No download history found on page N.` even with `--json`. Handle that
specific successful response as an empty page and report other parsing failures.

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
zlib download "id:hash" --dir ./books
```

`zlib tui`, `zlib search` without a query, bare `history`, and `kindle send`
without a file open interactive prompts. Bare `zlib` prints help. Use the JSON
forms above for lists and an explicit file path for delivery:

```sh
zlib kindle send ./books/book.epub
```

`profile --json` contains personal account information, though it omits
authentication keys. Handle that output as private account data.

## Maintain the skills

Edit `.agents/skills/<name>/SKILL.md` and its `references/` files. Run
`node scripts/sync-plugin-skills.mjs` to generate `plugins/zlib/skills/` for both
plugin packages. `node scripts/sync-plugin-skills.mjs --check` verifies the bundle
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
