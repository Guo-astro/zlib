# Search and download

[README](../README.md) · [Configuration](configuration.md) · [Troubleshooting](troubleshooting.md)

Complete the [quick start](../README.md#quick-start) first. Book commands use your
saved Z-Library session.

## Search

Open the terminal picker without a query:

```sh
zlib search
```

Enter a query, select with `↑/↓`, change pages with `←/→`, and press `Enter` to
download. For a plain table, supply the query on the command line:

```sh
zlib search "dune"
zlib search "dune" --page 2
zlib search "python crash course" --ext epub --ext pdf
zlib search "civilized to death" --full-title
```

`--ext` is repeatable; `--format` is its alias. `--full-title` prevents title
truncation in the table. Use `--json --count 10` for [scripts and agents](agents.md).

## Download a result

Copy the complete reference from a result's ID column or JSON `id` field.
EAPI references contain both the numeric ID and hash, separated by `:`.
Replace `id:hash` in these examples with that reference:

```sh
zlib download "id:hash"
zlib download "id:hash" --dir ./books
zlib download "id:hash" --dir "~/Downloads"
```

Create the destination directory first. `--dir` expands a leading `~`; without
the flag, zlib uses `ZLIB_DOWNLOAD_DIR`, then the current directory. See
[download settings](configuration.md#download-settings) to save a default.

Press `Ctrl+C` to cancel a download; the CLI removes the incomplete file.
After [Kindle setup](kindle.md), add `--send-to-kindle` to send the downloaded file.

## Saved books, recommendations and history

In the history browser, press `d` to delete the selected entry, then `y` to
confirm or `n` / `Esc` to cancel. This removes the remote history entry only;
local book files remain. Scripts can delete directly with
`zlib history --delete "id:hash"` (also accepts a numeric ID); add `--json`
for a machine-readable success response.

```sh
zlib saved
zlib favorites
zlib recommended
zlib most-popular
zlib history
```

`saved` and `favorites` show the same account collection. In the terminal picker,
select with `↑/↓` and press `Enter` to download. Saved books and history support
`←/→` pagination; recommendations and popular books are single lists.

For non-interactive output, use an explicit page or JSON:

```sh
zlib saved --page 2
zlib favorites --json
zlib recommended --json
zlib most-popular --json
zlib history --page 1
zlib history --format epub
zlib history --json
```

To download a history entry directly, copy its full book reference:

```sh
zlib history --download "id:hash" --dir ./books
```

## Account and appearance

```sh
zlib profile
zlib profile --json
zlib logout
```

`profile` shows account details and daily download quota. JSON output excludes
authentication keys but includes personal account fields; keep it private.
`logout` removes the saved session.

```sh
zlib theme
zlib theme mocha
zlib theme nord
```

Themes: `mocha` (default), `latte`, `dracula`, `tokyo`, `nord`, `gruvbox`.
`zlib theme` opens a list with the current theme selected. Use ↑/↓ to choose,
Enter to save, or Esc to cancel. With redirected output, it prints the current
theme instead. `ZLIB_THEME` overrides
the saved theme.

See [Terminal interface](tui.md) for interactive controls. For exact flags and
defaults, run `zlib <command> --help`.
