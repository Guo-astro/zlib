package cli

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

func TestActionButtonsShareConfirmationStyle(t *testing.T) {
	for _, label := range []string{"Search", "Save", "Cancel", "Download here"} {
		button := renderActionButton(label, true, false)
		if ansi.Strip(button) != "  "+label+"  " || button == renderActionButton(label, false, false) {
			t.Fatalf("%q does not use the shared focused button style", label)
		}
	}
	if renderActionButton("Delete", true, true) == renderActionButton("Delete", true, false) {
		t.Fatal("dangerous action must use a distinct focused color")
	}
	rows := actionButtonRows([]string{"Download here", "Choose another folder"}, 1, 24)
	if len(rows) != 2 {
		t.Fatalf("narrow action group should wrap: %d rows", len(rows))
	}
	for _, row := range rows {
		if ansi.StringWidth(row) > 24 {
			t.Fatalf("action row exceeds viewport: %d", ansi.StringWidth(row))
		}
	}
}

func TestConfirmationButtonGapUsesPanelBackground(t *testing.T) {
	original := currentTheme
	t.Cleanup(func() { currentTheme = original })
	for _, name := range themeOrder {
		currentTheme = themes[name]
		for _, focused := range []bool{false, true} {
			view := confirmationDialog(fullScreenView(80, 24, "").Content, 80, 24, "Send to Kindle?", "Book", "", "", "Send", focused, false)
			found := false
			for _, line := range strings.Split(view, "\n") {
				plain := ansi.Strip(line)
				start := strings.Index(plain, "Cancel")
				if start < 0 {
					continue
				}
				found = true
				gap := ansi.StringWidth(plain[:start]) + len("Cancel") + 2
				cells := uv.NewStyledString(line).Lines(ansi.GraphemeWidth)[0]
				for x := gap; x < gap+2; x++ {
					bg := cells[x].Style.Bg
					if bg == nil || color.NRGBAModel.Convert(bg) != color.NRGBAModel.Convert(currentTheme.Surface) {
						t.Fatalf("%s focused=%v: button gap exposes a different background at column %d", name, focused, x)
					}
				}
			}
			if !found {
				t.Fatal("confirmation buttons are missing")
			}
		}
	}
}

func TestDownloadConfirmationAcrossBookLists(t *testing.T) {
	for _, mode := range []string{"search", "recommended", "most-popular", "saved"} {
		t.Run(mode, func(t *testing.T) {
			m := newSearchSelectModel("query", nil, nil, false)
			if mode == "recommended" || mode == "most-popular" {
				m.feed = func() ([]zlib.Book, error) { return nil, nil }
			}
			if mode == "saved" {
				m.feedPage = func(int) (zlib.SearchResult, error) { return zlib.SearchResult{}, nil }
			}
			m.state = searchStateResults
			m.books = []zlib.Book{{ID: "42", Hash: "abc", Name: "Book"}}
			m = step(m, key("enter"))
			if m.selectedID != "" || !strings.Contains(m.View().Content, "Download book?") {
				t.Fatal("missing confirmation")
			}
			if strings.Contains(m.View().Content, "Download the selected book.") {
				t.Fatal("download confirmation repeats its action")
			}
			m = step(m, key("enter"))
			if m.confirmDownload || m.selectedID != "" {
				t.Fatal("default Enter must cancel")
			}
			m = step(m, key("enter"))
			m = step(m, tea.KeyPressMsg{Code: tea.KeyRight})
			m = step(m, key("esc"))
			if m.selectedID != "" || m.cursor != 0 {
				t.Fatal("cancel changed selection")
			}
			m = step(m, key("enter"))
			if m.downloadFocused {
				t.Fatal("reopened dialog must default to cancel")
			}
			m = step(m, tea.KeyPressMsg{Code: tea.KeyRight})
			m = step(m, key("enter"))
			if m.selectedID != "42:abc" {
				t.Fatal("confirmation lost book reference")
			}
		})
	}
}
