# README checks

- Both languages show a purpose, installation, first-use commands, and agent setup.
- The first-use sequence includes account and domain prerequisites.
- Each command and flag exists in current help or code; example placeholders are explained.
- Query search, interactive search, and JSON output are described accurately.
- Plugin selectors match the marketplace manifests. The CLI/session prerequisite is visible.
- Commands in corresponding fenced blocks are identical between languages.
- Local images and links exist. Heading anchors and inbound links resolve.
- Badges point to the intended workflow, release repository, and license.
- Detailed configuration and installation alternatives remain reachable in `docs/`.
- Research notes distinguish observations from zlib-specific editorial decisions.
- Skill references agree with the README structure and avoid maintaining duplicate full drafts.
- `node scripts/sync-plugin-skills.mjs --check` passes after skill synchronization.
- `git diff --check` passes.

Use the smallest checks that prove the change. Documentation edits do not require
installing packages, making authenticated requests, or sending messages.
