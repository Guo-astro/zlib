# English README Template (`README.md`)

Use this template as the standard architectural baseline for the English `README.md`.

```markdown
<div align="center">

<img src="docs/logo.svg" alt="zlib" width="160"/>

# zlib

Command-line client and terminal UI for Z-Library.

[![CI](https://img.shields.io/github/actions/workflow/status/heartleo/zlib/release.yml)](https://github.com/heartleo/zlib/actions)
[![Release](https://img.shields.io/github/v/release/heartleo/zlib)](https://github.com/heartleo/zlib/releases)
[![Downloads](https://img.shields.io/github/downloads/heartleo/zlib/total)](https://github.com/heartleo/zlib/releases)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

English | [中文](README.zh.md)

[Features](#features) · [Install](#install) · [Quick start](#quick-start) · [Commands](#commands) · [Agent setup](#agent-setup) · [Docs](#documentation)

</div>

Search and download books, browse saved collections and recommendations, and send files to Kindle. Run the interactive terminal UI, or pipe JSON output to scripts and AI agents.

![Interactive book search](docs/demo-search.gif)

## Features

- **Terminal UI**: Bubble Tea shell with live search, book cards (ISBN, publisher, year, format), responsive layouts, and slash commands (`/search`, `/favorites`, `/theme`, `/doctor`, `/config`).
- **Direct EAPI**: Talks to Z-Library's JSON API directly. No HTML scraping, browser challenge solving, or DiamWall blocks.
- **Kindle delivery**: Built-in SMTP delivery to send downloaded books straight to your Kindle.
- **Themes**: Over 20 built-in dark and light themes (Catppuccin, Dracula, Nord, Gruvbox, Tokyo Night, Solarized).
- **Agent skills**: Bundled skills for Claude Code and Codex, plus machine-readable `--json` across all query commands.
- **Network doctor**: Probes official mirrors in parallel to find working domains and test latency.

## Install

### Package managers

macOS / Linux (Homebrew):

```bash
brew install heartleo/tap/zlib
```

Windows (winget):

```powershell
winget install --id heartleo.zlib --exact
```

### Install script

macOS / Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/heartleo/zlib/main/install.sh | sh
```

### Pre-built binaries

Download archives for macOS, Linux, and Windows (x86_64, ARM64) from [Releases](https://github.com/heartleo/zlib/releases/latest). Extract the binary into a directory on your `PATH`.

### Go install

Requires Go 1.26 or later:

```bash
go install github.com/heartleo/zlib/cmd/zlib@latest
```

Ensure `$GOPATH/bin` (or `%USERPROFILE%\go\bin` on Windows) is in your `PATH`.

### Build from source

Requires Go:

```bash
git clone https://github.com/heartleo/zlib.git
cd zlib
go build -o zlib ./cmd/zlib
```

On Windows (PowerShell):

```powershell
go build -o zlib.exe ./cmd/zlib
```

Run `zlib update` to update to the latest release using your detected installation method. For checksum verification and build flags, see [Installation](docs/installation.md).

## Quick start

You need a Z-Library account.

1. Find a reachable domain:
   ```bash
   zlib doctor
   ```

2. Log in using a domain marked `Available` (saves session for future commands):
   ```bash
   zlib login --domain https://z-lib.gd
   ```
   If Cloudflare blocks login, [import your browser cookies](docs/troubleshooting.md#login-returns-authorization-failed).

3. Launch the terminal shell:
   ```bash
   zlib tui
   ```
   - Type a title and press `Enter` to search.
   - Use `↑/↓` to select a book, `←/→` to switch pages, and `Enter` to download.
   - Type `/` to open the command palette (`/theme`, `/favorites`, `/doctor`), or `/quit` to exit.

## Commands

| Goal | Command | Output / Notes |
| --- | --- | --- |
| Open interactive shell | `zlib tui` | Fullscreen terminal UI |
| Search with terminal picker | `zlib search` | Interactive selection |
| Search books | `zlib search "dune" --ext epub` | Plain text table |
| Search books as JSON | `zlib search "dune" --json --count 10` | JSON object (`books`, `page`, `total_pages`) |
| Download by ID | `zlib download "id:hash"` | Saves file locally |
| Browse saved books | `zlib favorites` | Interactive list or `--json` |
| Browse recommendations | `zlib recommended` | Curated books |
| Browse popular books | `zlib most-popular` | Top downloaded books |
| Browse download history | `zlib history` | Paginated download log |
| Check quota and profile | `zlib profile` | Daily limit and account status |
| Send book to Kindle | `zlib kindle send ./book.epub` | Sends via configured SMTP |
| Change theme | `zlib theme nord` | Sets active palette (`zlib theme` for picker) |
| Probe domains | `zlib doctor` | Latency and availability check |
| Update binary | `zlib update` | Detects package manager and updates |

Run `zlib <command> --help` to view all available flags. See [Usage](docs/usage.md) for pagination, custom directories, and filters.

## Agent setup

The CLI supports non-interactive automation via `--json`. The repository also bundles local skills in [`.agents/skills/`](.agents/skills/).

### Claude Code

Run inside Claude Code:

```text
/plugin marketplace add heartleo/zlib
/plugin install zlib@zlib-plugin-cc
```

Use `/zlib "the pragmatic programmer"` to search, or `/zlib:kindle ./book.epub` to deliver a file.

### Codex

Run in your terminal:

```bash
codex plugin marketplace add heartleo/zlib
codex plugin add zlib@zlib
```

Ask Codex: "Use zlib to find The Pragmatic Programmer in EPUB and show the results before downloading."

See [Agents](docs/agents.md) for non-interactive scripting and JSON formats.

## Configuration

| Environment variable | Description | Default |
| --- | --- | --- |
| `ZLIB_DOWNLOAD_DIR` | Directory for downloaded books | Current working directory |
| `ZLIB_THEME` | Color theme for current process | `catppuccin-mocha` |
| `ZLIB_DOMAIN` | Override target mirror domain | Saved domain from login |
| `HTTP_PROXY` / `HTTPS_PROXY` | HTTP or SOCKS5 proxy URL | None |

Configuration paths:
- macOS / Linux: `~/.config/zlib/`
- Windows: `%APPDATA%\zlib\`

See [Configuration](docs/configuration.md) for details on proxy setup and config files.

## Documentation

| Guide | Description |
| --- | --- |
| [Installation](docs/installation.md) | Package managers, Go install, source builds, upgrades |
| [Usage](docs/usage.md) | Search filters, downloads, favorites, history |
| [Terminal interface](docs/tui.md) | Shell commands, layout, keyboard shortcuts |
| [Themes](docs/themes.md) | Palette list and preview controls |
| [Configuration](docs/configuration.md) | Mirrors, proxies, directories, config files |
| [Agents](docs/agents.md) | Plugins, skills, non-interactive commands, JSON shapes |
| [Kindle](docs/kindle.md) | SMTP setup and file delivery |
| [Troubleshooting](docs/troubleshooting.md) | Login, domain availability, network recovery |
| [EAPI notes](docs/eapi.md) | API endpoints and known limits (Chinese) |

## Contributing

Report bugs in [Issues](https://github.com/heartleo/zlib/issues). Include your `zlib version`, operating system, command, and error output. Do not share cookies, passwords, or session files.
```
