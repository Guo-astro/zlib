# Open-Source README Style & Badge Guide

This guide establishes visual and typographical standards for open-source CLI/TUI repositories, drawing from top projects like Charmbracelet (Bubble Tea, Gum), Starship, Lazygit, Fzf, and Ripgrep.

---

## 1. Hero Header & Visual Hierarchy

The hero section creates the initial impression. Keep it centered, balanced, and free from cluttered text.

### Header Template
```html
<div align="center">

<img src="docs/logo.svg" alt="Project Logo" width="140"/>

# Project Name

**A modern, blazing-fast terminal client for [Service/Tool].**

[![CI Status](https://img.shields.io/github/actions/workflow/status/<owner>/<repo>/ci.yml?branch=main&style=flat-square&logo=github)](https://github.com/<owner>/<repo>/actions)
[![Latest Release](https://img.shields.io/github/v/release/<owner>/<repo>?style=flat-square&color=blue)](https://github.com/<owner>/<repo>/releases)
[![Downloads](https://img.shields.io/github/downloads/<owner>/<repo>/total?style=flat-square&color=orange)](https://github.com/<owner>/<repo>/releases)
[![Go Report](https://goreportcard.com/badge/github.com/<owner>/<repo>?style=flat-square)](https://goreportcard.com/report/github.com/<owner>/<repo>)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE)

English | [简体中文](README.zh.md)

[Features](#features) · [Installation](#installation) · [Quick Start](#quick-start) · [AI Agents](#ai-agent-integration) · [Docs](#documentation)

</div>
```

### Badge Design Rules
1. **Style Consistency**: Use a consistent badge style across the entire row (e.g., `style=flat-square` or standard flat). Do not mix rounded pills with square badges.
2. **Standard Badge Set**:
   - Build / CI status (`github/actions/workflow/status`)
   - Latest Release version (`github/v/release`)
   - Total or monthly downloads (`github/downloads/.../total`)
   - Quality / Linter score (e.g. Go Report Card for Go projects)
   - License (`badge/License-MIT-green`)
3. **Limit to 4–6 Badges**: Avoid walls of 15 badges that wrap onto multiple rows on mobile devices.
4. **Valid Links**: Every badge MUST be wrapped in a hyperlink pointing to the corresponding action, release, or license file.

---

## 2. Media & Demonstrations

A top-tier CLI README must include authentic media showing the product in action:
- **Format**: Optimized `.gif`, animated `.svg` (e.g., via `vhs` or `asciinema`), or high-DPI screenshots.
- **Placement**: Directly below the hero navigation pills and above the Features / Installation sections.
- **Alt Text**: Always provide meaningful `alt` text (e.g., `alt="Interactive book search and download"`).
- **Size Optimization**: Keep GIF sizes under 3–5MB so page load remains instantaneous globally.

---

## 3. Typography & Formatting Rules

### Emojis in Section Headers & Bullets
- Use clean, tasteful emojis to facilitate quick visual scanning.
- Recommended palette:
  - ⚡ Speed / Performance
  - 🖥️ Terminal / TUI / Shell
  - 📱 Hardware / Devices (e.g. Kindle)
  - 🎨 Aesthetics / Themes / Colors
  - 🤖 AI / Agents / Automation
  - 🩺 Diagnostics / Health / Doctor
  - 📦 Installation / Package Managers
  - ⚙️ Configuration / Settings
  - 📖 Documentation / Guides

### Code Blocks & Shell Syntax
- Always specify syntax highlighters:
  - `bash` or `sh` for POSIX commands.
  - `powershell` for Windows commands.
  - `json` for API payload examples.
  - `go`, `python`, etc., for library usage.
- Multi-line commands should use backslashes (`\`) for Bash or backticks (`` ` ``) for PowerShell only when clarity demands it; otherwise prefer single-line copy-pasteable commands.

### Markdown Tables
- Keep tables compact and readable.
- Column headers should be concise (`Task`, `Command`, `Description`).
- Backtick inline code for commands, flags, and paths.

---

## 4. Bilingual Localization Conventions (EN ⟷ ZH)

| Category | English Rule | Chinese Rule |
| :--- | :--- | :--- |
| **Command Names** | In backticks: `zlib search` | In backticks: `zlib search` (Never translated) |
| **Flag Names** | In backticks: `--json`, `--page` | In backticks: `--json`, `--page` (Never translated) |
| **Keybindings** | `Enter`, `Ctrl+K`, `↑/↓`, `Esc` | `Enter`、`Ctrl+K`、`↑/↓`、`Esc` (Punctuation adjusted) |
| **File Paths** | In backticks: `~/.config/zlib` | In backticks: `~/.config/zlib` |
| **Spacing** | Standard English spacing | Add spaces around English words and code spans within Chinese text (盘古之白规范) |
| **Tone** | Direct, active voice, imperative | 简洁、地道、积极祈使句，避免冗余生硬机翻 |
