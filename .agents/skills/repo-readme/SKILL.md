---
name: repo-readme
description: Author, audit, and maintain world-class GitHub open-source repository READMEs (bilingual EN/ZH), aligning codebase capabilities, visual identity, CLI/TUI showcases, and documentation matrices.
license: MIT
---

# Repository README Authoring & Maintenance Skill

This skill provides an end-to-end framework to author, audit, enhance, and synchronize GitHub open-source repository READMEs (`README.md` and `README.zh.md`). It incorporates design standards from trending CLI/TUI projects (e.g., Charm's Bubble Tea, Starship, Lazygit, Fzf, Ripgrep, Ast-Grep) and establishes strict bilingual fidelity and codebase alignment.

---

## 1. Core Principles: The "Golden Standard" of Open-Source READMEs

Top-tier open-source developer tools succeed on GitHub by delivering clarity within the first **30 seconds** of visit:

1. **Immediate Visual Hook (< 5s)**: A clean, high-resolution branding header and an authentic, high-contrast animated demo (GIF/asciinema/SVG) showing the tool in real action.
2. **Crystal-Clear Value Proposition (< 10s)**: A concise one-sentence elevator pitch, followed by 4–6 bulleted feature highlights emphasizing speed, ergonomics, and reliability.
3. **Zero-Friction Quick Start (< 30s)**: Copy-pasteable commands to install and see immediate output without getting bogged down in configuration.
4. **Two-Tier Usage Layout**:
   - **Interactive Mode**: Visual guidance for humans navigating the TUI/Shell (keybindings, hotkeys, commands).
   - **CLI / Scripting Mode**: Direct commands with structured outputs (e.g., `--json`) for power users, CI/CD, and pipelines.
5. **AI Agent & Ecosystem Ready**: Modern tooling highlights first-class agent workflows (Claude Code, Codex, Cursor, Antigravity, MCP servers, and local skill packages).
6. **Strict Bilingual Parity (EN / ZH)**: English serves as the international default; Chinese (`README.zh.md`) maintains identical structure, code blocks, anchor points, and badge sets without colloquial degradation or missing sections.

---

## 2. End-to-End Workflow

```
┌─────────────────────────┐
│ 1. Codebase & Git Delta │ ──> Discover new subcommands, flags, TUI screens,
│        Discovery        │     env vars, and architectural changes.
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ 2. Existing Doc Audit   │ ──> Cross-check docs/ directory, verify single-source-
│     & Gap Analysis      │     of-truth, find stale flags or dead links.
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ 3. Architecture Layout  │ ──> Construct the standard 10-block README structure.
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ 4. Bilingual Drafting   │ ──> Draft README.md and README.zh.md with 1:1
│     & Parity Sync       │     codeblock and anchor correspondence.
└───────────┬─────────────┘
            ▼
┌─────────────────────────┐
│ 5. Verification Check   │ ──> Validate markdown links, code snippets, badge
│     (Pre-flight)        │     URLs, and run plugin sync scripts.
└─────────────────────────┘
```

### Phase 1: Codebase & Git Delta Discovery
Before modifying or generating a README, run the following probes in the workspace:
1. **Recent Commits & Feature Branches**: Check `git log -n 15 --oneline` and `git diff --stat origin/main..HEAD` to spot recent additions (e.g., new TUI shell, theme picker, auto-update command, custom download dirs).
2. **CLI Entry Points & Flags**: Inspect `internal/cli/*.go` or `cmd/` to catalog all subcommands, flags, defaults, and aliases.
3. **Configuration & Environment**: Identify supported environment variables (e.g., `ZLIB_DOWNLOAD_DIR`, `ZLIB_THEME`, `ZLIB_DOMAIN`, `HTTP_PROXY`) and config file locations across OSes (`~/.config/zlib/` or `%APPDATA%\zlib\`).
4. **Interactive Capabilities**: Check for TUI frameworks (Bubble Tea, Lip Gloss) and note screen flows, keybindings (`↑/↓`, `←/→`, `Ctrl+K`), and slash commands (`/search`, `/favorites`, `/theme`).

### Phase 2: Document Audit & Gap Analysis
Review existing documentation in `docs/`:
- Ensure the main README stays high-level and punchy, delegating exhaustive details to dedicated `docs/*.md` files (e.g., `docs/installation.md`, `docs/usage.md`, `docs/tui.md`, `docs/themes.md`, `docs/agents.md`, `docs/troubleshooting.md`).
- Eliminate documentation drift: if a flag was renamed or deprecated in code, verify it is updated across all markdown files.

### Phase 3: The Standard 10-Block Information Architecture

Every production README should adhere to this modular 10-block layout:

| Block | Section | Key Elements & Best Practices |
| :--- | :--- | :--- |
| **1** | **Hero Header** | Centered `<div align="center">`, SVG logo (120–160px width), Project Title, Punchy 1-sentence tagline, curated badge row, language switcher (`English \| [中文](README.zh.md)`), and anchor navigation pills. |
| **2** | **Visual Showcase** | Prominent demo GIF / SVG tape / asciinema preview showing the terminal interface in action right below the hero. |
| **3** | **Key Features** | 4–6 standout bullet points with clear emojis (e.g., 🖥️ Interactive TUI Shell, ⚡ Native JSON EAPI, 📱 Kindle Delivery, 🎨 20+ Polished Themes, 🤖 Native AI Agent & MCP). |
| **4** | **30-Second Quick Start** | Immediate path to joy: check network reachability (`doctor`), log in once (`login`), and launch the interactive shell (`tui`). |
| **5** | **Installation Matrix** | Organized platform blocks: macOS / Linux (Homebrew), Windows (Winget), One-line curl install script, pre-built GitHub releases, and Go source build. Mention self-update (`zlib update`). |
| **6** | **Usage & Workflows** | Split into **Interactive TUI Shell** (keybindings, slash commands) and **CLI & Scripting Reference** (Markdown table matching Task → Command → Notes). |
| **7** | **AI Agent & MCP Integration** | How AI assistants (Claude Code, Codex, Antigravity, OpenCode) leverage the tool via plugins, `.agents/skills/`, and headless `--json` flags. |
| **8** | **Configuration & Environment** | Table of config paths per OS and recognized environment variables (`ZLIB_DOWNLOAD_DIR`, etc.). |
| **9** | **Documentation Hub** | Clean table deep-linking to dedicated topic guides in `docs/`. |
| **10** | **Footer & Community** | Issue reporting etiquette, Responsible use / Legal disclaimer, License badge, and author credits. |

---

## 3. Bilingual Parity Standards (EN / ZH)

Maintaining bilingual documentation requires strict adherence to parity rules:

1. **Structural Mirroring**: Every heading `##` and `###` in `README.md` must have a direct counterpart in `README.zh.md` at the same hierarchy level.
2. **Anchor Synchronization**: Keep internal cross-links compatible. Ensure language switchers at the top navigate between `README.md` and `README.zh.md`.
3. **Code & Command Consistency**: All code snippets, flags, command names, and output shapes must be 100% identical between both files. Do NOT translate command line tokens, flag names (`--json`, `--count`), or JSON keys.
4. **Natural, Idiomatic Localization**:
   - Avoid mechanical machine translation. Use authentic Chinese developer terminology (e.g., "开箱即用", "交互式终端", "会话复用", "配额", "排障").
   - Maintain uniform spacing between Chinese characters and English words/code tokens (e.g., `使用 zlib search 搜索图书` 而不是 `使用zlib search搜索图书`).
5. **Table Alignment**: Markdown table columns, order, and links must remain identical across language editions.

---

## 4. Project-Specific Domain Rules: Z-Library CLI (`zlib`)

When updating `zlib`'s README:
- **EAPI Architecture Emphasis**: Emphasize that `zlib` uses official JSON EAPI, bypassing brittle HTML scraping, proof-of-work challenges, and Cloudflare/DiamWall interstitials.
- **TUI & Themes**: Highlight the rich Lip Gloss themes (`nord`, `dracula`, `catppuccin-mocha`, `gruvbox`) and the `/` slash-command shell.
- **Kindle Integration**: Highlight direct 1-click delivery via SMTP (`zlib kindle send`).
- **Security & Privacy**: Never display real passwords, authorization cookies, SESSDATA, or personal token hashes in README snippets or error examples. Always use placeholders (`https://z-lib.gd`, `user@example.com`, `id:hash`).
- **Disclaimer**: Keep the educational/interoperability disclaimer prominent to clarify that `zlib` is an independent open-source client.

---

## 5. Supporting References

- [Style & Badge Guide](references/style-guide.md): Markdown formatting, badge design, typography, and visual rules.
- [English README Template](references/template-en.md): Complete, ready-to-use template for `README.md`.
- [Chinese README Template](references/template-zh.md): Complete, ready-to-use template for `README.zh.md`.
- [Pre-Flight Checklist](references/checklist.md): Step-by-step verification checklist before submitting README changes.
