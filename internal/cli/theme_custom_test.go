package cli

import (
	"image/color"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/heartleo/zlib/internal/config"
)

func TestCustomThemeCreateEditAndReload(t *testing.T) {
	isolateConfigHome(t)
	t.Setenv(themeEnvVar, "")
	base := themes[defaultTheme]
	if err := saveCustomTheme("reader-night", defaultTheme, base); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(selectableThemes(), "reader-night") || !isSelectableTheme("reader-night") {
		t.Fatal("saved theme is not selectable")
	}
	if err := config.SaveConfig(config.Config{Theme: "reader-night"}); err != nil {
		t.Fatal(err)
	}
	previous := currentTheme
	t.Cleanup(func() { currentTheme = previous })
	resolveTheme()
	if configuredThemeName() != "reader-night" || !reflect.DeepEqual(currentTheme, base) {
		t.Fatal("custom theme did not reload as the active palette")
	}
	m := newThemePicker("reader-night", "")
	next, _ := m.Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	m = next.(themePickerModel)
	if m.mode != "colors" || m.draftBase != defaultTheme {
		t.Fatal("custom theme did not open its color editor")
	}
	m.editor.SetValue("#123456")
	next, _ = m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m = next.(themePickerModel)
	if m.mode != "" || m.selected() != "reader-night" {
		t.Fatal("edited theme did not return to its list item")
	}
	p, err := loadCustomTheme("reader-night")
	if err != nil || themeColor(p.Background) != "#123456" {
		t.Fatalf("edited background did not persist: %v, %v", p.Background, err)
	}
	t.Setenv(themeEnvVar, "reader-night")
	resolveTheme()
	if themeColor(currentTheme.Background) != "#123456" {
		t.Fatal("environment override did not load the custom theme")
	}
}

func TestThemeCommandSelectsSavedCustomTheme(t *testing.T) {
	isolateConfigHome(t)
	t.Setenv(themeEnvVar, "")
	previous := currentTheme
	t.Cleanup(func() { currentTheme = previous })
	if err := saveCustomTheme("reading-light", "github", themes["github"]); err != nil {
		t.Fatal(err)
	}
	if err := themeCmd.RunE(themeCmd, []string{"reading-light"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig()
	if err != nil || cfg.Theme != "reading-light" || themeColor(currentTheme.Background) != themeColor(themes["github"].Background) {
		t.Fatalf("direct command did not activate custom theme: %q, %v", cfg.Theme, err)
	}
}

func TestCustomThemeValidationAndCancel(t *testing.T) {
	isolateConfigHome(t)
	for _, name := range []string{"../escape", "My Theme", "catppuccin-mocha", ""} {
		if err := validateCustomThemeName(name); err == nil {
			t.Fatalf("accepted unsafe theme name %q", name)
		}
	}
	m := newThemePicker(defaultTheme, "")
	next, _ := m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m = next.(themePickerModel)
	if m.mode != "name" {
		t.Fatal("new theme did not open name entry")
	}
	m.editor.SetValue("my-theme")
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(themePickerModel)
	if m.mode != "colors" {
		t.Fatal("valid name did not open color editor")
	}
	m.editor.SetValue("#oops")
	next, _ = m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m = next.(themePickerModel)
	if m.mode != "colors" || m.message == "" {
		t.Fatal("invalid color was saved")
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = next.(themePickerModel)
	if m.mode != "" || isSelectableTheme("my-theme") {
		t.Fatal("cancel wrote a custom theme")
	}
	if _, err := os.Stat(customThemePath("my-theme")); !os.IsNotExist(err) {
		t.Fatalf("cancel left a theme file: %v", err)
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m = next.(themePickerModel)
	m.editor.SetValue("my-theme")
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(themePickerModel)
	m.editor.SetValue("#112233")
	next, _ = m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m = next.(themePickerModel)
	if m.mode != "" || m.selected() != "my-theme" || themeColor(m.palette().Background) != "#112233" {
		t.Fatal("new theme was not saved and selected in the picker")
	}
	if _, err := config.LoadConfig(); !os.IsNotExist(err) {
		t.Fatalf("creating a palette changed the selected theme preference: %v", err)
	}
}

func TestThemePreviewBackgroundHasNoUnpaintedCells(t *testing.T) {
	for _, name := range []string{"github", "catppuccin-mocha"} {
		p := themes[name]
		for _, wide := range []bool{false, true} {
			preview := themePreview(p, 38, 10, wide)
			lines := uv.NewStyledString(preview).Lines(ansi.GraphemeWidth)
			for y, line := range lines {
				for x, cell := range line {
					if cell.Style.Bg == nil {
						t.Fatalf("%s wide=%v preview background missing at %d,%d", name, wide, x, y)
					}
					want := p.Background
					if (wide && y == 3) || (!wide && y == 4) {
						want = p.Surface
					}
					if color.RGBAModel.Convert(cell.Style.Bg) != color.RGBAModel.Convert(want) {
						t.Fatalf("%s wide=%v preview background at %d,%d is not the selected theme", name, wide, x, y)
					}
				}
			}
		}
	}
}

func TestThemePickerCustomThemeShowsPreview(t *testing.T) {
	isolateConfigHome(t)
	p, err := paletteFromColors(themes[defaultTheme], map[string]string{"background": "#123456"})
	if err != nil || saveCustomTheme("my-theme", defaultTheme, p) != nil {
		t.Fatal("could not create fixture theme")
	}
	m := newThemePicker(defaultTheme, "")
	m.search.SetValue("my-theme")
	m.filterNames()
	if m.selected() != "my-theme" || themeColor(m.palette().Background) != "#123456" || !strings.Contains(ansi.Strip(m.View().Content), "my-theme") {
		t.Fatal("custom theme did not appear in picker preview")
	}
}

func TestThemeEditorFitsAndKeepsPaintedBackground(t *testing.T) {
	isolateConfigHome(t)
	for _, size := range [][2]int{{100, 30}, {80, 24}, {40, 20}, {40, 17}, {20, 5}} {
		m := newThemePicker(defaultTheme, "")
		m.width, m.height = size[0], size[1]
		m.beginCustomTheme()
		m.draftName = "my-theme"
		m.mode = "colors"
		m.syncEditorColor()
		m.slot = len(themeColorSlots) - 1
		m.syncEditorColor()
		view := m.View()
		lines := strings.Split(view.Content, "\n")
		if len(lines) != size[1] {
			t.Fatalf("editor height at %v: %d", size, len(lines))
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("editor width exceeds %v", size)
			}
		}
		if size[1] >= 17 {
			plain := ansi.Strip(view.Content)
			if !strings.Contains(plain, "fb2") || !strings.Contains(plain, "save") {
				t.Fatalf("editor controls or selected slot clipped at %v", size)
			}
		}
		cells := uv.NewStyledString(view.Content).Lines(ansi.GraphemeWidth)
		for y, line := range cells {
			for x, cell := range line {
				if cell.Style.Bg == nil {
					t.Fatalf("editor background missing at %v cell %d,%d", size, x, y)
				}
			}
		}
	}
}
