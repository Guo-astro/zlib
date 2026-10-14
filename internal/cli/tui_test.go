package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zlib "github.com/heartleo/zlib"
)

func TestSearchInputHandlesUnicodeAndPaste(t *testing.T) {
	m := step(resultsModel(""), key("/"))
	m = step(m, tea.KeyPressMsg{Code: '三', Text: "三"})
	m = step(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	next, _ := m.Update(tea.PasteMsg{Content: "体 physics"})
	m = next.(searchSelectModel)
	if got := m.input.Value(); got != "三 体 physics" {
		t.Fatalf("input = %q", got)
	}
	m = step(m, key("enter"))
	if m.query != "三 体 physics" || m.state != searchStateLoading {
		t.Fatalf("query = %q, state = %v", m.query, m.state)
	}
}

func TestDownloadCtrlCCancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	m := newDownloadModel("42", t.TempDir(), nil)
	m.cancel = cancel
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if next.(downloadModel).state != stateCancelled || ctx.Err() == nil {
		t.Fatal("Ctrl+C did not cancel the download")
	}
	if cmd == nil {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("Ctrl+C returned a non-quit command")
	}
}

func TestStyledOutputToPipeHasNoANSI(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var out bytes.Buffer
	_, err := lipgloss.Fprintln(&out, renderProfileCard(zlib.UserProfile{}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Fatalf("output contains terminal escapes: %q", out.String())
	}
}
