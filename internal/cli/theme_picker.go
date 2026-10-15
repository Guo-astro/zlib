package cli

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	keys "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var themePickerKeys = struct {
	up, down, pageUp, pageDown, first, last, save, cancel keys.Binding
}{
	keys.NewBinding(keys.WithKeys("up"), keys.WithHelp("↑/↓", "preview")),
	keys.NewBinding(keys.WithKeys("down")),
	keys.NewBinding(keys.WithKeys("pgup"), keys.WithHelp("PgUp/PgDn", "scroll")),
	keys.NewBinding(keys.WithKeys("pgdown")),
	keys.NewBinding(keys.WithKeys("home"), keys.WithHelp("Home/End", "first/last")),
	keys.NewBinding(keys.WithKeys("end")),
	keys.NewBinding(keys.WithKeys("enter"), keys.WithHelp("enter", "save")),
	keys.NewBinding(keys.WithKeys("esc", "ctrl+c"), keys.WithHelp("esc", "cancel")),
}

// Preview has its own palette. Moving the cursor never changes global state or
// writes preferences, so cancellation and terminal errors need no rollback.
type themePickerModel struct {
	allNames          []string
	names             []string
	search            textinput.Model
	current, override string
	cursor, offset    int
	width, height     int
	accepted          bool
}

func newThemePicker(current, override string) themePickerModel {
	names := selectableThemes()
	search := textinput.New()
	search.Prompt = "> "
	search.Placeholder = "Type to filter themes..."
	search.SetWidth(72)
	search.Focus()
	m := themePickerModel{allNames: names, names: slices.Clone(names), search: search, current: current, override: override,
		cursor: max(0, slices.Index(names, current)), width: 80, height: 24}
	m.ensureVisible()
	return m
}

func (m themePickerModel) Init() tea.Cmd { return nil }
func (m themePickerModel) selected() string {
	if len(m.names) == 0 {
		return ""
	}
	return m.names[m.cursor]
}
func (m themePickerModel) palette() Theme {
	if len(m.names) == 0 {
		if p, ok := themes[m.current]; ok {
			return p
		}
		return themes[defaultTheme]
	}
	return themes[m.selected()]
}
func (m themePickerModel) visibleRows() int {
	if m.width >= 90 {
		return max(1, m.height-7)
	}
	return max(1, m.height-12)
}

func (m *themePickerModel) ensureVisible() {
	rows := m.visibleRows()
	m.offset = min(max(0, m.offset), max(0, len(m.names)-rows))
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
}

func (m *themePickerModel) filterNames() {
	selected := m.selected()
	query := strings.ToLower(strings.TrimSpace(m.search.Value()))
	m.names = m.names[:0]
	for _, name := range m.allNames {
		if strings.Contains(strings.ToLower(name), query) {
			m.names = append(m.names, name)
		}
	}
	m.cursor = max(0, slices.Index(m.names, selected))
	m.offset = 0
	m.ensureVisible()
}

func (m themePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.search.SetWidth(max(1, msg.Width-2))
		m.ensureVisible()
	case tea.KeyPressMsg:
		switch {
		case keys.Matches(msg, themePickerKeys.cancel):
			if msg.String() == "esc" && m.search.Value() != "" {
				m.search.SetValue("")
				m.filterNames()
				return m, nil
			}
			return m, tea.Quit
		case keys.Matches(msg, themePickerKeys.save):
			if len(m.names) == 0 {
				return m, nil
			}
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
		default:
			before := m.search.Value()
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			if m.search.Value() != before {
				m.filterNames()
			}
			return m, cmd
		}
		m.cursor = min(max(0, m.cursor), max(0, len(m.names)-1))
		m.ensureVisible()
	default:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m themePickerModel) listView(width int) string {
	p := m.palette()
	rows := m.visibleRows()
	start := m.offset
	end := min(len(m.names), start+rows)
	lines := make([]string, 0, rows)
	if len(m.names) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).Render("No matching themes"))
	}
	for i := start; i < end; i++ {
		label := m.names[i]
		if m.names[i] == m.current {
			label += " (current)"
		}
		lines = append(lines, styledListRow(label, i == m.cursor, width, p))
	}
	for len(lines) < rows {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

type themePreviewRow struct {
	text  string
	color color.Color
}

func themePreview(p Theme, width, height int, wide bool) string {
	rows := []themePreviewRow{
		{`zlib search "example"`, p.Accent},
		{"> 1  Example Book", p.Title},
		{"  EPUB · 4 MB", p.ExtEPUB},
		{"✓ Download complete", p.Success},
	}
	if wide {
		rows = []themePreviewRow{
			{`zlib search "example"`, p.Accent},
			{"Search: example · Page 1 / 4", p.Muted},
			{"> 1  Example Book", p.Title},
			{"     Example Author · EPUB · 4 MB", p.ExtEPUB},
			{"     Book details", p.Link},
			{"Enter  Download", p.Warning},
			{"Save to  ./books", p.Info},
			{"✓ Download complete", p.Success},
		}
	}
	lines := make([]string, 0, height)
	blank := lipgloss.NewStyle().Background(p.Background).Render(strings.Repeat(" ", width))
	for range max(0, (height-len(rows))/2) {
		lines = append(lines, blank)
	}
	for _, row := range rows {
		line := ansi.Truncate(row.text, width, "…")
		lines = append(lines, lipgloss.NewStyle().Foreground(row.color).Background(p.Background).Width(width).Render(line))
	}
	for len(lines) < height {
		lines = append(lines, blank)
	}
	return strings.Join(lines, "\n")
}

func (m themePickerModel) View() tea.View {
	w, h := max(1, m.width), max(1, m.height)
	v := tea.NewView("")
	v.AltScreen = true
	if w < 32 || h < 17 {
		v.SetContent(fullScreenView(w, h, ansi.Truncate("Enlarge terminal · Esc cancel", w, "…")).Content)
		return v
	}
	m.ensureVisible()
	p := m.palette()
	accent := lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(p.Muted)
	detail := "Preview: " + m.selected()
	if m.selected() == "" {
		detail = "No matching themes"
	}
	m.search.SetWidth(max(1, w-2))
	body := ""
	if w >= 90 {
		listWidth := max(40, min(44, (w-3)/2))
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.listView(listWidth), "   ", themePreview(p, w-listWidth-3, m.visibleRows(), true))
	} else {
		body = m.listView(w) + "\n\n" + themePreview(p, w, 4, false)
	}
	status := fmt.Sprintf("%d/%d themes · Enter saves", m.cursor+1, len(m.names))
	if len(m.names) == 0 {
		status = "No matching themes"
	}
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
	content := accent.Render("Theme") + "\n" + muted.Render(ansi.Truncate(detail, w, "…")) + "\n" + m.search.View() + "\n" + body
	lines := strings.Split(content, "\n")
	for len(lines) < h-3 {
		lines = append(lines, "")
	}
	lines = append(lines, muted.Render(ansi.Truncate(status, w, "…")), "", helpModel.ShortHelpView(bindings))
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], w, "…")
	}
	v.SetContent(fullScreenView(w, h, strings.Join(lines, "\n")).Content)
	return v
}
