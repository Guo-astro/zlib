package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

func searchFormStep(m searchFilterForm, msg tea.Msg) searchFilterForm {
	next, _ := m.Update(msg)
	return next.(searchFilterForm)
}

func TestSearchFilterFormSelectsEveryEAPIConstraint(t *testing.T) {
	m := newSearchFilterForm("books", nil)
	m.yearFrom, m.yearTo = "2001", "2020"
	for range 3 {
		m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeyTab})
	}
	if m.cursor != 3 {
		t.Fatalf("focus = %d, want languages", m.cursor)
	}
	m = searchFormStep(m, key("enter"))
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeySpace})
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeySpace})
	m = searchFormStep(m, key("esc"))
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = searchFormStep(m, key("enter"))
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeySpace})
	m = searchFormStep(m, key("esc"))
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeySpace})
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeyTab})
	m = searchFormStep(m, key("enter"))
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = searchFormStep(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = searchFormStep(m, key("enter"))
	m = searchFormStep(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if !m.accepted {
		t.Fatalf("form rejected filters: %s", m.message)
	}
	opts := m.searchOptions()
	if opts.FromYear != 2001 || opts.ToYear != 2020 || !opts.Exact || opts.Order != zlib.OrderBestMatch ||
		!slices.Equal(opts.Languages, []zlib.Language{zlib.LangEnglish, zlib.LangChinese}) ||
		!slices.Equal(opts.Extensions, []zlib.Extension{zlib.ExtEPUB}) {
		t.Fatalf("search options = %+v", opts)
	}
}

func TestSearchFilterFormValidatesYearsAndFits(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {40, 16}, {30, 12}} {
		m := searchFormStep(newSearchFilterForm("books", nil), tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		view := m.View().Content
		if size[1] >= 16 && !strings.Contains(ansi.Strip(view), "Search") {
			t.Fatalf("search action hidden at %v", size)
		}
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("line exceeds %v", size)
			}
		}
		m.cursor, m.editing, m.picker = 3, false, 3
		view = m.View().Content
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("picker exceeds %v", size)
			}
		}
		m.picker, m.cursor = -1, 7
		if !strings.Contains(ansi.Strip(m.View().Content), "Search") {
			t.Fatalf("focused Search action is hidden at %v", size)
		}
	}
	m := newSearchFilterForm("books", nil)
	m.yearFrom, m.yearTo = "2021", "2020"
	m = searchFormStep(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.accepted || !strings.Contains(m.message, "before") {
		t.Fatalf("invalid range accepted: %q", m.message)
	}
}

func TestSearchFilterFormFeedsPagedEAPIRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		for key, want := range map[string]string{"message": "books", "page": "2", "limit": "50", "yearFrom": "2001", "yearTo": "2020", "languages[]": "chinese", "extensions[]": "epub", "e": "1", "order": "bestmatch"} {
			if got := r.Form.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		fmt.Fprint(w, `{"success":1,"books":[],"pagination":{"current":2,"total_pages":3}}`)
	}))
	defer server.Close()
	client := zlib.NewClient(zlib.WithDomain(server.URL))
	client.SetCookies(map[string]string{"remix_userid": "1", "remix_userkey": "k"})
	form := newSearchFilterForm("books", &zlib.SearchOptions{
		FromYear: 2001, ToYear: 2020, Languages: []zlib.Language{zlib.LangChinese},
		Extensions: []zlib.Extension{zlib.ExtEPUB}, Exact: true, Order: zlib.OrderBestMatch,
	})
	form = searchFormStep(form, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if !form.accepted {
		t.Fatalf("form rejected: %s", form.message)
	}
	model := newSearchSelectModel(form.query, client, form.searchOptions(), false)
	if _, ok := model.fetchPage(2)().(searchResultMsg); !ok {
		t.Fatal("paged search failed")
	}
}

func TestSearchRefineKeepsAndAppliesFilters(t *testing.T) {
	opts := &zlib.SearchOptions{FromYear: 2010, ToYear: 2020, Languages: []zlib.Language{zlib.LangChinese}, Extensions: []zlib.Extension{zlib.ExtPDF}, Exact: true, Order: zlib.OrderDate}
	m := newSearchSelectModel("books", nil, opts, false)
	m.state = searchStateResults
	m.page, m.totalPages = 3, 5
	m = step(m, key("/"))
	if m.state != searchStateFilter || m.filter.query != "books" || m.filter.order != zlib.OrderDate ||
		m.filter.yearFrom != "2010" || m.filter.yearTo != "2020" || !m.filter.exact ||
		!slices.Equal(m.filter.languages, opts.Languages) || !slices.Equal(m.filter.extensions, opts.Extensions) {
		t.Fatalf("refine form lost filters: %+v", m.filter)
	}
	m.filter.order = zlib.OrderBestMatch
	m = step(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.state != searchStateLoading || m.page != 1 || m.opts.Order != zlib.OrderBestMatch || !m.opts.Exact {
		t.Fatalf("refine did not apply filters: state=%v page=%d opts=%+v", m.state, m.page, m.opts)
	}
}
