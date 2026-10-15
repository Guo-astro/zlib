# Themes

Run `zlib theme` in a terminal to choose a theme. Type to filter the list;
Up/Down, PgUp/PgDn, and Home/End move through it. Enter saves the selection.
Esc clears the filter first, then closes the picker. The preview shows a sample
zlib search result and download status.

Run `zlib theme <name>` to set a theme directly. `ZLIB_THEME=<name>` overrides
the saved choice for one process. With redirected output, `zlib theme` prints
the current theme and the available names.

| Dark | Light |
| --- | --- |
| `catppuccin-mocha` | `github` |
| `gruvbox-dark` | `catppuccin-latte` |
| `dracula` | `gruvbox-light` |
| `tokyo` | `solarized-light` |
| `nord` | `one-half-light` |
| `solarized-dark` | `monokai-extended-light` |
| `monokai-extended` | `base16-ocean-light` |
| `one-half-dark` | `coldark-cold` |
| `two-dark` | |
| `catppuccin-macchiato` | |
| `base16-ocean-dark` | |
| `sublime-snazzy` | |

The default is `catppuccin-mocha`. Existing saved `mocha`, `latte`, and
`gruvbox` settings resolve to `catppuccin-mocha`, `catppuccin-latte`, and
`gruvbox-dark`. To update the stored name, select a theme or run
`zlib theme <name>`.
