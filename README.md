<div align="center">

<img src="docs/logo.svg" alt="zlib" width="160"/>

# zlib

A CLI for Z-Library.

[![CI](https://img.shields.io/github/actions/workflow/status/heartleo/zlib/release.yml)](https://github.com/heartleo/zlib/actions)
[![Release](https://img.shields.io/github/v/release/heartleo/zlib)](https://github.com/heartleo/zlib/releases)
[![Downloads](https://img.shields.io/github/downloads/heartleo/zlib/total)](https://github.com/heartleo/zlib/releases)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

English | [中文](README.zh.md)

[Install](#install) · [Quick start](#quick-start) · [Agent setup](#agent-setup) · [Docs](#documentation)

</div>

Search and download books, browse saved books and recommendations, and send files
to Kindle. Use the terminal picker yourself, or let an agent use the CLI through JSON output.

![Interactive book search](docs/demo-search.gif)

## Install

macOS / Linux (Homebrew):

```bash
brew install heartleo/tap/zlib
```

Windows (PowerShell):

```powershell
winget install --id heartleo.zlib --exact
```

macOS / Linux (install script):

```bash
curl -fsSL https://raw.githubusercontent.com/heartleo/zlib/main/install.sh | sh
```

[Download a binary](https://github.com/heartleo/zlib/releases/latest) or
[install with Go / build from source](docs/installation.md).

## Quick start

You need a Z-Library account. Find a reachable domain first:

```sh
zlib doctor
```

Choose a domain marked `Available`. Replace the example below with that domain,
then log in and open the search picker. These commands work in Bash and PowerShell:

```sh
zlib login --domain https://z-lib.gd
zlib search
```

Enter a query, use `↑/↓` to select a book and `←/→` to change pages, then press
`Enter` to download. Files go to the current directory by default.

Login saves your session and domain for later commands. If login returns
`Authorization failed`, [import your browser cookies](docs/troubleshooting.md#login-returns-authorization-failed).
If `doctor` finds no usable domain, see [connection troubleshooting](docs/troubleshooting.md#no-available-domains).

## Agent setup

Install the CLI and complete the login above in your own terminal first. The plugins
provide skills that run your local `zlib` executable and reuse its saved session.

### Claude Code

Run inside Claude Code:

```text
/plugin marketplace add heartleo/zlib
/plugin install zlib@zlib-plugin-cc
```

Then use `/zlib "the pragmatic programmer"` to search, or `/zlib:kindle` to send a local file.

### Codex

Run in your terminal:

```sh
codex plugin marketplace add heartleo/zlib
codex plugin add zlib@zlib
```

Ask: “Use zlib to find The Pragmatic Programmer in EPUB. Show me the results first.”

[Agent workflows and JSON examples](docs/agents.md) · [Kindle setup](docs/kindle.md)

The shared skills live in [`.agents/skills/`](.agents/skills) for repo-level use.

## Commands

| Task | Command |
| --- | --- |
| Search with a terminal picker | `zlib search` |
| Print search results | `zlib search "dune" --ext epub` |
| Get results as JSON | `zlib search "dune" --json --count 10` |
| Browse saved books | `zlib favorites` |
| Browse recommendations / popular books | `zlib recommended` / `zlib most-popular` |
| Browse download history | `zlib history` |
| View account and daily quota | `zlib profile` |
| Send a local file to Kindle | `zlib kindle send ./book.epub` |
| Change the theme | `zlib theme nord` |

To download a specific result, copy its full `id:hash` reference and run
`zlib download "id:hash"`. See [search and download workflows](docs/usage.md)
for pagination, download folders and keyboard controls.

Use `zlib --help` or `zlib <command> --help` for all flags.

## Documentation

| Guide | Contents |
| --- | --- |
| [Installation](docs/installation.md) | Binary downloads, Go, source builds and upgrades |
| [Usage](docs/usage.md) | Search, downloads, saved books, history and themes |
| [Configuration](docs/configuration.md) | Domains, proxies, download folders and settings files |
| [Agents](docs/agents.md) | Plugin setup, non-interactive commands and JSON output |
| [Kindle](docs/kindle.md) | SMTP setup and file delivery |
| [Troubleshooting](docs/troubleshooting.md) | Login, connection and download failures |
| [EAPI notes](docs/eapi.md) | Supported endpoints and API limitations (Chinese) |

Report bugs in [Issues](https://github.com/heartleo/zlib/issues), including your
`zlib version`, OS, command and error message. Omit passwords, cookies and session files.
