# Configuration

[README](../README.md) · [Troubleshooting](troubleshooting.md)

zlib reads settings in this order, with the first value taking precedence:

1. Process environment variables.
2. `.env` in the working directory.
3. `~/.config/zlib/.env`.

Use the global `.env` for settings you want in every terminal. On Windows, `~`
is your user profile directory, usually `C:\Users\<name>`.

## Settings

### Theme

Run `zlib theme` to open the theme list. Move with ↑/↓ to preview
search results, format colors, and download status. Enter saves;
Esc or Ctrl+C cancels. Wide terminals show the preview beside the list;
narrow terminals show it below. PgUp/PgDn and Home/End navigate the list.

The default theme is `catppuccin-mocha`. Legacy `auto` values are treated as unset.
`zlib theme <name>` sets a theme directly; without a terminal, `zlib theme` prints the current
theme. `ZLIB_THEME` overrides the preference saved in `~/.config/zlib/config.json`.

### Interactive setup

```sh
zlib config
zlib config network
zlib config network domain
zlib config network proxy
zlib config download
zlib config download directory
zlib config download retries
zlib config kindle
zlib config kindle to
zlib config kindle from
zlib config kindle host
zlib config kindle port
```

`zlib config` shows network, download and Kindle settings together. The
subcommands open the same TUI at a specific section or field. Use arrow keys
or Tab to select, Enter to edit, and Esc to return to the selected setting.
The settings list keeps its controls and save feedback next to the fields;
it shows an override notice only when another source supplies the active value.
The domain page automatically checks candidates using the same EAPI probe as
`doctor`; `r` refreshes results.
Enter saves the selected domain. An empty result leaves your configuration intact.

Editors have Save, Cancel and Reset actions, reached with Tab. Enter in the
input saves the value. Reset clears the global value after you save; higher
priority environment settings still apply. Unsaved changes can be discarded
when returning to the parent page.
For Kindle, Reset clears an address or restores the default SMTP host or port.

- **Proxy:** enter a URL; Ctrl+T tests it without saving. The editor shows the full URL. Settings lists hide embedded credentials.
  Clearing the global proxy returns to existing environment proxy behavior.
- **Directory:** enter a path or use Browse (Ctrl+B). In the directory browser,
  Enter opens a directory and Space selects the current directory. Selection
  fills the editor; Save persists it. Directories must exist; relative paths
  are saved as absolute paths. Enter a drive path to switch Windows drives.
- **Retries:** enter a non-negative integer. Zero disables retries; the default
  is three.

Network and download changes are saved to `~/.config/zlib/.env`, preserving unrelated settings.
Kindle address and SMTP server settings are saved to `~/.config/zlib/kindle.json`.
The UI reports when process or working-directory settings override a saved
network or download value. It does not change login cookies. If the SMTP password
is missing when you send to Kindle in the TUI, a masked form saves it to the
global `.env` and sets `ZLIB_SMTP_PWD` for the running process. It is not stored
in `kindle.json` or shown in the settings list.

Without a terminal, these commands print a read-only summary. Add `--json` for
values, sources and saved preferences; proxy credentials are redacted.

```sh
zlib config --json
zlib config network proxy --json
```

| Variable | Purpose | Default |
| --- | --- | --- |
| `ZLIB_DOMAIN` | EAPI domain | Saved session domain for authenticated commands |
| `ZLIB_PROXY` | HTTP, HTTPS or SOCKS proxy URL | Standard `HTTP_PROXY` / `HTTPS_PROXY` environment settings |
| `ZLIB_DOWNLOAD_DIR` | Download destination when `--dir` is omitted | Current directory |
| `ZLIB_DOWNLOAD_RETRIES` | Retries for interrupted downloads and HTTP 502/503/504 | `3` |
| `ZLIB_THEME` | Override the saved terminal theme | Saved preference |
| `ZLIB_SMTP_PWD` | SMTP password for Kindle delivery | Unset |

See [.env.example](../.env.example) for sample values.

## Create or edit the global file

macOS / Linux:

```bash
mkdir -p ~/.config/zlib
touch ~/.config/zlib/.env
chmod 600 ~/.config/zlib/.env
${EDITOR:-nano} ~/.config/zlib/.env
```

Windows (PowerShell):

```powershell
$zlibConfigDir = Join-Path $env:USERPROFILE '.config\zlib'
New-Item -ItemType Directory -Force -Path $zlibConfigDir | Out-Null
$zlibEnvFile = Join-Path $zlibConfigDir '.env'
if (!(Test-Path -LiteralPath $zlibEnvFile)) {
    New-Item -ItemType File -Path $zlibEnvFile | Out-Null
}
notepad $zlibEnvFile
```

Enter one `KEY=value` per line. You can also enter `ZLIB_SMTP_PWD` in the masked
TUI prompt before sending to Kindle. The prompt writes it to this file for
future runs. Use an editor to set it before a non-interactive send.

## Domains and proxies

Run `zlib doctor` to find an available domain, then use `zlib login --domain URL`.
New logins require either `--domain` or `ZLIB_DOMAIN`; login checks the domain
before asking for credentials.

A successful login saves the session and writes its domain to the global `.env`.
An existing process variable or working-directory `.env` can still override that
value. When `ZLIB_DOMAIN` is unset or blank, authenticated commands use the domain
stored in the session.

Domains must be HTTPS origins accepted by the CLI's allowlist. See
[EAPI domain rules](eapi.md#域名白名单) for the supported hosts.

To test a proxy for one request:

```sh
zlib doctor --proxy http://127.0.0.1:7890
```

To use it for subsequent commands, add `ZLIB_PROXY=http://127.0.0.1:7890` to
the global `.env`. The doctor's `--proxy` flag affects only that check.

## Download settings

Add this line to the global `.env` to use an existing Downloads directory:

```dotenv
ZLIB_DOWNLOAD_DIR=~/Downloads
```

An explicit `--dir` overrides the setting; `--dir .` selects the current
directory for that invocation. Both forms expand a leading `~`, and neither
creates the destination directory.

## Saved files

These files live under `~/.config/zlib/`:

| File | Contents |
| --- | --- |
| `.env` | Environment settings and any secrets you add |
| `session.json` | Login cookies and domain |
| `config.json` | Theme and last successful login email |
| `kindle.json` | Kindle recipient, sender and SMTP server settings |

Login does not save your password. Session cookies provide account access, so
keep `session.json` and exported Cookie files private. The Kindle configuration
file does not contain the SMTP password; the CLI reads it from `ZLIB_SMTP_PWD`.

Interactive downloads show the destination before starting. Choose **Skip** (the default) or press Esc to keep it. Choose **Change directory** to enter a path; Ctrl+B opens the directory browser, Enter opens a folder, and Space selects it. The change applies only to that download. Ctrl+C cancels. An explicit `--dir` or non-interactive execution skips this prompt. Without `--dir` or `ZLIB_DOWNLOAD_DIR`, downloads still use the working directory.
