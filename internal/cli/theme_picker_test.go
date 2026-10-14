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
	if err := config.SaveConfig(config.Config{Theme: "mocha"}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	palette := currentTheme
	m := newThemePicker("mocha", "")
	if m.selected() != "mocha" {
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
	if m.cursor != 0 {
		t.Fatal("page up did not reach start")
	}
	m = themePickerKey(m, tea.KeyEnter)
	if !m.accepted || m.selected() != m.names[0] {
		t.Fatal("Enter did not accept selection")
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
