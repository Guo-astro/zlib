package cli

import (
	"os"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/heartleo/zlib/internal/config"
)

func themePickerKey(m themePickerModel, code rune) themePickerModel {
	next, _ := m.Update(tea.KeyPressMsg{Code: code})
	return next.(themePickerModel)
}

func TestThemePickerPreviewDoesNotSave(t *testing.T) {
	isolateConfigHome(t)
	if err := config.SaveConfig(config.Config{Theme: defaultTheme}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	palette := currentTheme
	m := newThemePicker(defaultTheme, "")
	if m.selected() != defaultTheme {
		t.Fatal("current theme not selected")
	}
	m = themePickerKey(m, tea.KeyHome)
	if m.selected() != m.names[0] || !reflect.DeepEqual(m.palette(), themes[m.names[0]]) {
		t.Fatal("preview did not use selected palette")
	}
	_ = m.View()
	m = themePickerKey(m, tea.KeyEscape)
	if m.accepted {
		t.Fatal("cancel accepted selection")
	}
	after, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) || !reflect.DeepEqual(palette, currentTheme) {
		t.Fatal("preview changed persistent or global theme")
	}
}

func TestThemePickerNavigationAndAcceptance(t *testing.T) {
	m := newThemePicker("missing", "")
	m = themePickerKey(m, tea.KeyUp)
	if m.cursor != 0 {
		t.Fatal("cursor went before first theme")
	}
	m = themePickerKey(m, tea.KeyEnd)
	m = themePickerKey(m, tea.KeyDown)
	if m.cursor != len(m.names)-1 {
		t.Fatal("cursor went past final theme")
	}
	m = themePickerKey(m, tea.KeyPgUp)
	if m.cursor != len(m.names)-1-m.visibleRows() {
		t.Fatal("page up did not move by one visible page")
	}
	m = themePickerKey(m, tea.KeyHome)
	m = themePickerKey(m, tea.KeyEnter)
	if !m.accepted || m.selected() != m.names[0] {
		t.Fatal("Enter did not accept selection")
	}
}

func TestThemePickerFiltersAndRestoresList(t *testing.T) {
	m := newThemePicker("catppuccin-mocha", "")
	for _, r := range "snazzy" {
		next, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = next.(themePickerModel)
	}
	if len(m.names) != 1 || m.selected() != "sublime-snazzy" || !strings.Contains(ansi.Strip(m.View().Content), "Preview: sublime-snazzy") {
		t.Fatalf("filter did not select matching theme: %v", m.names)
	}
	m = themePickerKey(m, tea.KeyEscape)
	if m.accepted || len(m.names) != len(m.allNames) {
		t.Fatal("Esc did not clear the filter")
	}
	m = themePickerKey(m, tea.KeyEscape)
	if m.accepted {
		t.Fatal("Esc accepted a theme")
	}
}

func TestThemePickerBookPreviewAdaptsToWidth(t *testing.T) {
	m := newThemePicker("dracula", "")
	m.width, m.height = 100, 24
	wide := ansi.Strip(m.View().Content)
	if !strings.Contains(wide, `zlib search "example"`) || !strings.Contains(wide, "Example Author") || !strings.Contains(wide, "Save to  ./books") {
		t.Fatal("wide picker did not show the full zlib preview")
	}
	m.width, m.height = 40, 17
	narrow := ansi.Strip(m.View().Content)
	if !strings.Contains(narrow, `zlib search "example"`) || !strings.Contains(narrow, "EPUB · 4 MB") || !strings.Contains(narrow, "Download complete") || strings.Contains(narrow, "Example Author") {
		t.Fatal("narrow picker did not use its compact zlib preview")
	}
}

func TestThemePreviewRowsFillThePanel(t *testing.T) {
	for _, name := range selectableThemes() {
		p := themes[name]
		if p.Background == nil {
			t.Fatalf("%s has no preview background", name)
		}
		for _, wide := range []bool{false, true} {
			preview := themePreview(p, 38, 10, wide)
			for _, line := range strings.Split(preview, "\n") {
				if got := ansi.StringWidth(line); got != 38 {
					t.Fatalf("%s wide=%t: preview row width %d, want 38", name, wide, got)
				}
			}
		}
	}
}

func TestThemePickerEmptyFilterCannotSave(t *testing.T) {
	m := newThemePicker("dracula", "")
	m.search.SetValue("missing-theme-name")
	m.filterNames()
	if m.selected() != "" || !strings.Contains(ansi.Strip(m.View().Content), "No matching themes") {
		t.Fatal("empty filter did not show recovery text")
	}
	m = themePickerKey(m, tea.KeyEnter)
	if m.accepted {
		t.Fatal("empty filter accepted a theme")
	}
}

func TestThemePickerViewport(t *testing.T) {
	for _, size := range [][2]int{{120, 30}, {90, 17}, {80, 24}, {40, 17}, {32, 17}, {20, 5}} {
		for _, name := range selectableThemes() {
			m := newThemePicker(name, "")
			m.width, m.height = size[0], size[1]
			v := m.View()
			if !v.AltScreen {
				t.Fatal("theme picker must use alternate screen")
			}
			lines := strings.Split(v.Content, "\n")
			if len(lines) > m.height {
				t.Fatalf("%v: %d lines", size, len(lines))
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > m.width {
					t.Fatalf("%v: line exceeds width: %q", size, line)
				}
			}
			if m.width >= 32 && m.height >= 17 {
				plain := ansi.Strip(v.Content)
				if !strings.Contains(plain, "> "+name) || !strings.Contains(plain, "cancel") || !strings.Contains(plain, "Preview") {
					t.Fatalf("%v: selection or help missing:\n%s", size, plain)
				}
			}
		}
	}
}
