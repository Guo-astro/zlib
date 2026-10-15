package cli

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestInlineLoginForm(t *testing.T) {
	m := newInlineForm("Log in", "Log in", []formField{
		{label: "Email", validate: requiredField("email")},
		{label: "Password", secret: true, validate: requiredField("password")},
	})
	step := func(msg tea.Msg) { next, _ := m.Update(msg); m = next.(inlineForm) }
	step(tea.PasteMsg{Content: "reader@example.test"})
	step(key("tab"))
	step(tea.PasteMsg{Content: "secret-fixture"})
	v := m.View().Content
	for _, label := range []string{"Email", "Password", "reader@example.test"} {
		if !strings.Contains(v, label) {
			t.Fatalf("missing visible field %s", label)
		}
	}
	if strings.Contains(v, "Not set") || strings.Contains(v, "secret-fixture") {
		t.Fatal("placeholder or password exposed")
	}
	step(key("tab"))
	step(key("enter"))
	if !m.accepted || m.fields[1].value != "secret-fixture" {
		t.Fatal("form values not submitted")
	}
}

func TestInlineSearchFormats(t *testing.T) {
	m := newInlineForm("Search books", "Search", []formField{
		{label: "Query", validate: requiredField("query")},
		{label: "Formats", options: []string{"EPUB", "PDF"}},
	})
	step := func(msg tea.Msg) { next, _ := m.Update(msg); m = next.(inlineForm) }
	step(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.accepted || m.cursor != 0 || !m.inputs[0].Focused() {
		t.Fatal("invalid form did not focus query")
	}
	step(tea.PasteMsg{Content: "test query"})
	step(key("tab"))
	step(tea.KeyPressMsg{Code: tea.KeySpace})
	step(key("tab"))
	step(key("enter"))
	if !m.accepted || len(m.fields[1].selected) != 1 || m.fields[1].selected[0] != "EPUB" {
		t.Fatal("format selection not submitted")
	}
}

func TestInlineSearchFormatComponentFitsAndSelects(t *testing.T) {
	formats := []string{"EPUB", "AZW", "AZW3", "MOBI", "PDF", "RTF", "TXT"}
	m := newInlineForm("Search books", "Search", []formField{
		{label: "Query"},
		{label: "Formats", options: formats, hint: "No selection includes all formats"},
	})
	step := func(msg tea.Msg) { next, _ := m.Update(msg); m = next.(inlineForm) }
	for _, size := range [][2]int{{80, 24}, {40, 16}, {30, 14}} {
		step(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		step(key("tab"))
		view := ansi.Strip(m.View().Content)
		if strings.Count(view, "Formats") != 1 || !strings.Contains(view, "[ ] EPUB") || strings.Count(view, "Search") < 2 {
			t.Fatalf("format component missing or duplicated at %v:\n%s", size, view)
		}
		if len(strings.Split(view, "\n")) > size[1] {
			t.Fatalf("search form exceeds height at %v", size)
		}
		for _, line := range strings.Split(m.View().Content, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("search form exceeds width at %v: %q", size, line)
			}
		}
		step(key("shift+tab"))
	}
	step(key("tab"))
	for range len(formats) - 1 {
		step(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "[ ] TXT") {
		t.Fatal("narrow format list did not scroll to the last option")
	}
	step(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	if len(m.fields[1].selected) != len(formats) {
		t.Fatalf("select all chose %d formats", len(m.fields[1].selected))
	}
	step(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	if len(m.fields[1].selected) != 0 {
		t.Fatal("select none did not clear formats")
	}
}
