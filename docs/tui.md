# Terminal interface

Run `zlib tui` in a terminal for a persistent shell. Type a book title to search, or
type `/` for commands. Tab completes the selected command; Enter runs it, even
when you've typed only part of its name (for example, `/th` selects `/theme`).
`/quit` exits. The shell keeps the current book list and selection when you return from
a download, theme, settings or Kindle screen.

The signed-in start screen shows a five-row ZLIB wordmark with theme colors
for each letter. Its build version sits below the right edge of the logo; long
development versions are shortened there and remain available through
`zlib version`. The logo has no decorative side strokes.
With at least 16 content rows, one blank row separates the wordmark and version,
and two separate the version and command hints. Compact layouts reduce these
gaps. Command columns have a four-cell gutter; single-column hints fit their content.
It does not display the current directory or a persistent sign-in label. At 76
columns or wider, command hints use two columns; smaller terminals use one.
The hints show commands to type in the input; typing `/` opens the selectable
command menu. Without a saved session, `zlib tui` opens the email and password
form immediately. Press Esc to reach the signed-out start screen, where Enter
reopens login and `/` gives access to settings and domain checks. A successful
login returns to the start screen. If you tried a protected command first, its
draft stays in the input so
you can press Enter to continue. Login errors stay on the form for another try.
The TUI login form accepts email and password. To import a Cookie file, use
`zlib login --cookies <file> --domain <domain>` outside the TUI.
`/logout` asks for confirmation, clears the saved session and account pages, then
opens the login form. If session removal fails, the current screen shows an error.

The command menu appears when you type `/`; it shows up to six matches with short
descriptions and no count heading. ↑/↓ cycles through the matches, including from
last to first.
The menu scrolls only when the selection leaves its visible rows. Type more
characters to narrow the list.
Ctrl+K opens the menu without discarding an unfinished search; Esc restores the
search text.

Direct commands such as `zlib search` and `zlib history --json` still work.
The search form uses a borderless Search button with an accent background when
focused. Tab focuses it, Enter submits, and Ctrl+S submits from any field.
Bare `zlib` prints command help. `zlib tui` requires an interactive terminal.

The shell accepts `/search`, `/favorites`, `/recommended`, `/most-popular`,
`/history`, `/login`, `/logout`, `/profile`, `/doctor`,
`/config`, `/config network`, `/config download`, `/config kindle`, `/theme`, `/kindle`,
`/kindle config`, `/kindle send <file>` and `/quit`. Use `zlib version`
outside the TUI to check the installed version.
For automation, use direct commands and their flags, including `--json`.
To download a selected book in the TUI, press Enter on its list row. To download
by `id:hash` directly, use `zlib download "id:hash"` outside the TUI.
The search list keeps its scroll position while you move within visible rows.
If a search returns nothing or fails, edit the query with Ctrl+F, or retry a
failed request with r.
Search and the other book lists show a Charm dot spinner while fetching results.
Page numbers appear after loading. Profile, login, and settings domain checks
also use the dot spinner while waiting for a response.
Doctor shows only the Charm dot spinner while checking domains; Esc returns.
`zlib doctor` opens the same page in an interactive terminal. Results contain
only available domains, without a status label on each row. The accent border
marks the focused row; `Current domain` marks the configured domain, which gets
initial focus when available. Moving the focus does not change configuration.
Redirected output uses a single-column table; `--json` keeps the probe fields.
The position bar follows the selected book.
Search loads up to 50 books per server page, matching the direct `search` command.
The number of visible rows follows the terminal height; scroll within a page with
↑/↓ or PgUp/PgDn and change server pages with ←/→.

## Book list spacing

Search, Favorites, Recommended, Most popular, and History share one list layout:

- Put the title and status above the selected-book card. At 78 columns or wider
  with at least 20 page rows, the card uses 11 rows. From 30 to 77 columns it
  uses 13 rows when the page has at least 24 rows. Otherwise it uses 8 rows.
- The expanded card shows category, year, edition, publisher, language, pages,
  and ISBN 10/13 when available. Missing EAPI fields are left blank. Valid
  ISBN 10 and `978` ISBN 13 values can supply each other's equivalent.
- Leave one empty row between the card and the column header.
- At a page height of 15 rows or more, leave one empty row between the list area
  and the bottom controls.
- At a page height of 14 rows, omit the bottom gap to keep a book visible. Below
  14 rows or narrower than 30 columns, show the enlarge-terminal prompt.

Calculate visible book rows from the page height. The shell reserves space for
its footer and, while typing a command, its input. Keep `feedRows()` and
`feedViewWith()` in sync so scrolling, selection, and rendering agree. Use
`fullScreenView()` to fill any remaining terminal space. The cross-page layout
test is `TestBookListSpacingMatchesSearchAcrossPages`.

