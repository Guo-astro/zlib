---
name: zlib-tui
description: Change zlib's Bubble Tea terminal screens, book cards, list spacing, or shell navigation while preserving shared layout and selection behavior.
---

# zlib TUI changes

Read [the TUI layout rules](../../../docs/tui.md#book-list-spacing) before changing book lists or cards. `book_feed_view.go` renders the shared Search, Favorites, Recommended, Most popular, and History list; keep its row calculation and rendered height aligned at compact and expanded terminal sizes.

Book details come from EAPI for the selected `id:hash`. Fetch after selection settles, cache by reference, and keep late results tied to their original book. Show fields the response provides; the Android model does not guarantee categories or separate ISBN fields. See [EAPI notes](../../../docs/eapi.md#当前范围) for the evidence boundary.

Check the shared layout and selection tests in `internal/cli/book_feed_view_test.go` and the EAPI mapping tests in `eapi_book_test.go`. When changing shared layout code, run `go test ./...` and `go vet ./...` after the focused test passes. Update `docs/tui.md` when interaction or spacing rules change.

For shell home changes, keep the original five-row wordmark, per-letter theme colors, version below its right edge, and signed-in command hints in `shell_home.go`. Keep the logo free of side strokes. The hints name commands registered in `shellCommands`; typing `/` opens the selectable palette. The signed-out home shows the login action and method; keep Enter available on compact terminals. Do not put session status or the working directory below the logo. Check `TestShellHomeLogoAndShortcutsResponsive` and `TestShellSignedOutHomeOpensLogin`.
