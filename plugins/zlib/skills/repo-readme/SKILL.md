---
name: repo-readme
description: Edit or review zlib's English and Chinese READMEs, research CLI/TUI README examples, and verify onboarding commands and agent installation.
license: MIT
---

# Repository README

Keep installation, quick start, and agent setup in both READMEs.
Use `docs/` for detailed workflows and CLI help for exact flags.
Apply `no-ai-slop` when editing prose.

## Workflow

1. Read both READMEs and the docs for the workflows being changed. Inspect the
   relevant command help, implementation, and plugin manifests. Preserve existing
   worktree changes. Finish with evidence for each command, prerequisite, and
   behavior you plan to describe.
2. For structural changes, read [the research notes](references/research.md).
   If new research is requested, inspect current primary README sources and record
   the date, links, observations, and decisions. Borrow useful patterns; choose
   sections for zlib's users.
3. Put one installation route per main platform before the quick start. Link to
   releases and the installation guide for alternatives. Keep domain selection,
   prompted login, and opening the TUI together. Tell readers what to type next.
4. Draft both languages together. Keep commands, flags, destinations, and code
   blocks identical. Translate prose naturally. Use the language outlines below
   when restructuring; they describe content rather than duplicate the README.
5. Run [the checklist](references/checklist.md). Check local links and heading
   anchors, compare code blocks, and verify examples with command help. Use
   fixtures or help output for validation that would otherwise authenticate,
   download a book, or send mail.
6. If skill files changed, run `node scripts/sync-plugin-skills.mjs`, then
   `node scripts/sync-plugin-skills.mjs --check`. Finish with
   `git diff --check`. Report what was checked and any unverified claim.

## Evidence to check

- Login needs a configured domain or `--domain`. An example domain is not a
  guarantee of availability; direct readers to `zlib doctor`.
- Search with a query prints a table; search without a query opens a form.
  JSON examples need an explicit query and a complete `id:hash` for downloading.
- A TUI download starts with confirmation. Distinguish selecting from downloading.
- Plugins require the installed CLI and a session in the agent's environment.
  Verify marketplace names and syntax from manifests and installed CLI help.
- Read configuration paths and theme names from code. Put the maintained detail
  in the corresponding guide, rather than another README table.
- Use the CI workflow for the CI badge. Verify local demo assets exist; inspect
  them before claiming they show the current UI.

## References

- [Research](references/research.md): sources and decisions behind the layout.
- [Style](references/style-guide.md): presentation and wording.
- [English outline](references/template-en.md): English section responsibilities.
- [Chinese outline](references/template-zh.md): Chinese wording and section mapping.
- [Checklist](references/checklist.md): completion checks.
