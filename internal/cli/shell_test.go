package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/heartleo/zlib"
	"github.com/heartleo/zlib/internal/config"
)

func shellStep(m shellModel, msg tea.Msg) (shellModel, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(shellModel), cmd
}

func TestShellDownloadReturnsToSelectedBook(t *testing.T) {
	m := newShellModel()
	list := newBookFeedSelectModel("Recommended", nil)
	list.state = searchStateResults
	list.books = []zlib.Book{{ID: "1", Hash: "abc", Name: "First"}, {ID: "2", Hash: "def", Name: "Second"}}
	list.cursor = 1
	m.open("list", list)

	m, cmd := shellStep(m, key("enter"))
	if cmd != nil || !m.page.(searchSelectModel).confirmDownload {
		t.Fatal("list Enter did not open download confirmation")
	}
	m, _ = shellStep(m, key("enter"))
	if m.kind != "list" || m.page.(searchSelectModel).cursor != 1 {
		t.Fatal("default cancel did not restore selection")
	}

	m, _ = shellStep(m, key("enter"))
	m, _ = shellStep(m, key("right"))
	m, cmd = shellStep(m, key("enter"))
	event := cmd()
	if _, ok := event.(shellEvent); !ok {
		t.Fatal("list confirmation was not scoped to the shell")
	}
	m, _ = shellStep(m, event)
	if m.kind != "directory" {
		t.Fatalf("confirmed download opened %q", m.kind)
	}
	m, cmd = shellStep(m, key("enter"))
	m, _ = shellStep(m, cmd())
	if m.kind != "download" {
		t.Fatalf("directory selection opened %q", m.kind)
	}
	download := m.page.(downloadModel)
	download.state = stateComplete
	download.savedPath = "book.epub"
	m.page = download
	m, _ = shellStep(m, shellEvent{generation: m.generation, msg: tea.QuitMsg{}})
	got := m.page.(searchSelectModel)
	if m.kind != "list" || got.cursor != 1 || m.status != "Saved to: book.epub" {
		t.Fatalf("download did not restore selection: %q, %d, %q", m.kind, got.cursor, m.status)
	}
}

func TestShellUnknownCommandAndQuotedKindlePath(t *testing.T) {
	m := newShellModel()
	m.input.SetValue("/missing")
	m, _ = shellStep(m, key("enter"))
	if m.input.Value() != "/missing" || !strings.Contains(m.status, "Unknown command") {
		t.Fatal("unknown command should keep the draft")
	}
	path := filepath.Join(t.TempDir(), "book with spaces.epub")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	m.dispatch("/kindle send \"" + path + "\"")
	if m.kind != "kindle-confirm" || m.kindlePath != path {
		t.Fatalf("quoted path was not parsed: %q, %q", m.kind, m.kindlePath)
	}
}

