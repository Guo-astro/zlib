package cli

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m *themePickerModel) beginCustomTheme() {
	m.mode = "name"
	m.message = ""
	m.draftName = ""
	m.draftBase = m.selected()
	if _, ok := themes[m.draftBase]; !ok {
		m.draftBase = defaultTheme
	}
	m.draftPalette = m.palette()
	m.editor = textinput.New()
	m.editor.Prompt = "Name  "
	m.editor.Placeholder = "my-theme"
	m.editor.Focus()
}

func (m *themePickerModel) editSelectedTheme() {
	name := m.selected()
	if name == "" {
		return
	}
	if slicesContainsTheme(themeOrder, name) {
		m.beginCustomTheme()
		return
	}
	m.draftName = name
	m.draftBase = name
	// Saved files contain their built-in base so edits remain reproducible.
	m.draftBase = defaultTheme
	if file, err := readCustomThemeFile(name); err == nil {
		m.draftBase = file.Base
	}
	m.draftPalette = m.palette()
	m.slot = 0
	m.mode = "colors"
	m.message = ""
	m.syncEditorColor()
}

func (m *themePickerModel) syncEditorColor() {
	m.editor = textinput.New()
	m.editor.Prompt = themeColorSlots[m.slot] + "  "
	m.editor.SetValue(themeColors(m.draftPalette)[themeColorSlots[m.slot]])
	m.editor.CursorEnd()
	m.editor.Focus()
}

func (m *themePickerModel) applyEditorColor() bool {
	value := strings.TrimSpace(m.editor.Value())
	if !themeColorPattern.MatchString(value) {
		m.message = "Enter a color as #RRGGBB"
		return false
	}
	palette, err := paletteFromColors(m.draftPalette, map[string]string{themeColorSlots[m.slot]: value})
	if err != nil {
		m.message = err.Error()
		return false
	}
	m.draftPalette = palette
	m.message = ""
	return true
}

func (m themePickerModel) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		m.editor.SetWidth(max(1, m.width-18))
		return m, nil
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch m.mode {
		case "name":
			switch key.String() {
			case "esc":
				m.mode = ""
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.editor.Value())
				if err := validateCustomThemeName(name); err != nil {
					m.message = err.Error()
					return m, nil
				}
				m.draftName = name
				m.mode = "colors"
				m.slot = 0
				m.message = ""
				m.syncEditorColor()
				return m, nil
			}
		case "colors":
			switch key.String() {
			case "esc":
				m.mode = ""
				m.message = ""
				return m, nil
			case "up", "down", "tab", "shift+tab":
				if !m.applyEditorColor() {
					return m, nil
				}
				if key.String() == "up" || key.String() == "shift+tab" {
					m.slot = (m.slot - 1 + len(themeColorSlots)) % len(themeColorSlots)
				} else {
					m.slot = (m.slot + 1) % len(themeColorSlots)
				}
				m.syncEditorColor()
				return m, nil
			case "ctrl+s":
				if !m.applyEditorColor() {
					return m, nil
				}
				if err := saveCustomTheme(m.draftName, m.draftBase, m.draftPalette); err != nil {
					m.message = err.Error()
					return m, nil
				}
				m.mode = ""
				m.search.SetValue("")
				m.allNames = selectableThemes()
				m.names = append([]string(nil), m.allNames...)
				for i, name := range m.names {
					if name == m.draftName {
						m.cursor = i
						break
					}
				}
				m.ensureVisible()
				m.message = "Saved " + m.draftName
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	if m.mode == "colors" {
		m.applyEditorColor()
	} else {
		m.message = ""
	}
	return m, cmd
}

func slicesContainsTheme(names []string, name string) bool {
	for _, item := range names {
		if item == name {
			return true
		}
	}
	return false
}

func (m themePickerModel) editorView() tea.View {
	w, h := max(1, m.width), max(1, m.height)
	if m.mode == "name" {
		view := screenView(w, h, "New theme", "Copy of "+m.draftBase,
			"  Enter a name for the new theme.\n\n  "+m.editor.View(), m.message,
			uiKey("enter", "continue"), uiKey("esc", "cancel"))
		view.SetContent(paintThemeCanvas(view.Content, w, h, m.chromePalette()))
		return view
	}
	p := m.draftPalette
	showPreview := h >= 20
	rows := max(1, min(len(themeColorSlots), h-8))
	if showPreview {
		rows = max(1, min(len(themeColorSlots), h-13))
	}
	start := min(max(0, m.slot-rows+1), max(0, len(themeColorSlots)-rows))
	if m.slot < rows {
		start = 0
	}
	lines := make([]string, 0, rows+3)
	values := themeColors(p)
	for i := start; i < min(len(themeColorSlots), start+rows); i++ {
		name := themeColorSlots[i]
		label := fmt.Sprintf("  %-12s %s", name, values[name])
		if i == m.slot {
			label = "> " + strings.TrimPrefix(label, "  ")
		}
		style := lipgloss.NewStyle().Foreground(currentTheme.Title)
		if i == m.slot {
			style = style.Background(currentTheme.Surface).Bold(true)
		}
		lines = append(lines, style.Render(ansi.Truncate(label, w, "…")))
	}
	lines = append(lines, "", "  "+m.editor.View())
	previewWidth := min(w-2, 44)
	if previewWidth > 0 && showPreview {
		lines = append(lines, "", themePreview(p, previewWidth, 4, false))
	}
	view := screenView(w, h, "Edit theme: "+m.draftName, "Base: "+m.draftBase,
		strings.Join(lines, "\n"), m.message,
		uiKey("↑/↓", "color"), uiKey("ctrl+s", "save"), uiKey("esc", "cancel"))
	view.SetContent(paintThemeCanvas(view.Content, w, h, m.chromePalette()))
	return view
}
