# zlib plugin

Claude Code and Codex load the same generated `skills/` bundle from this directory.
Both manifests and all skill references stay inside the plugin package.

Edit skills in [`.agents/skills/`](../../.agents/skills), then regenerate the bundle:

```sh
node scripts/sync-plugin-skills.mjs
node scripts/sync-plugin-skills.mjs --check
```

Run these commands from the repository root. Include the source and generated files
in the same change. CI checks that they match. See [agent setup](../../docs/agents.md)
for installation and usage.