func TestShellCommandPaletteAndReturn(t *testing.T) {
	m := newShellModel()
	m, _ = shellStep(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m.input.SetValue("/fa")
	if got := m.matches(); len(got) != 1 || got[0].name != "favorites" {
		t.Fatalf("matches = %v", got)
	}
	m, _ = shellStep(m, key("enter"))
	// No persisted session is needed to verify matching: the command was resolved
	// before authentication and its value is retained for login recovery.
	if m.pending != "/favorites" && m.kind != "list" {
		t.Fatalf("slash prefix did not dispatch favorites: %q, %q", m.pending, m.kind)
	}
	m.home()
	m.signedIn = true
	m, _ = shellStep(m, tea.PasteMsg{Content: "/logout"})
	if m.kind != "" {
		t.Fatal("paste executed a command")
	}
	m, _ = shellStep(m, key("enter"))
	if m.kind != "logout" {
		t.Fatal("logout confirmation did not open")
	}
	confirm := m.page.(shellConfirm)
	if confirm.focused || confirm.confirmed {
		t.Fatal("logout must default to cancel")
	}
	m, _ = shellStep(m, shellEvent{generation: m.generation, msg: tea.QuitMsg{}})
	if m.page != nil || m.kind != "" {
		t.Fatal("cancel did not return home")
	}
}

func isolateShellSession(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	path := config.SessionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestShellLogoutReturnsToSignedOutHome(t *testing.T) {
	path := isolateShellSession(t)
	if err := os.WriteFile(path, []byte(`{"cookies":{"session":"test"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newShellModel()
	if !m.signedIn {
		t.Fatal("test session was not loaded")
	}
	list := newBookFeedSelectModel("Recommended", nil)
	list.state = searchStateResults
	list.books = []zlib.Book{{ID: "1", Name: "Private book"}}
	m.open("list", list)
	m.pending = "/favorites"
	m.dispatch("/logout")
	confirm := m.page.(shellConfirm)
	confirm.confirmed = true
	m.page = confirm
	m, _ = shellStep(m, shellEvent{generation: m.generation, msg: tea.QuitMsg{}})
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("session remains after logout: %v", err)
	}
	if m.signedIn || m.kind != "" || m.page != nil || m.previous != nil || m.pending != "" || m.input.Value() != "" || !m.typing {
		t.Fatalf("logout kept account state: signedIn=%v kind=%q pending=%q", m.signedIn, m.kind, m.pending)
	}
	if m.status != "" {
		t.Fatalf("logout left a redundant footer status: %q", m.status)
	}
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "Sign in to Z-Library") || !strings.Contains(view, "› Log in") || strings.Contains(view, "Private book") {
		t.Fatal("logout did not show the signed-out home")
	}
	m, _ = shellStep(m, key("enter"))
	if m.kind != "login" {
		t.Fatalf("sign-in action opened %q", m.kind)
	}
}

func TestShellLogoutCancelKeepsSession(t *testing.T) {
	path := isolateShellSession(t)
	if err := os.WriteFile(path, []byte(`{"cookies":{"session":"test"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newShellModel()
	list := newBookFeedSelectModel("Recommended", nil)
	list.state = searchStateResults
	list.books = []zlib.Book{{ID: "1", Name: "First"}, {ID: "2", Name: "Second"}}
	list.cursor = 1
	m.open("list", list)
	m.dispatch("/logout")
	m, _ = shellStep(m, shellEvent{generation: m.generation, msg: tea.QuitMsg{}})
	if !m.signedIn || m.kind != "list" || m.page.(searchSelectModel).cursor != 1 {
		t.Fatal("cancel did not restore the signed-in list")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cancel removed the saved session: %v", err)
	}
}

func TestShellLogoutFailureKeepsAccountState(t *testing.T) {
	path := isolateShellSession(t)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "blocker"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	m := newShellModel()
	m.signedIn = true
	list := newBookFeedSelectModel("Recommended", nil)
	list.state = searchStateResults
	list.books = []zlib.Book{{ID: "1", Name: "First"}, {ID: "2", Name: "Second"}}
	list.cursor = 1
	m.open("list", list)
	m.dispatch("/logout")
	confirm := m.page.(shellConfirm)
	confirm.confirmed = true
	m.page = confirm
	m, _ = shellStep(m, shellEvent{generation: m.generation, msg: tea.QuitMsg{}})
	if !m.signedIn || m.kind != "list" || m.page.(searchSelectModel).cursor != 1 || !strings.Contains(m.status, "Could not log out") {
		t.Fatalf("logout failure lost active account or selection: signedIn=%v kind=%q status=%q", m.signedIn, m.kind, m.status)
	}
}

func TestShellLogoutWhenAlreadySignedOut(t *testing.T) {
	path := isolateShellSession(t)
	m := newShellModel()
	m.dispatch("/logout")
	if m.kind != "" || m.page != nil || m.status != "Not logged in" {
		t.Fatalf("signed-out logout opened a dialog: kind=%q status=%q", m.kind, m.status)
	}
	if err := os.WriteFile(path, []byte(`{"cookies":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m.dispatch("/logout")
	if m.status != "Logged out" {
		t.Fatalf("stale session was not cleared: %q", m.status)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stale session remains: %v", err)
	}
}

func TestShellPartialCommandEnterOpensSelectedScreen(t *testing.T) {
	m := newShellModel()
	m, _ = shellStep(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m.input.SetValue("/th")
	m, _ = shellStep(m, key("enter"))
	if m.kind != "theme" {
		t.Fatalf("partial command opened %q, want theme", m.kind)
	}
}

func TestShellSignedOutHomeOpensLogin(t *testing.T) {
	m := newShellModel()
	m.signedIn = false
	m.input.SetValue("")
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "Sign in to Z-Library") || !strings.Contains(view, "› Log in") || !strings.Contains(view, "Email and password") || !strings.Contains(view, "enter sign in") {
		t.Fatal("signed-out home has no clear sign-in action")
	}
	if strings.Contains(view, "/search") || strings.Contains(view, "/favorites") {
		t.Fatal("signed-out home shows account commands before login")
	}
	m, _ = shellStep(m, tea.WindowSizeMsg{Width: 20, Height: 5})
	if !strings.Contains(ansi.Strip(m.View().Content), "Log in") {
		t.Fatal("small signed-out home hides its sign-in action")
	}
	m, _ = shellStep(m, key("enter"))
	if m.kind != "login" {
		t.Fatalf("Enter on signed-out home opened %q", m.kind)
	}
}

func TestShellLoginSuccessReturnsHomeWithPendingDraft(t *testing.T) {
	m := newShellModel()
	m.signedIn = false
	m.kind = "login-wait"
	m.typing = false
	m.pending = "/favorites"
	form := newInlineForm("Log in", "Log in", []formField{{label: "Email"}, {label: "Password", secret: true}})
	m.loginForm = new(form)
	m, cmd := shellStep(m, shellEvent{generation: m.generation, msg: shellText{text: "Logged in"}})
	if cmd != nil || m.kind != "" || m.page != nil || !m.typing || !m.signedIn || m.previous != nil {
		t.Fatalf("login did not return home: kind=%q signedIn=%v", m.kind, m.signedIn)
	}
	if m.input.Value() != "/favorites" || m.pending != "" || !strings.Contains(m.status, "Press Enter") {
		t.Fatalf("pending command was lost: input=%q pending=%q status=%q", m.input.Value(), m.pending, m.status)
	}
}

func TestShellLoginSuccessHomeHasNoSessionLabel(t *testing.T) {
	m := newShellModel()
	m.signedIn = false
	m.kind = "login-wait"
	m.typing = false
	m, _ = shellStep(m, shellEvent{generation: m.generation, msg: shellText{text: "Logged in"}})
	if !m.signedIn || m.status != "" || strings.Contains(ansi.Strip(m.View().Content), "Signed in") {
		t.Fatal("login left a persistent session label on the home screen")
	}
}

func TestShellLoginFailureReturnsToForm(t *testing.T) {
	m := newShellModel()
	form := newInlineForm("Log in", "Log in", []formField{{label: "Email", value: "reader@example.com"}, {label: "Password", value: "secret", secret: true}})
	m.loginForm = new(form)
	m.kind = "login-wait"
	m.typing = false
	m, _ = shellStep(m, shellEvent{generation: m.generation, msg: shellText{err: errors.New("invalid credentials")}})
	if m.kind != "login" || m.loginForm != nil {
		t.Fatalf("login failure opened %q", m.kind)
	}
	got := m.page.(inlineForm)
	if got.message != "invalid credentials" || got.fields[0].value != "reader@example.com" || got.fields[1].value != "" || got.cursor != 1 {
		t.Fatalf("retry form lost its state: message=%q cursor=%d", got.message, got.cursor)
	}
}

func TestShellLoginFormHasOnlyEmailAndPassword(t *testing.T) {
	m := newShellModel()
	m.dispatch("/login")
	if m.kind != "login" {
		t.Fatalf("login opened %q", m.kind)
	}
	form := m.page.(inlineForm)
	if len(form.fields) != 2 || form.fields[0].label != "Email" || form.fields[1].label != "Password" || !form.fields[1].secret {
		t.Fatal("login form must contain only email and masked password")
	}
	if strings.Contains(ansi.Strip(m.View().Content), "Cookie file") {
		t.Fatal("TUI login still offers Cookie file import")
	}
}

func TestShellDownloadIsNotASlashCommand(t *testing.T) {
	m := newShellModel()
	m.input.SetValue("/download 1:abc")
	m, _ = shellStep(m, key("enter"))
	if m.kind != "" || !strings.Contains(m.status, "Unknown command: /download") {
		t.Fatalf("download command was accepted: kind=%q status=%q", m.kind, m.status)
	}
	m.input.SetValue("/")
	m.status = ""
	if strings.Contains(ansi.Strip(m.View().Content), "/download") {
		t.Fatal("download is still shown in the command palette")
	}
}

func TestShellIgnoresOldResultsAndReturnsToSelection(t *testing.T) {
	m := newShellModel()
	list := newBookFeedSelectModel("Recommended", nil)
	list.state = searchStateResults
	list.books = []zlib.Book{{ID: "1", Hash: "abc", Name: "First"}, {ID: "2", Hash: "def", Name: "Second"}}
	list.cursor = 1
	m.open("list", list)
	firstGeneration := m.generation
	m, _ = shellStep(m, shellEvent{generation: firstGeneration - 1, msg: searchResultMsg{books: []zlib.Book{{Name: "Stale"}}}})
	if m.page.(searchSelectModel).books[0].Name != "First" {
		t.Fatal("stale page result applied")
	}
	m.previous = m.page
	m.open("directory", newDownloadDirectoryModel("."))
	m.restore()
	got := m.page.(searchSelectModel)
	if got.cursor != 1 || got.books[1].Name != "Second" {
		t.Fatal("return lost list position")
	}
	if strings.Contains(ansi.Strip(m.View().Content), "Stale") {
		t.Fatal("stale data rendered")
	}
}

func TestShellViewFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{120, 30}, {80, 24}, {40, 16}, {20, 5}} {
		m := newShellModel()
		m, _ = shellStep(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m.input.SetValue("/")
		view := m.View()
		if !view.AltScreen || len(strings.Split(view.Content, "\n")) > size[1] {
			t.Fatalf("shell exceeds height at %v", size)
		}
		for _, line := range strings.Split(view.Content, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("shell exceeds width at %v", size)
			}
		}
	}
}

func TestShellCommandsStayInPalette(t *testing.T) {
	m := newShellModel()
	m, _ = shellStep(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m.signedIn = true
	home := ansi.Strip(m.View().Content)
	for _, command := range []string{"/search", "/favorites", "/recommended", "/history", "/config", "/kindle"} {
		if !strings.Contains(home, command) {
			t.Fatalf("home omits %s", command)
		}
	}
	if strings.Index(home, "█████") > strings.Index(home, "/search") {
		t.Fatal("commands are above the wordmark")
	}
	m.input.SetValue("/")
	palette := ansi.Strip(m.View().Content)
	if !strings.Contains(palette, "/search") || !strings.Contains(palette, "Search books by title") || strings.Contains(palette, "Commands  ") {
		t.Fatal("palette should show commands and descriptions without a count heading")
	}
	if strings.Count(palette, "/search") != 1 {
		t.Fatal("home shortcuts duplicate the active command palette")
	}
	if rows := m.paletteLines(m.contentHeight()); len(rows) != 6 {
		t.Fatalf("palette has %d rows, want 6 command rows", len(rows))
	}
	if strings.Contains(palette, "/quit") {
		t.Fatal("palette should show a bounded set of commands")
	}
	m.selection = 8
	if later := ansi.Strip(m.View().Content); !strings.Contains(later, "/theme") || strings.Contains(later, "/search") {
		t.Fatal("palette did not scroll to the selected command")
	}
	m.input.SetValue("/pop")
	if got := m.matches(); len(got) != 1 || got[0].name != "most-popular" {
		t.Fatalf("substring command match = %v", got)
	}
	m.input.SetValue("/help")
	m, _ = shellStep(m, key("enter"))
	if m.kind != "" || !strings.Contains(m.status, "Unknown command: /help") {
		t.Fatal("help should not be a TUI command")
	}
}

func TestShellHomeLogoAndShortcutsResponsive(t *testing.T) {
	for _, tc := range []struct {
		width, height int
		large         bool
		columns       int
	}{
		{80, 24, true, 2},
		{40, 16, true, 1},
		{24, 12, false, 1},
	} {
		m := newShellModel()
		m.signedIn = true
		m.width, m.height = tc.width, tc.height
		view := ansi.Strip(m.View().Content)
		if strings.Contains(view, "Signed in") {
			t.Fatalf("%dx%d: home shows a redundant session row", tc.width, tc.height)
		}
		if !strings.Contains(view, "/search") || !strings.Contains(view, "/favorites") {
			t.Fatalf("%dx%d: primary commands are missing", tc.width, tc.height)
		}
		if tc.large && !strings.Contains(view, "█████") {
			t.Fatalf("%dx%d: large wordmark is missing", tc.width, tc.height)
		}
		if !tc.large && !strings.Contains(view, "ZLIB") {
			t.Fatalf("%dx%d: compact wordmark is missing", tc.width, tc.height)
		}
		if !strings.Contains(view, zlib.Version) || strings.Contains(view, "╱") {
			t.Fatalf("%dx%d: version is missing or thin strokes remain", tc.width, tc.height)
		}
		for _, row := range homeLogo(tc.width, tc.height) {
			plain := strings.TrimSpace(ansi.Strip(row))
			if strings.HasPrefix(plain, "╱") || strings.HasSuffix(plain, "╱") {
				t.Fatalf("%dx%d: decorative slashes surround the logo: %q", tc.width, tc.height, plain)
			}
		}
		if tc.large && strings.Index(view, zlib.Version) < strings.Index(view, "█████") {
			t.Fatalf("%dx%d: version is above the wordmark", tc.width, tc.height)
		}
		for _, line := range strings.Split(view, "\n") {
			if strings.Contains(line, "/search") {
				both := strings.Contains(line, "/favorites")
				if both != (tc.columns == 2) {
					t.Fatalf("%dx%d: unexpected shortcut columns", tc.width, tc.height)
				}
			}
		}
		if tc.width >= 76 {
			commandStart := -1
			for _, line := range strings.Split(view, "\n") {
				for _, name := range []string{"/search", "/recommended", "/config"} {
					if start := strings.Index(line, name); start >= 0 {
						if commandStart >= 0 && commandStart != start {
							t.Fatal("shortcut columns do not align")
						}
						commandStart = start
					}
				}
			}
		}
	}
	logo := homeLogo(80, 24)
	if len(logo) != 7 {
		t.Fatalf("large wordmark has %d rows, want five letterform rows, a gap, and version", len(logo))
	}
	width := ansi.StringWidth(logo[0])
	for _, row := range logo {
		if ansi.StringWidth(row) != width {
			t.Fatal("wordmark and version rows have inconsistent width")
		}
	}
	if !strings.HasSuffix(ansi.Strip(logo[len(logo)-1]), zlib.Version) {
		t.Fatal("version is not aligned to the wordmark's right edge")
	}
	originalVersion := zlib.Version
	zlib.Version = "v0.0.9-0.20261015033749-d94e112583df+dirty"
	t.Cleanup(func() { zlib.Version = originalVersion })
	wideLogo := homeLogo(80, 24)
	if !strings.HasSuffix(ansi.Strip(wideLogo[len(wideLogo)-1]), ansi.Truncate(zlib.Version, width, "…")) {
		t.Fatal("long build version is not right-aligned below the wordmark")
	}
	if ansi.StringWidth(wideLogo[0]) > 80 || ansi.StringWidth(homeLogo(40, 16)[0]) > 40 {
		t.Fatal("version metadata exceeds the terminal width")
	}
	compactLogo := homeLogo(24, 12)
	if len(compactLogo) != 2 || ansi.StringWidth(compactLogo[0]) != ansi.StringWidth(compactLogo[1]) {
		t.Fatal("compact logo does not place version below its right edge")
	}
	for _, shortcut := range signedInShortcuts {
		if !slices.ContainsFunc(shellCommands, func(command shellCommand) bool { return command.name == shortcut.name }) {
			t.Fatalf("home shortcut /%s is not a shell command", shortcut.name)
		}
	}
}

func TestShellCommandPaletteWrapsAndKeepsSelectionVisible(t *testing.T) {
	m := newShellModel()
	m, _ = shellStep(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m.input.SetValue("/")
	m, _ = shellStep(m, key("up"))
	if m.selection != len(shellCommands)-1 || m.paletteOffset != len(shellCommands)-6 {
		t.Fatalf("up from first did not wrap to last: selection=%d offset=%d", m.selection, m.paletteOffset)
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "> /quit") {
		t.Fatal("wrapped selection is outside the visible command list")
	}
	m, _ = shellStep(m, key("down"))
	if m.selection != 0 || m.paletteOffset != 0 {
		t.Fatalf("down from last did not wrap to first: selection=%d offset=%d", m.selection, m.paletteOffset)
	}
	for range 5 {
		m, _ = shellStep(m, key("down"))
	}
	if m.paletteOffset != 0 {
		t.Fatal("palette scrolled before selection left the visible window")
	}
	m, _ = shellStep(m, key("down"))
	if m.paletteOffset != 1 {
		t.Fatalf("palette did not scroll at the visible edge: offset=%d", m.paletteOffset)
	}
	m.input.SetValue("/th")
	m.selection, m.paletteOffset = 0, 0
	m, _ = shellStep(m, key("up"))
	if m.selection != 0 || m.paletteOffset != 0 {
		t.Fatal("single filtered command should stay selected")
	}
}

func TestShellInformationUsesFullScreenAndScrolls(t *testing.T) {
	m := newShellModel()
	m, _ = shellStep(m, tea.WindowSizeMsg{Width: 40, Height: 16})
	m.openInfo("Available domains", strings.Join([]string{
		"domain 1", "domain 2", "domain 3", "domain 4", "domain 5", "domain 6",
		"domain 7", "domain 8", "domain 9", "domain 10", "domain 11", "domain 12",
	}, "\n"))
	view := m.View()
	if !view.AltScreen || len(strings.Split(view.Content, "\n")) != 16 || !strings.Contains(ansi.Strip(view.Content), "1–10 of 12") {
		t.Fatal("information page did not fill the viewport or show the scroll range")
	}
	m, _ = shellStep(m, key("end"))
	if m.infoOffset != 2 || !strings.Contains(ansi.Strip(m.View().Content), "domain 12") {
		t.Fatal("end did not reveal the final information row")
	}
	m, _ = shellStep(m, key("esc"))
	if m.kind != "" || m.status != "" {
		t.Fatal("closing information did not restore the home screen")
	}
}

func TestShellVersionIsNotACommand(t *testing.T) {
	m := newShellModel()
	m.input.SetValue("/ver")
	if len(m.matches()) != 0 {
		t.Fatal("version still appears in the command palette")
	}
	m.dispatch("/version")
	if m.kind != "" || !m.typing || m.status != "Unknown command: /version" {
		t.Fatal("version was accepted in the TUI")
	}
}

func TestShellSearchDraftSurvivesCommandPalette(t *testing.T) {
	m := newShellModel()
	m.input.SetValue("search")
	if matches := m.matches(); len(matches) != 0 {
		t.Fatalf("plain search text matched slash commands: %v", matches)
	}
	m, _ = shellStep(m, key("tab"))
	if m.input.Value() != "search" {
		t.Fatal("Tab changed a plain search query")
	}
	m, _ = shellStep(m, key("ctrl+k"))
	if m.input.Value() != "/" || m.paletteDraft != "search" {
		t.Fatal("Ctrl+K did not open commands while saving the draft")
	}
	m, _ = shellStep(m, key("esc"))
	if m.input.Value() != "search" || m.paletteDraft != "" {
		t.Fatal("Esc did not restore the search draft")
	}
}

func TestShellListUsesAvailableRowsAndRestoresFocus(t *testing.T) {
	m := newShellModel()
	m, _ = shellStep(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	list := newBookFeedSelectModel("Recommended", nil)
	list.state = searchStateResults
	list.books = []zlib.Book{{ID: "1", Hash: "abc", Name: "First"}}
	m.open("list", list)
	if got := m.page.(searchSelectModel).height; got != 23 {
		t.Fatalf("list content height = %d, want 23", got)
	}
	m, _ = shellStep(m, key("/"))
	if !m.typing || m.page.(searchSelectModel).height != 22 {
		t.Fatal("command input did not reserve its row")
	}
	m, _ = shellStep(m, key("esc"))
	if m.typing || m.page.(searchSelectModel).height != 23 || m.page.(searchSelectModel).cursor != 0 {
		t.Fatal("closing command input did not restore the list")
	}
	m.open("logout", shellConfirm{title: "Log out?", action: "Log out"})
	if got := m.page.(shellConfirm).height; got != 24 {
		t.Fatalf("dialog content height = %d, want full terminal", got)
	}
}

func TestShellSearchErrorCanRefineQuery(t *testing.T) {
	m := newShellModel()
	list := newSearchSelectModel("golang", nil, nil, false)
	list.state = searchStateResults
	list.err = errors.New("request timed out")
	list.retryPage = 1
	m.open("list", list)
	if !strings.Contains(ansi.Strip(m.View().Content), "ctrl+f search") {
		t.Fatalf("shell search error does not show its refine shortcut: %q", ansi.Strip(m.View().Content))
	}
	m, _ = shellStep(m, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	got := m.page.(searchSelectModel)
	if got.state != searchStateFilter || got.input.Value() != "golang" {
		t.Fatal("Ctrl+F did not open the failed search query")
	}
}
