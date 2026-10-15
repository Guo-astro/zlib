# README Pre-Flight & Review Checklist

Use this checklist before committing or proposing updates to `README.md` and `README.zh.md`.

---

## 1. Visual & Header Verification

- [ ] **Branding**: Logo image path exists and renders properly in both light and dark GitHub preview modes (`docs/logo.svg`).
- [ ] **Tagline**: The elevator pitch is accurate, compelling, and under 120 characters.
- [ ] **Badge Integrity**:
  - [ ] GitHub Actions CI badge points to the active workflow file.
  - [ ] Release badge dynamically resolves to `heartleo/zlib`.
  - [ ] License badge matches `LICENSE` (MIT).
  - [ ] Badges share a unified style (e.g., `flat-square`).
- [ ] **Language Switcher**:
  - [ ] `README.md` links to `[简体中文](README.zh.md)`.
  - [ ] `README.zh.md` links to `[English](README.md)`.
- [ ] **Nav Anchor Pills**: All anchor links (`#quick-start`, `#install`, etc.) resolve to valid headings.

---

## 2. Codebase & Delta Sync

- [ ] **New CLI Commands**: Every subcommand implemented in `internal/cli/` is represented in the command table.
- [ ] **Flags & Options**: All documented flags match the current Go source (`--json`, `--count`, `--ext`, `--dir`, etc.).
- [ ] **Interactive TUI**: Screen navigation, slash commands (`/search`, `/favorites`, `/theme`, etc.), and keybindings reflect `docs/tui.md`.
- [ ] **Themes**: Theme names match those supported in `internal/cli/theme.go` and `docs/themes.md`.
- [ ] **Environment Variables**: Table includes all active env vars (`ZLIB_DOWNLOAD_DIR`, `ZLIB_THEME`, `ZLIB_DOMAIN`, `HTTP_PROXY`).
- [ ] **Auto-Update**: `zlib update` is properly documented in the installation and command sections.

---

## 3. Bilingual Parity (EN ⟷ ZH)

- [ ] **1:1 Heading Alignment**: Every `##` and `###` heading in English has an exact equivalent in Chinese.
- [ ] **Code Block Consistency**: Every command, script snippet, and JSON example is identical across both language versions.
- [ ] **Table Equivalence**: All tables have the same number of rows, columns, and link targets.
- [ ] **No Raw Machine Translations**: Chinese phrasing follows standard developer conventions (e.g., 盘古之白空格规范, proper technical terminology).

---

## 4. Security & Privacy

- [ ] **No Secret Leaks**: No personal tokens, SESSDATA cookies, remix user keys, passwords, or internal domains are exposed in code blocks or curl snippets.
- [ ] **Legal / Disclaimer**: The educational / interoperability disclaimer is intact at the bottom of both README files.

---

## 5. Tooling & Plugin Synchronization

- [ ] **Plugin Skills Sync**: Run `node scripts/sync-plugin-skills.mjs` to mirror `.agents/skills/` into `plugins/zlib/skills/`.
- [ ] **CI Verification**: Ensure `node scripts/sync-plugin-skills.mjs --check` passes cleanly.
