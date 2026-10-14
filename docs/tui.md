# Terminal interface

Interactive commands use a full-screen list with a title, selection, details and
keyboard hints. Output redirected to a file or pipe keeps the command's text or
JSON format.

## Controls

| Screen | Controls |
| --- | --- |
| Book lists | ↑/↓ or j/k select; PgUp/PgDn scroll; Home/End jump; Enter opens download confirmation; q or Esc exits |
| Search, saved books, history | ←/→ changes page; `/` edits the query in search |
| Failed list request | r retries the failed page; Esc closes |
| Login, search options, Kindle settings | Enter edits a field; Enter applies an edit; Tab advances; choose the final action or press Ctrl+S to submit; Esc cancels |
| Format selection | Space toggles; Enter returns; no selected formats means all formats |
| Config | Enter opens or saves; Tab selects editor actions; Esc returns; q exits lists |
| File picker | Enter opens a folder or selects a file; ← goes to the parent; `/` accepts a directory path; Esc exits |
| Download directory | Skip uses the displayed directory; Change directory applies only to this download; Ctrl+B browses folders |
| Theme | ↑/↓ previews; Enter saves; Esc cancels |
| Confirmation | Cancel is selected initially; ←/→ or Tab changes the action; Enter confirms |
| Download progress | Esc or Ctrl+C cancels the transfer |

Passwords are masked. Invalid field values stay in the editor with an error.
Kindle SMTP sending cannot be recalled: Ctrl+C stops waiting, and the message
may still be delivered.

## Implementation references

Crush revision: `b7b5522585e7302d2ee19b1a74a7003ac29ea562`.
The implementation uses Bubbles v2 and Lip Gloss v2, with interaction patterns
from Crush's `internal/ui/dialog/models.go`, `filepicker.go`, `inline_editor.go`,
`dialog.go`, `internal/ui/list/list.go` and `internal/ui/model/status.go`.
Crush internal packages are not imported or copied.

- `tui_layout.go`: bounded screen layout, selected rows and Bubbles help.
- `tui_form.go`: field lists, masked text input, validation and format selection.
- `tui_filepicker.go`: Bubbles file picker, path input and send confirmation.
- `book_feed_view.go`: shared book list for search, recommendations, popular,
  saved/favorites and history.
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
