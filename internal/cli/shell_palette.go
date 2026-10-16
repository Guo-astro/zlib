package cli

import (
	"strings"

	"charm.land/bubbles/v2/help"
	keys "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m shellModel) paletteOpen() bool {
	return m.typing && strings.HasPrefix(m.input.Value(), "/") && !strings.Contains(m.input.Value(), " ")
}

func (m shellModel) commandDialogView() tea.View {
	width, height := max(1, m.width), max(1, m.height)
	items := m.matches()
	selected := min(max(0, m.selection), max(0, len(items)-1))
	if width < 32 || height < 12 {
		row := "No matching commands"
		if len(items) > 0 {
			row = "> /" + items[selected].name
		}
		return fullScreenView(width, height, "Commands\n"+m.input.View()+"\n"+row+"\nEsc close")
	}

	background := m
	background.typing = false
	background.input.SetValue("/")
	base := background.View().Content
	panelWidth := min(70, width-2)
	inner := panelWidth - 4
	input := m.input
	input.SetWidth(max(1, inner-ansi.StringWidth(input.Prompt)-1))
	twoLine := width >= 56 && height >= 18
	rows := m.paletteVisibleRows(height)
	start := min(max(0, m.paletteOffset), max(0, len(items)-rows))
	if selected < start {
		start = selected
	} else if selected >= start+rows {
		start = selected - rows + 1
	}
	end := min(len(items), start+rows)
	lines := []string{
		lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true).Render("Commands"),
		input.View(),
		"",
	}
	if len(items) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(currentTheme.Muted).Render("  No matching commands"))
	}
	for i := start; i < end; i++ {
		name := "/" + items[i].name
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		style := lipgloss.NewStyle().Foreground(currentTheme.Title).Background(currentTheme.Surface).Width(inner)
		if i == selected {
			style = style.Foreground(currentTheme.Surface).Background(currentTheme.Accent).Bold(true)
		}
		if twoLine {
			lines = append(lines, style.Render(ansi.Truncate(prefix+name, inner, "…")))
			detail := lipgloss.NewStyle().Foreground(currentTheme.Muted).Background(currentTheme.Surface).Width(inner)
			if i == selected {
				detail = detail.Foreground(currentTheme.Surface).Background(currentTheme.Accent)
			}
			lines = append(lines, detail.Render(ansi.Truncate("  "+items[i].description, inner, "…")))
		} else {
			lines = append(lines, style.Render(ansi.Truncate(prefix+name, inner, "…")))
		}
	}
	listHeight := rows
	if twoLine {
		listHeight *= 2
	}
	for len(lines) < listHeight+3 {
		lines = append(lines, "")
	}
	h := help.New()
	h.SetWidth(inner)
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(currentTheme.Muted)
	bindings := []keys.Binding{uiKey("↑/↓", "choose"), uiKey("Tab", "complete"), uiKey("Enter", "run"), uiKey("Esc", "close")}
	if inner < 48 {
		bindings = []keys.Binding{uiKey("↑/↓", "choose"), uiKey("Enter", "run"), uiKey("Esc", "close")}
	}
	lines = append(lines, "", h.ShortHelpView(bindings))
	panel := lipgloss.NewStyle().Background(currentTheme.Surface).Width(inner).Padding(1, 1).
		Border(lipgloss.RoundedBorder()).BorderForeground(currentTheme.Accent).Render(strings.Join(lines, "\n"))
	x := max(0, (width-lipgloss.Width(panel))/2)
	y := max(0, (height-lipgloss.Height(panel))/2)
	dim := lipgloss.NewStyle().Foreground(currentTheme.Muted).Render(ansi.Strip(base))
	content := lipgloss.NewCompositor(lipgloss.NewLayer(dim), lipgloss.NewLayer(panel).X(x).Y(y)).Render()
	return fullScreenView(width, height, content)
}
