# Themes

Run `zlib theme` in a terminal to choose a theme. Type to filter the list;
Up/Down, PgUp/PgDn, and Home/End move through it. Enter saves the selection.
Esc clears the filter first, then closes the picker. The preview shows a sample
zlib search result and download status.

Press Ctrl+N to create a theme from the highlighted palette. Enter a lowercase
name, then edit colors as `#RRGGBB`. Up/Down or Tab moves between color fields;
Ctrl+S saves and Esc cancels. Ctrl+E edits a saved custom theme. On a built-in
theme, Ctrl+E starts a copy so the built-in palette stays available. Saved themes
appear after the built-in themes in the picker. Enter selects one for zlib.
Theme files are stored in `~/.config/zlib/themes/<name>.json` (or the equivalent
user home on Windows). Each file records its built-in base and color values.

The list and filter keep the current theme while the preview uses the highlighted
theme. The selected-book row uses that theme's surface color. Both areas paint
their backgrounds, including blank cells, so light and dark previews stay
readable regardless of the terminal background. Moving through the list does not
save a preference; Enter selects and Esc cancels.

Every other screen sets the terminal's default background and text color to the
theme's own while zlib runs, so a light theme stays readable on a dark terminal
and the reverse. zlib restores the terminal's colors on exit. Terminals that do
not support changing their default colors keep their own background.

Built-in themes adapt their source palettes for reading. Every text color,
including secondary text, reaches at least 4.5:1 contrast (WCAG AA) on the
theme background. Colors that miss it are moved toward black or white. The four
format colors (EPUB, PDF, MOBI, FB2) stay visibly different, and warnings use
amber or orange rather than a palette color. Custom themes keep the exact
colors you enter.

Run `zlib theme <name>` to set a built-in or saved custom theme directly.
`ZLIB_THEME=<name>` overrides
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
