package cli

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib/internal/zlib"
)

func formStep(m formModel, msg tea.Msg) formModel { next, _ := m.Update(msg); return next.(formModel) }

func TestPasswordPaste(t *testing.T) {
	for _, shortcut := range []tea.KeyPressMsg{
		{Code: 'v', Mod: tea.ModCtrl},
		{Code: 'v', Mod: tea.ModCtrl | tea.ModShift},
		{Code: tea.KeyInsert, Mod: tea.ModShift},
	} {
		t.Run(shortcut.String(), func(t *testing.T) {
			m := newFormModel("Log in", "Log in", []formField{{label: "Password", secret: true}})
			m = formStep(m, key("enter"))
			_, cmd := m.Update(shortcut)
			if cmd == nil {
				t.Fatal("paste shortcut did not request clipboard")
			}
			const password = "test 密码!q+123"
			m = formStep(m, tea.PasteMsg{Content: password})
			if m.input.Value() != password || strings.Contains(m.View().Content, password) {
				t.Fatal("password paste lost or exposed")
			}
			m = formStep(m, key("enter"))
			if m.fields[0].value != password || strings.Contains(m.View().Content, password) {
				t.Fatal("applied password lost or exposed")
			}
		})
	}
}

func TestFormEditsValidateAndMask(t *testing.T) {
	m := newFormModel("Log in", "Log in", []formField{{label: "Email", validate: requiredField("email")}, {label: "Password", secret: true, value: "private-value"}})
	m = formStep(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.accepted || m.message == "" {
		t.Fatal("empty email accepted")
	}
	m = formStep(m, key("enter"))
	m = formStep(m, tea.PasteMsg{Content: "reader@example.test"})
	m = formStep(m, key("enter"))
	if m.fields[0].value != "reader@example.test" {
		t.Fatal("paste lost")
	}
	m = formStep(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = formStep(m, key("enter"))
	if strings.Contains(m.View().Content, "private-value") {
		t.Fatal("password exposed in input")
	}
	m.input.SetValue("changed")
	m = formStep(m, key("esc"))
	if m.fields[1].value != "private-value" || strings.Contains(m.View().Content, "private-value") {
		t.Fatal("cancel changed password or exposed it")
	}
	m = formStep(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if !m.accepted {
		t.Fatal("valid form not accepted")
	}
}

func TestFormatSelectionCanClearExistingFilter(t *testing.T) {
	m := newFormModel("Search", "Search", []formField{{label: "Formats", options: []string{"EPUB", "PDF"}, selected: []string{"EPUB"}}})
	m = formStep(m, key("enter"))
	m = formStep(m, tea.KeyPressMsg{Code: tea.KeySpace})
	m = formStep(m, key("enter"))
	if len(m.fields[0].selected) != 0 || !strings.Contains(m.View().Content, "All formats") {
		t.Fatal("cannot clear filter")
	}
}

func TestAllNewScreensFitTerminal(t *testing.T) {
	for _, size := range [][2]int{{120, 30}, {80, 24}, {40, 16}, {32, 12}, {20, 5}} {
		form := newFormModel("Search", "Search", []formField{{label: "Query", value: strings.Repeat("书籍", 100)}, {label: "Formats", options: []string{"EPUB", "PDF"}}})
		form.cursor = 1
		file := newKindleFileModel(t.TempDir())
		settings, _ := newSettingsModel(t.Context(), testSettingsStore(t), "download")
		settings.edit(downloadDirEnvVar)
		download := newDownloadModel("42:abcdef", t.TempDir(), nil)
		download.state = stateDownloading
		download.bookName = strings.Repeat("中文", 100)
		kindle := newKindleSendModel("book.epub", zlib.KindleConfig{To: "reader@example.test"}, "")
		models := []tea.Model{form, file, settings, download, kindle, newDownloadDirectoryModel(t.TempDir())}
		for _, model := range models {
			model, _ = model.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			v := model.View()
			if !v.AltScreen {
				t.Fatalf("%T is not full screen", model)
			}
			if len(strings.Split(v.Content, "\n")) > size[1] {
				t.Fatalf("%T height exceeds %v", model, size)
			}
			for _, line := range strings.Split(v.Content, "\n") {
				if ansi.StringWidth(line) > size[0] {
					t.Fatalf("%T width exceeds %v", model, size)
				}
			}
		}
	}
}

func TestSavedUsesBoundedFeed(t *testing.T) {
	m := newSavedBooksSelectModel(nil)
	m.feedPage = func(int) (zlib.SearchResult, error) { return zlib.SearchResult{}, nil }
	m.state = searchStateResults
	m.width, m.height = 40, 16
	for i := range 100 {
		m.books = append(m.books, zlib.Book{ID: fmt.Sprint(i + 1), Hash: "abcdef", Name: "中文标题"})
	}
	m = step(m, tea.KeyPressMsg{Code: tea.KeyEnd})
	view := m.View()
	if !view.AltScreen || !strings.Contains(view.Content, "> 100") || len(strings.Split(view.Content, "\n")) > 16 {
		t.Fatal("saved list is not bounded")
	}
}

func TestListFailureCanRetryWithoutActingOnStaleResults(t *testing.T) {
	m := feedTestModel()
	next, cmd := m.Update(listLoadError{errors.New("offline"), 2})
	m = next.(searchSelectModel)
	if cmd != nil || !strings.Contains(m.View().Content, "retry") {
		t.Fatal("failure must remain open with retry")
	}
	m = step(m, key("enter"))
	if m.confirmDownload {
		t.Fatal("acted on hidden results")
	}
	next, cmd = m.Update(key("r"))
	m = next.(searchSelectModel)
	if cmd == nil || m.state != searchStateLoading || m.err != nil {
		t.Fatal("retry did not start")
	}
	next, _ = m.Update(searchResultMsg{page: 2})
	m = next.(searchSelectModel)
	if m.err != nil || m.page != 2 {
		t.Fatal("retry did not recover")
	}
	h := newHistorySelectModel(nil)
	hn, hc := h.Update(listLoadError{errors.New("offline"), 3})
	h = hn.(historySelectModel)
	if hc != nil || !strings.Contains(h.View().Content, "retry") {
		t.Fatal("history failure exited")
	}
	hn, hc = h.Update(key("r"))
	h = hn.(historySelectModel)
	if hc == nil || h.err != nil || h.state != historyStateLoading {
		t.Fatal("history retry missing")
	}
}
