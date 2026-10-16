# CLI/TUI README research

Reviewed 2026-10-15. These observations describe the linked READMEs at review time;
they are not universal rules or a popularity ranking. Recheck sources before
using them for a later redesign.

| Source | Observed structure | Decision for zlib |
| --- | --- | --- |
| [Crush](https://github.com/charmbracelet/crush/blob/main/README.md) | Logo and demo precede features and installation. Platform commands are visible; some longer install recipes use details blocks. | Keep the real demo, visible platform commands, and links for alternatives. |
| [Lazygit](https://github.com/jesseduffield/lazygit/blob/master/README.md) | Shows a terminal demo, describes features through concrete keyboard actions, and links to keybinding documentation. Installation covers many platforms. | Explain the first search and download confirmation through keys; link to the full TUI guide. Keep the platform list shorter. |
| [fzf](https://github.com/junegunn/fzf/blob/master/README.md) | Explains stdin/stdout behavior with a runnable pipeline. Build instructions have a separate document; advanced usage remains extensive. | Distinguish table output, JSON, and interactive commands. Move source builds out of the README. |
| [GitHub CLI](https://github.com/cli/cli/blob/trunk/README.md) | Gives a short product description and screenshot, links to the manual, and includes an agent-skills entry point. | Keep agent installation visible and link to JSON details in the agent guide. |

## zlib decisions

The primary path is install → check domains → prompted login → TUI search.
Script users get one JSON search and a download reference example. Agent users
get the CLI/session prerequisite and both plugin installation blocks.

Keep installation alternatives, environment-variable tables, file locations, and
API research in existing guides. A short command table points to common tasks;
`--help` remains the exact flag reference.

During review, the old README and templates repeated installation and configuration
details and carried stale claims: “over 20” themes, Windows configuration under
`%APPDATA%`, and “live search”. The revised README removes these claims and links
to the maintained guides. The CI badge now points to `ci.yml`.

## Verification sources

- Command behavior: `internal/cli/login.go`, `search.go`, `shell.go`, and command help.
- Configuration paths: `internal/config/config.go`; theme names: `internal/cli/theme.go`.
- Claude marketplace: `.claude-plugin/marketplace.json`.
- Codex marketplace: `.agents/plugins/marketplace.json`; syntax checked with installed
  `codex plugin add --help` and `codex plugin marketplace add --help`.
- [Homebrew tap](https://github.com/heartleo/homebrew-tap/blob/main/zlib.rb) and
  [winget manifests](https://github.com/microsoft/winget-pkgs/tree/master/manifests/h/heartleo/zlib)
  identify the package routes. Package installation was not run during this review.

A direct fetch during review found Homebrew at 0.0.8 and winget manifests through
0.0.8. These versions can change and need not match development-branch behavior.
The README links to source builds when a packaged version lacks a documented command.
