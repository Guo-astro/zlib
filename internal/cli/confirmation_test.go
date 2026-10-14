package cli

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	zlib "github.com/heartleo/zlib"
)

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
