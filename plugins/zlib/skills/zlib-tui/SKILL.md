---
name: zlib-tui
description: Change zlib's Bubble Tea terminal screens, book cards, list spacing, or shell navigation while preserving shared layout and selection behavior.
---

# zlib TUI changes

For the theme picker, keep the list and filter in the current palette while the preview follows the highlighted name. Paint component backgrounds, including cursor and separator cells. Check `TestThemePickerColorsFollowSelection` for all palettes at compact and wide sizes; preview and cancellation must not write preferences.

Crush source is available at `D:\myspace\github\AI\crush` (`/d/myspace/github/AI/crush` in Git Bash, `/mnt/d/myspace/github/AI/crush` in WSL). Check its current HEAD before relying on an old revision cited in docs. Browse `internal/ui/` for layout and component patterns. Its `internal` packages cannot be imported by zlib; use the Charm packages already in `go.mod`.

Read [the TUI layout rules](../../../docs/tui.md#book-list-spacing) before changing book lists or cards. `book_feed_view.go` renders the shared Search, Favorites, Recommended, Most popular, and History list; keep its row calculation and rendered height aligned at compact and expanded terminal sizes.

Book details come from EAPI for the selected `id:hash`. Fetch after selection settles, cache by reference, and keep late results tied to their original book. Show fields the response provides; the Android model does not guarantee categories or separate ISBN fields. See [EAPI notes](../../../docs/eapi.md#当前范围) for the evidence boundary.

Check the shared layout and selection tests in `internal/cli/book_feed_view_test.go` and the EAPI mapping tests in `eapi_book_test.go`. When changing shared layout code, run `go test ./...` and `go vet ./...` after the focused test passes. Update `docs/tui.md` when interaction or spacing rules change.

Keep recoverable failures on the page that owns the request. Download failure shows the book, a URL-safe reason, and `r` to retry with the same directory; retry must start a new shell generation so late messages are ignored. Kindle send failure shows the file and reason but has no automatic retry because SMTP delivery may have completed. Keep login and setting validation on their forms. Check `TestShellDownloadFailureKeepsContextAndRetries` and `TestShellKindleFailureRequiresExplicitResend` when changing these paths.

The Doctor shell page uses Bubbles list, spinner, and the shared help renderer, following the local Crush command-list pattern. Keep its results limited to domains accepted by `availableDomains`. Show one domain per row, use the accent border for the focused row, and mark the current configured domain separately. Retry and page exit must cancel the prior probe context. Check `TestDoctorPageListsOnlyAvailableDomains` and `TestDoctorPageRetryRejectsOldResultsAndCancelsOnExit` after changing it.

For shell home changes, keep the original five-row wordmark, per-letter theme colors, version below its right edge, and signed-in command hints in `shell_home.go`. Keep the logo free of side strokes. The hints name commands registered in `shellCommands`; typing `/` opens the selectable palette. Without a saved session, shell startup opens the login form directly; Esc reaches the signed-out home, which shows the login action and method. Keep Enter available there on compact terminals. After logout, open the login form again. Do not put session status or the working directory below the logo. Check `TestShellHomeLogoAndShortcutsResponsive`, `TestShellSignedOutHomeOpensLogin`, and `TestShellStartsOnLoginWithoutSession`.
