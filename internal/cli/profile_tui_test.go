package cli

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

func TestProfilePageUsesFullScreenAndScrolls(t *testing.T) {
	premium := true
	m := newProfilePageModel(nil)
	profile := zlib.UserProfile{
		Name: "Reader", Email: "reader@example.com", KindleEmail: "kindle@example.com",
		Premium: &premium, QuotaKnown: true,
		DownloadLimit:   zlib.DownloadLimit{DailyAmount: 3, DailyAllowed: 10, DailyRemaining: 7},
		PersonalDomains: []string{"example.test", "books.example.test"},
	}
	for _, size := range [][2]int{{80, 24}, {30, 14}} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = next.(profilePageModel)
		next, _ = m.Update(profileLoadedMsg{profile: profile})
		m = next.(profilePageModel)
		view := m.View().Content
		lines := strings.Split(view, "\n")
		if len(lines) != size[1] {
			t.Fatalf("profile at %v uses %d rows", size, len(lines))
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("profile at %v exceeds width: %q", size, line)
			}
		}
		if !strings.Contains(ansi.Strip(view), "DAILY DOWNLOADS") || !strings.Contains(ansi.Strip(view), "3 / 10 used") {
			t.Fatalf("profile at %v omits quota", size)
		}
		if !strings.Contains(ansi.Strip(m.viewport.GetContent()), "books.example.test") {
			t.Fatal("profile content omits personal domains")
		}
	}
	before := m.viewport.YOffset()
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	m = next.(profilePageModel)
	if m.viewport.YOffset() <= before || !strings.Contains(ansi.Strip(m.View().Content), "kindle@example.com") {
		t.Fatalf("profile does not scroll to account details: offset=%d, view=%q", m.viewport.YOffset(), ansi.Strip(m.View().Content))
	}
}

func TestProfilePageUnknownQuota(t *testing.T) {
	page := renderProfilePage(zlib.UserProfile{Name: "Reader"}, 80)
	if !strings.Contains(ansi.Strip(page), "Quota unavailable") || strings.Contains(page, "0 / 0") {
		t.Fatalf("unknown quota rendered incorrectly: %q", page)
	}
}

func TestProfileQuotaBarUsesWidthAndDoesNotRepeatRemaining(t *testing.T) {
	profile := zlib.UserProfile{
		Name: "Reader", Email: "reader@example.com", KindleEmail: "kindle@example.com", QuotaKnown: true,
		DownloadLimit: zlib.DownloadLimit{DailyAmount: 1, DailyAllowed: 10, DailyRemaining: 9},
	}
	for _, width := range []int{20, 30, 40, 72, 80, 120} {
		page := renderProfilePage(profile, width)
		plain := ansi.Strip(page)
		if strings.Count(plain, "1 / 10 used") != 1 || strings.Contains(plain, "Remaining") {
			t.Fatalf("quota text is repeated at width %d: %q", width, plain)
		}
		barFound := false
		for _, line := range strings.Split(page, "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatalf("profile line exceeds width %d: %q", width, line)
			}
			if strings.Contains(ansi.Strip(line), "━") {
				barFound = true
				if ansi.StringWidth(line) != width || !strings.Contains(ansi.Strip(line), "─") {
					t.Fatalf("quota bar does not fill width %d: %q", width, line)
				}
			}
		}
		if !barFound {
			t.Fatalf("used quota is missing from the bar at width %d", width)
		}
		if width >= 72 {
			paired := false
			for _, line := range strings.Split(plain, "\n") {
				if strings.Contains(line, "Email") && strings.Contains(line, "Kindle email") {
					paired = true
				}
			}
			if !paired {
				t.Fatal("wide profile did not use account columns")
			}
		}
	}
}

func TestShellProfileOpensFullScreenPage(t *testing.T) {
	path := isolateShellSession(t)
	if err := os.WriteFile(path, []byte(`{"domain":"https://z-lib.gd","cookies":{"session":"test"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m := newShellModel()
	if cmd := m.dispatch("/profile"); cmd == nil {
		t.Fatal("profile did not start loading")
	}
	if m.kind != "profile" {
		t.Fatalf("profile opened %q", m.kind)
	}
	if _, ok := m.page.(profilePageModel); !ok {
		t.Fatalf("profile opened %T instead of its full-screen page", m.page)
	}
}