The selected book's EAPI details load after the selection settles. Results are
cached by `id:hash`; moving to another book keeps an older response from
replacing its card. The list response remains visible while details load.
The card border uses the theme accent, and its author uses the theme's primary
text color. Moving the selection updates the card and position bar immediately,
without a color flash or animated position change.
History uses the same `#`, `Title`, and `Author` columns as the other book lists.
The selected entry's download date appears beside the page status. Its card uses
the book's author and metadata when EAPI provides them; moving within visible
history rows keeps the list viewport still.

## Controls

| Screen | Controls |
| --- | --- |
| Shell command input | `/` opens commands from a list; ↑/↓ cycles; Ctrl+K also opens it; Tab completes; Esc closes; pasted text needs Enter to run |
| Profile | Full-screen account and quota page; ↑/↓ or PgUp/PgDn scrolls; r retries a failed request; Esc returns |
| Doctor | Full-screen available-domain list; the focused row has an accent border and the current configured domain has a separate marker; ↑/↓ selects, PgUp/PgDn scrolls, r checks again, Esc returns |
| Book lists | ↑/↓ or j/k select; PgUp/PgDn scroll; Home/End jump; Enter opens download confirmation; q or Esc exits |
| Search, saved books, history | ←/→ changes page; Ctrl+F edits the query in shell search; `/` edits it in direct `zlib search` |
| Failed list request | r retries the failed page; Esc closes |
| Login, search options, Kindle settings | Enter edits a field; Enter applies an edit; Tab advances; choose the final action or press Ctrl+S to submit; Esc cancels |
| Format selection | Huh checkbox list: ↑/↓ moves; Space toggles; Ctrl+A selects all or clears all; Enter continues; no selected formats means all formats |
| Config | Network, download and Kindle settings appear in one compact list; arrows or Tab select a field; Enter opens it; Esc returns; Tab selects editor actions |
| File picker | Enter opens a folder or selects a file; ← goes to the parent; `/` accepts a directory path; Esc exits |
| Download directory | Skip uses the displayed directory; Change directory applies only to this download; Ctrl+B browses folders |
| Theme | Type to filter; ↑/↓ previews; Enter saves; Esc clears the filter, then closes |
| Confirmation | Cancel is selected initially; ←/→ or Tab changes the action; Enter confirms |
| Download progress | Esc or Ctrl+C cancels the transfer; a failure stays on its own page with r to retry using the same book and directory, or Esc to return |

Passwords are masked. Invalid field values stay in the editor with an error.
Kindle SMTP sending cannot be recalled: Ctrl+C stops waiting, and the message
may still be delivered.
After a Kindle send error, the file and error remain visible. Check the Kindle
inbox before starting another send; the error page does not resend automatically.
Download errors hide signed URLs in the terminal view.

## Implementation references

Crush revision: `f49b28113b3bab3fcfe193676cc3f7fb858d6411`.
The local source checkout is `D:\myspace\github\AI\crush` (Git Bash:
`/d/myspace/github/AI/crush`; WSL: `/mnt/d/myspace/github/AI/crush`). Check its
current HEAD before comparing code with the revision above.
The implementation uses Bubbles v2, Huh v2 and Lip Gloss v2, with interaction patterns
from Crush's `internal/ui/dialog/models.go`, `filepicker.go`, `inline_editor.go`,
`dialog.go`, `internal/ui/list/list.go` and `internal/ui/model/status.go`.
Crush internal packages are not imported or copied.
See [Crush TUI patterns](crush-ui-patterns.md) for the component and layout analysis.

- `tui_layout.go`: terminal-sized screen layout, selected rows and Bubbles help.
- `tui_form.go`, `tui_inline_form.go`: field lists, masked text input, validation
  and Huh format checkboxes in the inline search form.
- `tui_filepicker.go`: Bubbles file picker, path input and send confirmation.
- `book_feed_view.go`: shared book list for search, recommendations, popular,
  favorites and history.
- `confirmation.go`: modal overlay with Cancel selected by default.

## Verification

Automated tests cover Unicode and paste, masked passwords, invalid inputs,
clearing format filters, cancellation, list retry, modal actions and dimensions
from 20×5 to 120×30. Small windows preserve controls or request more space.

The terminal walkthrough uses local fixtures and an HTTP server. It covers
recommendations, popular books, saved pagination, search refinement, history
confirmation, domain selection, download directory, Kindle file confirmation and
a 1 MiB download. It never contacts Z-Library or sends SMTP mail.

```powershell
go test -c ./internal/cli -o "$env:TEMP/zlib-tui-smoke.exe"
$env:ZLIB_TUI_SMOKE = '1'
& "$env:TEMP/zlib-tui-smoke.exe" '-test.run=^TestTUIWalkthrough$' '-test.v'
```

The walkthrough was run in a Windows terminal. Login masking and cancellation,
search query/format editing, theme preview/cancellation and isolated Kindle and
retry-setting saves were also exercised through `zlib.exe`.
