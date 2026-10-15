package cli

import (
	"strings"

	"charm.land/bubbles/v2/help"
	keys "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// screenView keeps the title and controls in place as the body changes.
func screenView(width, height int, title, subtitle, body, status string, bindings ...keys.Binding) tea.View {
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	width, height = max(1, width), max(1, height)
	muted := lipgloss.NewStyle().Foreground(currentTheme.Muted)
	h := help.New()
	h.SetWidth(width)
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	h.Styles.ShortDesc = muted
	footer := h.ShortHelpView(bindings)
	lines := []string{lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true).Render(title), muted.Render(subtitle), ""}
	available := max(0, height-6)
	bodyLines := strings.Split(body, "\n")
	lines = append(lines, bodyLines[:min(len(bodyLines), available)]...)
	for len(lines) < height-3 {
		lines = append(lines, "")
	}
	lines = append(lines, muted.Render(status), "", footer)
	if height < 7 {
		lines = []string{title, "Enlarge terminal", footer}
		lines = lines[:min(height, len(lines))]
	}
	return fullScreenView(width, height, strings.Join(lines, "\n"))
}

// fullScreenView gives every page the same terminal-sized canvas. Keep each
// line's own styling; repainting styled text with an outer background can
// leave gaps when an inner ANSI reset clears that background.
func fullScreenView(width, height int, content string) tea.View {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	lines := strings.Split(content, "\n")
	lines = lines[:min(len(lines), height)]
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i, line := range lines {
		line = ansi.Truncate(line, width, "…")
		lines[i] = line + strings.Repeat(" ", max(0, width-ansi.StringWidth(line)))
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}

func uiKey(key, label string) keys.Binding {
	return keys.NewBinding(keys.WithKeys(key), keys.WithHelp(key, label))
}

func selectionRow(label string, selected bool, width int) string {
	return styledListRow(bookCell(label), selected, width, currentTheme)
}

// styledListRow preserves measured column spacing; callers sanitize cell text.
func styledListRow(label string, selected bool, width int, palette Theme) string {
	style := lipgloss.NewStyle().Foreground(palette.Title)
	prefix := "  "
	if selected {
		prefix = "> "
		style = style.Background(palette.Surface).Bold(true)
	}
	return style.Width(max(1, width)).Render(ansi.Truncate(prefix+label, max(1, width), "…"))
}

func listCursor(k string, cursor, count, pageSize int) int {
	switch k {
	case "up", "k", "shift+tab":
		cursor--
	case "down", "j", "tab":
		cursor++
	case "home":
		cursor = 0
	case "end":
		cursor = count - 1
	case "pgup":
		cursor -= pageSize
	case "pgdown":
		cursor += pageSize
	}
	return min(max(0, cursor), max(0, count-1))
}
