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
	uv "github.com/charmbracelet/ultraviolet"
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
	mode              string
	editor            textinput.Model
	draftName         string
	draftBase         string
	draftPalette      Theme
	slot              int
	message           string
}

func newThemePicker(current, override string) themePickerModel {
	names := selectableThemes()
	search := textinput.New()
	search.Prompt = "> "
	search.Placeholder = "Type to filter themes..."
	search.SetWidth(77)
	search.Focus()
	m := themePickerModel{allNames: names, names: slices.Clone(names), search: search, current: current, override: override,
		cursor: max(0, slices.Index(names, current)), width: 80, height: 24}
	p := m.chromePalette()
	styles := search.Styles()
	styles.Focused.Text = lipgloss.NewStyle().Foreground(p.Title).Background(p.Background)
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(p.Accent).Background(p.Background)
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(p.Muted).Background(p.Background)
	styles.Focused.Suggestion = styles.Focused.Placeholder
	styles.Blurred = styles.Focused
	styles.Cursor.Color = p.Accent
	styles.Cursor.Blink = false
	m.search.SetStyles(styles)
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
		if p, ok := namedTheme(m.current); ok {
			return p
		}
		return themes[defaultTheme]
	}
	p, _ := namedTheme(m.selected())
	return p
}

// Keep navigation readable while the preview crosses dark and light palettes.
func (m themePickerModel) chromePalette() Theme {
	if p, ok := namedTheme(recognizedTheme(m.current)); ok {
		return p
	}
	return themes[defaultTheme]
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
	if m.mode != "" {
		return m.updateEditor(msg)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.search.SetWidth(max(1, msg.Width-3))
		m.ensureVisible()
	case tea.KeyPressMsg:
		switch {
		case msg.String() == "ctrl+n":
			m.beginCustomTheme()
			return m, nil
		case msg.String() == "ctrl+e":
			m.editSelectedTheme()
			return m, nil
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
	p := m.chromePalette()
	rows := m.visibleRows()
	start := m.offset
	end := min(len(m.names), start+rows)
	lines := make([]string, 0, rows)
	if len(m.names) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).Background(p.Background).Width(width).Render("No matching themes"))
	}
	for i := start; i < end; i++ {
		label := m.names[i]
		if m.names[i] == m.current {
			label += " (current)"
		}
		style := lipgloss.NewStyle().Foreground(p.Title).Background(p.Background).Width(width)
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
			style = style.Background(p.Surface).Bold(true)
		}
		lines = append(lines, style.Render(ansi.Truncate(prefix+label, width, "…")))
	}
	for len(lines) < rows {
		lines = append(lines, lipgloss.NewStyle().Background(p.Background).Render(strings.Repeat(" ", width)))
	}
	return strings.Join(lines, "\n")
}

type themePreviewRow struct {
	text       string
	color      color.Color
	background color.Color
}

func themePreview(p Theme, width, height int, wide bool) string {
	rows := []themePreviewRow{
		{`zlib search "example"`, p.Accent, nil},
		{"> 1  Example Book", p.Title, p.Surface},
		{"  EPUB · 4 MB", p.ExtEPUB, nil},
		{"✓ Download complete", p.Success, nil},
	}
	if wide {
		rows = []themePreviewRow{
			{`zlib search "example"`, p.Accent, nil},
			{"Search: example · Page 1 / 4", p.Muted, nil},
			{"> 1  Example Book", p.Title, p.Surface},
			{"     Example Author · EPUB · 4 MB", p.ExtEPUB, nil},
			{"     Book details", p.Link, nil},
			{"Enter  Download", p.Warning, nil},
			{"Save to  ./books", p.Info, nil},
			{"✓ Download complete", p.Success, nil},
		}
	}
	lines := make([]string, 0, height)
	blank := lipgloss.NewStyle().Background(p.Background).Render(strings.Repeat(" ", width))
	for range max(0, (height-len(rows))/2) {
		lines = append(lines, blank)
	}
	for _, row := range rows {
		line := ansi.Truncate(row.text, width, "…")
		background := p.Background
		if row.background != nil {
			background = row.background
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(row.color).Background(background).Width(width).Render(line))
	}
	for len(lines) < height {
		lines = append(lines, blank)
	}
	return strings.Join(lines, "\n")
}

func (m themePickerModel) View() tea.View {
	if m.mode != "" {
		return m.editorView()
	}
	w, h := max(1, m.width), max(1, m.height)
	v := tea.NewView("")
	v.AltScreen = true
	if w < 32 || h < 17 {
		v.SetContent(fullScreenView(w, h, ansi.Truncate("Enlarge terminal · Esc cancel", w, "…")).Content)
		return v
	}
	m.ensureVisible()
	p := m.palette()
	chrome := m.chromePalette()
	base := lipgloss.NewStyle().Foreground(chrome.Title).Background(chrome.Background)
	accent := base.Foreground(chrome.Accent).Bold(true)
	muted := base.Foreground(chrome.Muted)
	detail := "Preview: " + m.selected()
	if m.selected() == "" {
		detail = "No matching themes"
	}
	m.search.SetWidth(max(1, w-3))
	body := ""
	if w >= 90 {
		listWidth := max(40, min(44, (w-3)/2))
		gutter := base.Render(strings.TrimSuffix(strings.Repeat("   \n", m.visibleRows()), "\n"))
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.listView(listWidth), gutter, themePreview(p, w-listWidth-3, m.visibleRows(), true))
	} else {
		body = m.listView(w) + "\n\n" + themePreview(p, w, 4, false)
	}
	status := fmt.Sprintf("%d/%d themes", m.cursor+1, len(m.names))
	if len(m.names) == 0 {
		status = "No matching themes"
	}
	if m.override != "" {
		status = "ZLIB_THEME=" + bookCell(m.override) + " overrides the saved preference"
	}
	if m.message != "" {
		status = m.message
	} else if w < 48 && m.override == "" {
		status = "Ctrl+N new · Ctrl+E edit"
	}
	helpModel := help.New()
	helpModel.SetWidth(w)
	helpModel.Styles.ShortKey = base.Foreground(chrome.Accent)
	helpModel.Styles.ShortDesc = muted
	helpModel.Styles.ShortSeparator = muted
	helpModel.Styles.Ellipsis = muted
	bindings := []keys.Binding{themePickerKeys.up, themePickerKeys.save, themePickerKeys.cancel, uiKey("ctrl+n", "new"), uiKey("ctrl+e", "edit")}
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
		lines[i] += base.Render(strings.Repeat(" ", max(0, w-ansi.StringWidth(lines[i]))))
	}
	v.SetContent(paintThemeCanvas(strings.Join(lines, "\n"), w, h, chrome))
	return v
}

// Paint unstyled cells with the fixed picker chrome without covering the
// candidate preview's own background or selected-row surface.
func paintThemeCanvas(content string, width, height int, chrome Theme) string {
	cells := uv.NewStyledString(fullScreenView(width, height, content).Content).Lines(ansi.GraphemeWidth)
	for y := range cells {
		for x := range cells[y] {
			if cells[y][x].Style.Bg == nil {
				cells[y][x].Style.Bg = chrome.Background
			}
			if cells[y][x].Style.Fg == nil {
				cells[y][x].Style.Fg = chrome.Title
			}
		}
	}
	return uv.Lines(cells).Render()
}
