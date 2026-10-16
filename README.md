<div align="center">

<img src="docs/logo.svg" alt="zlib" width="160"/>

# zlib

Search and download Z-Library books from your terminal.

[![CI](https://img.shields.io/github/actions/workflow/status/heartleo/zlib/ci.yml)](https://github.com/heartleo/zlib/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/heartleo/zlib)](https://github.com/heartleo/zlib/releases/latest)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

English | [简体中文](README.zh.md)

[Install](#install) · [Quick start](#quick-start) · [Commands](#commands) · [Agent setup](#agent-setup) · [Docs](#documentation)

</div>

![Book search in the terminal](docs/demo-search.gif)

- Browse search results, favorites, recommendations, popular books, and download history.
- Inspect book details, choose a download folder, or [send a file to Kindle](docs/kindle.md).
- Use the full-screen interface or JSON commands for scripts and agents.

## Install

macOS / Linux with Homebrew:

```bash
brew install heartleo/tap/zlib
```

Windows with winget:

```powershell
winget install --id heartleo.zlib --exact
```

Or download a binary from [Releases](https://github.com/heartleo/zlib/releases/latest)
and put it on your `PATH`. See [Installation](docs/installation.md) for the install
script, Go installation, source builds, and upgrades.

If your installed version lacks a command shown here, see the
[source-build instructions](docs/installation.md#build-from-source) for this branch.

## Quick start

You need a Z-Library account. First, find a domain reachable from your network:

```sh
zlib doctor
```

Log in with a domain from the results. Replace the example URL below with one
from your results; enter your email and password in the prompt:

```sh
zlib login --domain https://z-lib.gd
zlib tui
```

In the terminal interface:

- Type a title, author, or ISBN and press `Enter` to search.
- Use `↑/↓` to select a book and `←/→` to change pages. `Enter` opens download confirmation.
- Type `/` for commands. Use `/config` for network, download, and Kindle settings,
  `/theme` for colors, and `/quit` to exit.

Downloads default to the current directory. You can choose another folder before
downloading. Login is saved locally; subsequent sessions only need `zlib tui`.

If no domain works or login fails, see [Troubleshooting](docs/troubleshooting.md).
Existing browser sessions can be [imported as cookies](docs/troubleshooting.md#login-returns-authorization-failed).

## Commands

Pass a query for table output, or add `--json` for scripts:

```sh
zlib search "dune" --ext epub
zlib search "dune" --ext epub --json --count 10
```

| Task | Command |
| --- | --- |
| Browse favorites | `zlib favorites` |
| Browse recommendations | `zlib recommended` |
| Browse popular books | `zlib most-popular` |
| Browse download history | `zlib history` |
| Check account and quota | `zlib profile` |
| Send a file after [Kindle setup](docs/kindle.md) | `zlib kindle send ./book.epub` |

To download from a script, replace `id:hash` with a book's complete `id` from
the JSON results:

```sh
zlib download "id:hash"
```

Run `zlib <command> --help` for flags. [Usage](docs/usage.md) covers filters,
pagination, and download options.

## Agent setup

Install zlib and complete the quick start in the same environment your agent uses.
The plugins provide skills; they require the CLI on `PATH` and a local login.

### Claude Code

Run inside Claude Code:

```text
/plugin marketplace add heartleo/zlib
/plugin install zlib@zlib-plugin-cc
```

### Codex

Run in your terminal:

```sh
codex plugin marketplace add heartleo/zlib
codex plugin add zlib@zlib
```

Then ask your agent:

> Use zlib to find The Pragmatic Programmer in EPUB. Show the results before downloading.

Skills live in [`.agents/skills/`](.agents/skills/). See [Agents and scripts](docs/agents.md)
for JSON formats, local skill discovery, and automation examples.

## Documentation

| Guide | Covers |
| --- | --- |
| [Installation](docs/installation.md) | Install methods, source builds, `zlib update` |
| [Usage](docs/usage.md) | Search, downloads, favorites, history |
| [Terminal interface](docs/tui.md) | Commands and keyboard controls |
| [Configuration](docs/configuration.md) | Domains, proxies, folders, configuration paths |
| [Themes](docs/themes.md) | Available palettes and preview controls |
| [Kindle](docs/kindle.md) | SMTP setup and delivery |
| [Agents and scripts](docs/agents.md) | Plugins, skills, JSON output |
| [Troubleshooting](docs/troubleshooting.md) | Login and network problems |
| [EAPI notes](docs/eapi.md) | API behavior and known limits (Chinese) |

Report bugs in [Issues](https://github.com/heartleo/zlib/issues) with your
`zlib version`, OS, command, and error. Omit passwords, cookies, and session files.
