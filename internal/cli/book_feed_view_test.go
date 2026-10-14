package cli

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

func feedTestModel() searchSelectModel {
	m := newBookFeedSelectModel("Browse recommended books", nil)
	m.feed = func() ([]zlib.Book, error) { return nil, nil }
	m.state = searchStateResults
	for i := range 100 {
		m.books = append(m.books, zlib.Book{ID: fmt.Sprint(i + 1), Hash: "abcdef", Name: fmt.Sprintf("书籍 %d：中文与 English 长标题", i+1), Authors: []string{"作者 Author"}})
	}
	return m
}

func TestFeedFitsTerminalAndKeepsSelectionVisible(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {40, 16}, {30, 14}, {20, 5}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			m := feedTestModel()
			next, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			m = next.(searchSelectModel)
			m = step(m, tea.KeyPressMsg{Code: tea.KeyEnd})
			view := m.View()
			if !view.AltScreen {
				t.Fatal("feed must use alternate screen")
			}
			lines := strings.Split(view.Content, "\n")
			if len(lines) > size[1] {
				t.Fatalf("%d lines exceed height %d", len(lines), size[1])
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > size[0] {
					t.Fatalf("line exceeds width %d: %q", size[0], line)
				}
			}
			if size[0] >= 30 && !strings.Contains(ansi.Strip(view.Content), "> 100") {
				t.Fatal("last selected book is not visible")
			}
		})
	}
}

func TestFeedNavigationAndDownloadID(t *testing.T) {
	m := feedTestModel()
	m.width, m.height = 80, 24
	m = step(m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	if m.cursor != m.feedRows() {
		t.Fatalf("page down cursor = %d", m.cursor)
	}
	m = step(m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	if m.cursor != 0 {
		t.Fatalf("page up cursor = %d", m.cursor)
	}
	m = step(m, tea.KeyPressMsg{Code: tea.KeyEnd})
	if !strings.Contains(m.View().Content, "100:abcdef") {
		t.Fatal("preview lost the complete ID")
	}
	m = step(m, key("enter"))
	m = step(m, tea.KeyPressMsg{Code: tea.KeyRight})
	m = step(m, key("enter"))
	if m.selectedID != "100:abcdef" {
		t.Fatalf("download ID = %q", m.selectedID)
	}
	m = step(m, tea.KeyPressMsg{Code: tea.KeyHome})
	if m.cursor != 0 {
		t.Fatal("home did not select the first book")
	}
}

func TestFeedEmptyAndLoadingHaveExitHint(t *testing.T) {
	m := feedTestModel()
	m.books = nil
	for _, state := range []searchState{searchStateLoading, searchStateResults} {
		m.state = state
		if !strings.Contains(m.View().Content, "q quit") {
			t.Fatal("missing exit hint")
		}
	}
}

func TestSearchCardTracksSelectionAndFits(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 16}, {30, 14}} {
		m := feedTestModel()
		m.feed = nil
		m.query = "books"
		m.width, m.height = size[0], size[1]
		m.books[0].Authors = []string{"First author"}
		m.books[1].Authors = []string{"Second author"}
		m.books[1].Extension = "epub"
		before := ansi.Strip(m.View().Content)
		m = step(m, tea.KeyPressMsg{Code: tea.KeyDown})
		after := ansi.Strip(m.View().Content)
		cardEnd := strings.Index(after, "╰")
		if cardEnd < 0 || !strings.Contains(after[:cardEnd], "Second author") || !strings.Contains(after[:cardEnd], "EPUB") || strings.Contains(after[:cardEnd], "First author") {
			t.Fatal("card did not track selection")
		}
		if strings.Count(before[:strings.Index(before, "╰")], "\n") != strings.Count(after[:cardEnd], "\n") {
			t.Fatal("card height changed on selection")
		}
		if len(strings.Split(after, "\n")) > size[1] {
			t.Fatal("search exceeds terminal height")
		}
		for _, line := range strings.Split(after, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("search exceeds terminal width: %q", line)
			}
		}
	}
}
