package cli

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
