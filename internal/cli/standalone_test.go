package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
	"github.com/heartleo/zlib/internal/config"
)

func TestStandaloneKindleSharesShellPages(t *testing.T) {
	isolateConfigHome(t)
	path := filepath.Join(t.TempDir(), "A book.epub")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"/kindle", "/kindle config", "/kindle send " + path} {
		standalone, err := newStandaloneShellCommand(command)
		if err != nil {
			t.Fatal(err)
		}
		shell := newShellModel()
		shell.dispatch(command)
		if standalone.kind != shell.kind || standalone.View().Content != shell.View().Content {
			t.Fatalf("%s differs between CLI and shell", command)
		}
		standalone, _ = shellStep(standalone, key("esc"))
		_, cmd := shellStep(standalone, shellEvent{generation: standalone.generation, msg: tea.QuitMsg{}})
		if cmd == nil {
			t.Fatalf("%s did not exit on cancellation", command)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%s returned to shell home instead of exiting", command)
		}
	}
}

func TestStandaloneKindlePasswordAndResult(t *testing.T) {
	isolateConfigHome(t)
	t.Setenv(kindleSMTPPasswordEnv, "")
	cfg := zlib.KindleConfig{To: "reader@kindle.com", From: "sender@example.com", SMTPHost: "smtp.example.com", SMTPPort: 587}
	if err := config.SaveKindleConfig(cfg); err != nil {
		t.Fatal(err)
	}
	m := newShellModel()
	m.standalone = true
	m.sendKindle("book.epub")
	if m.kind != "kindle-password" || !m.page.(inlineForm).fields[0].secret {
		t.Fatal("standalone send omitted the masked password prompt")
	}
	// Inject SMTP completion; this test never sends mail.
	m.open("kindle-send", kindleSendModel{filePath: "book.epub", cfg: cfg, done: true})
	m, _ = shellStep(m, shellEvent{generation: m.generation, msg: tea.QuitMsg{}})
	view := ansi.Strip(m.View().Content)
	if m.kind != "info" || !strings.Contains(view, "Email sent to Kindle") || !strings.Contains(view, cfg.To) || strings.Contains(view, "/ commands") {
		t.Fatal("send result did not stay on its own page")
	}
	_, cmd := shellStep(m, key("esc"))
	if cmd == nil {
		t.Fatal("result page did not exit")
	}
	failure := errors.New("SMTP rejected")
	m.open("kindle-send", kindleSendModel{filePath: "book.epub", err: failure})
	m, _ = shellStep(m, shellEvent{generation: m.generation, msg: tea.QuitMsg{}})
	if m.kind != "kindle-error" || !errors.Is(m.commandErr, failure) {
		t.Fatal("standalone send lost its error page or exit error")
	}
}

func TestStandaloneProfileMatchesShellPage(t *testing.T) {
	profile := zlib.UserProfile{Name: "Reader", QuotaKnown: true,
		DownloadLimit: zlib.DownloadLimit{DailyAmount: 1, DailyAllowed: 10, DailyRemaining: 9}}
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 40, Height: 16}} {
		direct := newProfilePageModel(nil)
		page, _ := direct.Update(size)
		page, _ = page.Update(profileLoadedMsg{profile: profile})
		shell := newShellModel()
		shell, _ = shellStep(shell, size)
		shell.open("profile", newProfilePageModel(nil))
		shell, _ = shellStep(shell, shellEvent{generation: shell.generation, msg: profileLoadedMsg{profile: profile}})
		if page.View().Content != shell.View().Content {
			t.Fatalf("profile layout differs at %dx%d", size.Width, size.Height)
		}
	}
}
