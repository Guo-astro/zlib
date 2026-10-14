# Config TUI implementation

See [Terminal interface](tui.md) for the shared layout, controls and verification.

Scope: domain, proxy URL, download directory and retry count. All `config`
subcommands route into one Bubble Tea program. Existing environment precedence
and configuration file locations remain in use.

## Component references

Crush reference revision: `b7b5522585e7302d2ee19b1a74a7003ac29ea562`.
The implementation uses our existing Bubble Tea v2, Bubbles and Lip Gloss
dependencies; it does not import or copy Crush's internal components.

| Component | Reference | Application here |
| --- | --- | --- |
| List and selection | `internal/ui/dialog/models.go` | Current selection, bounded rows, keyboard navigation and responsive width; matches our recommended-books list |
| Page and modal ownership | `internal/ui/dialog/dialog.go` | One root model routes input to the active page or modal |
| Inputs and verification | `internal/ui/dialog/api_key_input.go`, `inline_editor.go` | Masked proxy input, draft values, async verification and errors without losing edits |
| Directory navigation | `internal/ui/dialog/filepicker.go` | Directory-only navigation, parent traversal, selection separate from opening; selected path returns to the editor |
| Confirmation | `internal/logout/confirm.go` | Cancel is the default; switching focus and explicit confirmation discard edits |

## Boundaries

- `settings_store.go` reads effective values and sources, validates writes and
  replaces the global `.env` through a temporary file in the same directory.
- `settings_model.go` owns drafts, navigation, directory browsing and probes.
  Probe generations reject stale results; leaving a page cancels its request.
- `config.go` registers entry points and read-only non-interactive summaries.
- `doctor.go` shares `checkAvailableDomains` with the domain picker.

The process environment is captured before dotenv loading. This distinguishes
explicit process overrides from values loaded from configuration files.
Saving updates the process's derived setting only when no explicit override
exists. Domain candidates use the allowlist and are deduplicated by final
origin. Proxy testing does not save the draft or send account credentials.

Directory access is local; no directories are created by the picker. Config
parent directories are created on save. Domain availability describes one
unauthenticated probe, not account access or download permission.
