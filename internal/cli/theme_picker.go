package cli

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	keys "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var themePickerKeys = struct {
	up, down, pageUp, pageDown, first, last, save, cancel keys.Binding
}{
	keys.NewBinding(keys.WithKeys("up", "k"), keys.WithHelp("↑/↓", "preview")),
	keys.NewBinding(keys.WithKeys("down", "j")),
	keys.NewBinding(keys.WithKeys("pgup"), keys.WithHelp("PgUp/PgDn", "scroll")),
	keys.NewBinding(keys.WithKeys("pgdown")),
	keys.NewBinding(keys.WithKeys("home"), keys.WithHelp("Home/End", "first/last")),
	keys.NewBinding(keys.WithKeys("end")),
	keys.NewBinding(keys.WithKeys("enter"), keys.WithHelp("enter", "save")),
	keys.NewBinding(keys.WithKeys("esc", "q", "ctrl+c"), keys.WithHelp("esc", "cancel")),
}

// Preview has its own palette. Moving the cursor never changes global state or
// writes preferences, so cancellation and terminal errors need no rollback.
type themePickerModel struct {
	names                 []string
	current, override     string
	cursor, width, height int
	accepted              bool
}

func newThemePicker(current, override string) themePickerModel {
	names := selectableThemes()
	return themePickerModel{names: names, current: current, override: override,
		cursor: max(0, slices.Index(names, current)), width: 80, height: 24}
}

func (m themePickerModel) Init() tea.Cmd    { return nil }
func (m themePickerModel) selected() string { return m.names[m.cursor] }
func (m themePickerModel) palette() Theme {
	return themes[m.selected()]
}
func (m themePickerModel) visibleRows() int {
	if m.width >= 90 {
		return max(1, m.height-7)
	}
	return max(1, m.height-15)
}

func (m themePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch {
		case keys.Matches(msg, themePickerKeys.cancel):
			return m, tea.Quit
		case keys.Matches(msg, themePickerKeys.save):
			m.accepted = true
			return m, tea.Quit
		case keys.Matches(msg, themePickerKeys.up):
			m.cursor--
		case keys.Matches(msg, themePickerKeys.down):
			m.cursor++
		case keys.Matches(msg, themePickerKeys.pageUp):
			m.cursor -= m.visibleRows()
		case keys.Matches(msg, themePickerKeys.pageDown):
			m.cursor += m.visibleRows()
		case keys.Matches(msg, themePickerKeys.first):
			m.cursor = 0
		case keys.Matches(msg, themePickerKeys.last):
			m.cursor = len(m.names) - 1
		}
		m.cursor = min(max(0, m.cursor), len(m.names)-1)
	}
	return m, nil
}

func (m themePickerModel) listView(width int) string {
	p := m.palette()
	rows := m.visibleRows()
	start := max(0, m.cursor-rows+1)
	end := min(len(m.names), start+rows)
	lines := make([]string, 0, rows)
	for i := start; i < end; i++ {
		label := "  " + m.names[i]
		if m.names[i] == m.current {
			label += " (current)"
		}
		style := lipgloss.NewStyle().Foreground(p.Title)
		if i == m.cursor {
			label = "> " + label[2:]
			style = style.Background(p.Surface).Bold(true)
		}
		lines = append(lines, style.Width(width).Render(ansi.Truncate(label, width, "…")))
	}
	for len(lines) < rows {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func themePreview(p Theme, width int) string {
	paint := func(c color.Color, s string) string {
		return lipgloss.NewStyle().Foreground(c).Render(ansi.Truncate(s, width, "…"))
	}
	bar := progress.New(progress.WithWidth(max(1, width-10)), progress.WithColors(p.Accent), progress.WithoutPercentage())
	bar.EmptyColor = p.Surface
	selected := lipgloss.NewStyle().Foreground(p.Title).Background(p.Surface).Bold(true).Width(width)
	return strings.Join([]string{
		paint(p.Accent, "Preview · Recommendations"),
		selected.Render(ansi.Truncate("> 1 Pride and Prejudice", width, "…")),
		paint(p.Title, "  2 The Time Machine"),
		paint(p.Muted, "Jane Austen · English · 1813"),
		paint(p.ExtEPUB, "EPUB") + "  " + paint(p.ExtPDF, "PDF") + "  " + paint(p.ExtMOBI, "MOBI") + "  " + paint(p.ExtFB2, "FB2"),
		bar.ViewAs(0.65) + paint(p.Info, " 65%"),
		paint(p.Success, "✓ Saved") + "  " + paint(p.Warning, "! Quota") + "  " + paint(p.Error, "× Error"),
	}, "\n")
}

func (m themePickerModel) View() tea.View {
	w, h := max(1, m.width), max(1, m.height)
	v := tea.NewView("")
	v.AltScreen = true
	if w < 32 || h < 17 {
		v.SetContent(ansi.Truncate("Enlarge terminal · Esc cancel", w, "…"))
		return v
	}
	p := m.palette()
	accent := lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(p.Muted)
	detail := "Preview: " + m.selected()
	body := ""
	if w >= 90 {
		listWidth := min(30, w/3)
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.listView(listWidth), "   ", themePreview(p, w-listWidth-3))
	} else {
		body = m.listView(w) + "\n\n" + themePreview(p, w)
	}
	status := fmt.Sprintf("%d/%d · Enter saves; preview is not saved", m.cursor+1, len(m.names))
	if m.override != "" {
		status = "ZLIB_THEME=" + bookCell(m.override) + " overrides the saved preference"
	}
	helpModel := help.New()
	helpModel.SetWidth(w)
	helpModel.Styles.ShortKey = lipgloss.NewStyle().Foreground(p.Accent)
	helpModel.Styles.ShortDesc = muted
	bindings := []keys.Binding{themePickerKeys.up, themePickerKeys.save, themePickerKeys.cancel}
	if w < 48 {
		bindings = []keys.Binding{themePickerKeys.save, themePickerKeys.cancel}
	}
	if w >= 90 {
		bindings = append(bindings, themePickerKeys.pageUp, themePickerKeys.first)
	}
	content := accent.Render("Theme") + "\n" + muted.Render(ansi.Truncate(detail, w, "…")) + "\n\n" + body
	lines := strings.Split(content, "\n")
	for len(lines) < h-3 {
		lines = append(lines, "")
	}
	lines = append(lines, muted.Render(ansi.Truncate(status, w, "…")), "", helpModel.ShortHelpView(bindings))
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "…")
	}
	v.SetContent(strings.Join(lines, "\n"))
	return v
}
