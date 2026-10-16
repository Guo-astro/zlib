package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

func TestSearchInputHandlesUnicodeAndPaste(t *testing.T) {
	m := step(resultsModel(""), key("/"))
	m = step(m, tea.KeyPressMsg{Code: '三', Text: "三"})
	m = step(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	next, _ := m.Update(tea.PasteMsg{Content: "体 physics"})
	m = next.(searchSelectModel)
	if got := m.filter.input.Value(); got != "三 体 physics" {
		t.Fatalf("input = %q", got)
	}
	m = step(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
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

func TestDownloadCompletePathWrapsAndWaitsForReturn(t *testing.T) {
	path := filepath.Join(t.TempDir(), strings.Repeat("long-folder/", 12), "book.epub")
	m := newDownloadModel("42", ".", nil)
	m.width, m.height = 32, 12
	m.stayOnComplete = true
	next, cmd := m.Update(downloadDoneMsg{path: path, size: 42})
	m = next.(downloadModel)
	if cmd != nil || m.state != stateComplete || m.savedPath != path {
		t.Fatal("completion did not wait with the full saved path")
	}
	if got := strings.Join(m.completeLines()[1:], ""); got != path {
		t.Fatalf("wrapped path changed: %q", got)
	}
	if m.maxCompleteOffset() == 0 {
		t.Fatal("long path did not enable scrolling")
	}
	for _, line := range strings.Split(m.View().Content, "\n") {
		if ansi.StringWidth(line) > m.width {
			t.Fatal("completion page exceeded terminal width")
		}
	}
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "Send to Kindle") || !strings.Contains(view, "Back") || strings.Contains(view, "enter back") || strings.Contains(view, "esc back") {
		t.Fatalf("completion actions are unclear: %q", view)
	}
	next, cmd = m.Update(key("down"))
	m = next.(downloadModel)
	if cmd != nil || m.completeOffset != 1 {
		t.Fatal("path did not scroll")
	}
	for _, keyName := range []string{"esc", "enter"} {
		_, cmd = m.Update(key(keyName))
		if cmd == nil {
			t.Fatalf("%s did not leave the completion page", keyName)
		}
	}
	next, _ = m.Update(key("right"))
	m = next.(downloadModel)
	if m.completeAction != 1 {
		t.Fatal("Kindle action was not selected")
	}
	next, cmd = m.Update(key("enter"))
	if cmd == nil || !next.(downloadModel).sendToKindle {
		t.Fatal("Kindle action did not route to the shell")
	}
	m.stayOnComplete = false
	_, cmd = m.Update(downloadDoneMsg{path: path, size: 42})
	if cmd == nil {
		t.Fatal("standalone download no longer exits after completion")
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
