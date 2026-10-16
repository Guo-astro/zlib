package cli

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	keys "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

type searchSort struct {
	label string
	value zlib.OrderOption
}

var searchSorts = []searchSort{
	{"Default", ""}, {"Popular", zlib.OrderPopular}, {"Best match", zlib.OrderBestMatch},
	{"Recently added", zlib.OrderDate}, {"Title A–Z", zlib.OrderTitleAZ},
	{"Title Z–A", zlib.OrderTitleZA}, {"Year", zlib.OrderYear},
	{"Largest file", zlib.OrderFileSizeDesc}, {"Smallest file", zlib.OrderFileSizeAsc},
}

var searchLanguages = []zlib.Language{
	zlib.LangEnglish, zlib.LangChinese, zlib.LangRussian, zlib.LangFrench,
	zlib.LangGerman, zlib.LangSpanish, zlib.LangJapanese, zlib.LangKorean,
	zlib.LangItalian, zlib.LangPortuguese, zlib.LangArabic, zlib.LangDutch,
	zlib.LangPolish, zlib.LangTurkish, zlib.LangHindi,
}

var searchFormats = []zlib.Extension{
	zlib.ExtEPUB, zlib.ExtPDF, zlib.ExtAZW, zlib.ExtAZW3, zlib.ExtMOBI,
	zlib.ExtFB2, zlib.ExtDJVU, zlib.ExtDJV, zlib.ExtRTF, zlib.ExtTXT, zlib.ExtLIT,
}

const searchFilterFields = 8

// searchFilterForm shows every filter in a compact form. Long choice lists open
// a picker while keeping the other field values in the model.
type searchFilterForm struct {
	width, height                        int
	cursor, picker, choice, choiceOffset int
	input                                textinput.Model
	query, yearFrom, yearTo              string
	languages                            []zlib.Language
	extensions                           []zlib.Extension
	exact                                bool
	order                                zlib.OrderOption
	message                              string
	accepted                             bool
	editing                              bool
}

func newSearchFilterForm(query string, opts *zlib.SearchOptions) searchFilterForm {
	m := searchFilterForm{width: 80, height: 24, picker: -1, query: query}
	m.input = textinput.New()
	m.input.Prompt = "> "
	m.input.Placeholder = "Title, author, or ISBN"
	m.input.KeyMap.Paste.SetKeys("ctrl+v", "ctrl+shift+v", "shift+insert")
	m.input.SetWidth(70)
	if opts != nil {
		m.exact, m.order = opts.Exact, opts.Order
		m.languages = slices.Clone(opts.Languages)
		m.extensions = slices.Clone(opts.Extensions)
		if opts.FromYear > 0 {
			m.yearFrom = strconv.Itoa(opts.FromYear)
		}
		if opts.ToYear > 0 {
			m.yearTo = strconv.Itoa(opts.ToYear)
		}
	}
	m.startEdit()
	return m
}

func (m searchFilterForm) Init() tea.Cmd { return m.input.Focus() }

func (m *searchFilterForm) startEdit() tea.Cmd {
	m.editing = true
	m.input.Prompt = []string{"> ", "Year from > ", "Year to > "}[m.cursor]
	m.input.Placeholder = "Any year"
	if m.cursor == 0 {
		m.input.Placeholder = "Title, author, or ISBN"
	}
	m.input.SetWidth(max(1, m.width-ansi.StringWidth(m.input.Prompt)-2))
	m.input.SetValue([]string{m.query, m.yearFrom, m.yearTo}[m.cursor])
	m.input.CursorEnd()
	return m.input.Focus()
}

func (m *searchFilterForm) finishEdit() {
	if !m.editing {
		return
	}
	switch m.cursor {
	case 0:
		m.query = m.input.Value()
	case 1:
		m.yearFrom = m.input.Value()
	case 2:
		m.yearTo = m.input.Value()
	}
	m.editing = false
	m.input.Blur()
}

func (m searchFilterForm) options() (string, []string) {
	switch m.picker {
	case 3:
		out := make([]string, len(searchLanguages))
		for i, v := range searchLanguages {
			out[i] = strings.ToUpper(string(v[:1])) + string(v[1:])
		}
		return "Languages", out
	case 4:
		out := make([]string, len(searchFormats))
		for i, v := range searchFormats {
			out[i] = v.String()
		}
		return "Formats", out
	case 6:
		out := make([]string, len(searchSorts))
		for i, v := range searchSorts {
			out[i] = v.label
		}
		return "Sort by", out
	}
	return "", nil
}

func (m searchFilterForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		m.input.SetWidth(max(1, size.Width-ansi.StringWidth(m.input.Prompt)-2))
		return m, nil
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if m.picker >= 0 {
			_, options := m.options()
			switch k.String() {
			case "esc", "ctrl+c", "tab":
				m.picker = -1
			case "up", "k":
				m.choice = (m.choice - 1 + len(options)) % len(options)
			case "down", "j":
				m.choice = (m.choice + 1) % len(options)
			case "space", "enter":
				switch m.picker {
				case 3:
					value := searchLanguages[m.choice]
					if i := slices.Index(m.languages, value); i >= 0 {
						m.languages = slices.Delete(m.languages, i, i+1)
					} else {
						m.languages = append(m.languages, value)
					}
				case 4:
					value := searchFormats[m.choice]
					if i := slices.Index(m.extensions, value); i >= 0 {
						m.extensions = slices.Delete(m.extensions, i, i+1)
					} else {
						m.extensions = append(m.extensions, value)
					}
				case 6:
					m.order = searchSorts[m.choice].value
					m.picker = -1
				}
			}
			return m, nil
		}
		switch k.String() {
		case "esc", "ctrl+c":
			return m, tea.Quit
		case "ctrl+s":
			m.finishEdit()
			return m, m.submit()
		case "tab", "shift+tab":
			m.finishEdit()
			if k.String() == "tab" {
				m.cursor = (m.cursor + 1) % searchFilterFields
			} else {
				m.cursor = (m.cursor - 1 + searchFilterFields) % searchFilterFields
			}
			if m.cursor <= 2 {
				return m, m.startEdit()
			}
			return m, nil
		case "enter":
			if m.editing {
				m.finishEdit()
				m.cursor = (m.cursor + 1) % searchFilterFields
				if m.cursor <= 2 {
					return m, m.startEdit()
				}
				return m, nil
			}
			switch m.cursor {
			case 0, 1, 2:
				return m, m.startEdit()
			case 3, 4, 6:
				m.picker, m.choice, m.choiceOffset = m.cursor, 0, 0
			case 5:
				m.exact = !m.exact
			case 7:
				return m, m.submit()
			}
			return m, nil
		case "space":
			if !m.editing && m.cursor == 5 {
				m.exact = !m.exact
				return m, nil
			}
		case "up", "down":
			if !m.editing {
				if k.String() == "up" {
					m.cursor = (m.cursor - 1 + searchFilterFields) % searchFilterFields
				} else {
					m.cursor = (m.cursor + 1) % searchFilterFields
				}
				return m, nil
			}
		}
	}
	if m.editing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *searchFilterForm) submit() tea.Cmd {
	if strings.TrimSpace(m.query) == "" {
		m.message = "Enter a title, author, or ISBN"
		m.cursor = 0
		m.startEdit()
		return nil
	}
	from, err := parseSearchYear(m.yearFrom)
	if err != nil {
		m.message = "Year from must be a four-digit year"
		m.cursor = 1
		m.startEdit()
		return nil
	}
	to, err := parseSearchYear(m.yearTo)
	if err != nil {
		m.message = "Year to must be a four-digit year"
		m.cursor = 2
		m.startEdit()
		return nil
	}
	if from > 0 && to > 0 && from > to {
		m.message = "Year from must be before year to"
		m.cursor = 1
		m.startEdit()
		return nil
	}
	m.message = ""
	m.accepted = true
	return tea.Quit
}

func parseSearchYear(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	year, err := strconv.Atoi(value)
	if err != nil || len(value) != 4 || year < 1000 || year > 9999 {
		return 0, errors.New("invalid year")
	}
	return year, nil
}

func (m searchFilterForm) searchOptions() *zlib.SearchOptions {
	from, _ := parseSearchYear(m.yearFrom)
	to, _ := parseSearchYear(m.yearTo)
	return &zlib.SearchOptions{FromYear: from, ToYear: to, Languages: slices.Clone(m.languages), Extensions: slices.Clone(m.extensions), Exact: m.exact, Order: m.order}
}

func (m searchFilterForm) View() tea.View {
	if m.picker >= 0 {
		return m.pickerView()
	}
	selected := func(index int) bool { return m.cursor == index }
	value := func(index int, text, empty string) string {
		if selected(index) && m.editing {
			return m.input.View()
		}
		if text == "" {
			text = empty
		}
		return selectionRow(text, selected(index), m.width)
	}
	langs := make([]string, len(m.languages))
	for i, v := range m.languages {
		langs[i] = string(v)
	}
	fmts := make([]string, len(m.extensions))
	for i, v := range m.extensions {
		fmts[i] = v.String()
	}
	order := "Default"
	for _, item := range searchSorts {
		if item.value == m.order {
			order = item.label
			break
		}
	}
	exact := "[ ] Exact match"
	if m.exact {
		exact = "[x] Exact match"
	}
	body := []string{
		"Query", value(0, m.query, "Title, author, or ISBN"),
		value(1, "Year from                 "+displaySearchValue(m.yearFrom), ""),
		value(2, "Year to                   "+displaySearchValue(m.yearTo), ""),
		value(3, "Languages                 "+choiceSummary(langs), ""),
		value(4, "Formats                   "+choiceSummary(fmts), ""),
		value(5, exact, ""),
		value(6, "Sort by                   "+order, ""), "",
	}
	body = append(body, renderActionButton("Search", selected(7), false))
	visible := max(1, m.height-6)
	if len(body) > visible {
		focusRow := []int{1, 2, 3, 4, 5, 6, 7, 9}[m.cursor]
		start := min(max(0, focusRow-visible+1), len(body)-visible)
		body = body[start : start+visible]
	}
	return screenView(m.width, m.height, "Search books", "Set filters, then search", strings.Join(body, "\n"), m.message,
		uiKey("ctrl+s", "search"), uiKey("esc", "back"), uiKey("tab", "next"), uiKey("enter", "edit / select"))
}

func choiceSummary(values []string) string {
	if len(values) == 0 {
		return "All"
	}
	return strings.Join(values, ", ")
}

func displaySearchValue(value string) string {
	if value == "" {
		return "Any"
	}
	return value
}

func (m searchFilterForm) pickerView() tea.View {
	title, options := m.options()
	if m.width >= 34 && m.height >= 13 {
		return m.pickerOverlayView(title, options)
	}
	visible := max(1, min(len(options), m.height-7))
	start := min(max(0, m.choiceOffset), max(0, len(options)-visible))
	if m.choice < start {
		start = m.choice
	} else if m.choice >= start+visible {
		start = m.choice - visible + 1
	}
	rows := make([]string, 0, visible)
	for i := start; i < min(len(options), start+visible); i++ {
		mark := "[ ] "
		if m.picker == 3 && slices.Contains(m.languages, searchLanguages[i]) || m.picker == 4 && slices.Contains(m.extensions, searchFormats[i]) || m.picker == 6 && m.order == searchSorts[i].value {
			mark = "[x] "
		}
		rows = append(rows, selectionRow(mark+options[i], i == m.choice, m.width))
	}
	bindings := []keys.Binding{uiKey("↑/↓", "choose"), uiKey("space", "toggle"), uiKey("enter", "select"), uiKey("esc", "done")}
	if m.picker == 6 {
		bindings = []keys.Binding{uiKey("↑/↓", "choose"), uiKey("enter", "select"), uiKey("esc", "back")}
	}
	return screenView(m.width, m.height, title, "", strings.Join(rows, "\n"), "", bindings...)
}

func (m searchFilterForm) pickerOverlayView(title string, options []string) tea.View {
	inner := min(58, m.width-2) - 4
	visible := max(1, min(len(options), min(17, m.height-2)-8))
	start := min(max(0, m.choice-visible+1), max(0, len(options)-visible))
	rows := []string{lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true).Render(title), ""}
	for i := start; i < min(len(options), start+visible); i++ {
		mark := "[ ] "
		if m.picker == 3 && slices.Contains(m.languages, searchLanguages[i]) ||
			m.picker == 4 && slices.Contains(m.extensions, searchFormats[i]) ||
			m.picker == 6 && m.order == searchSorts[i].value {
			mark = "[x] "
		}
		style := lipgloss.NewStyle().Foreground(currentTheme.Title).Background(currentTheme.Surface).Width(inner)
		if i == m.choice {
			style = style.Foreground(currentTheme.Surface).Background(currentTheme.Accent).Bold(true)
		}
		rows = append(rows, style.Render(ansi.Truncate(mark+options[i], inner, "…")))
	}
	for len(rows) < visible+2 {
		rows = append(rows, "")
	}
	h := help.New()
	h.SetWidth(inner)
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(currentTheme.Muted)
	bindings := []keys.Binding{uiKey("↑/↓", "choose"), uiKey("space", "toggle"), uiKey("esc", "done")}
	if m.picker == 6 {
		bindings = []keys.Binding{uiKey("↑/↓", "choose"), uiKey("enter", "select"), uiKey("esc", "back")}
	}
	rows = append(rows, "", h.ShortHelpView(bindings))
	panel := lipgloss.NewStyle().Background(currentTheme.Surface).Width(inner).Padding(1, 1).
		Border(lipgloss.RoundedBorder()).BorderForeground(currentTheme.Accent).Render(strings.Join(rows, "\n"))
	background := m
	background.picker = -1
	base := lipgloss.NewStyle().Foreground(currentTheme.Muted).Render(ansi.Strip(background.View().Content))
	x := max(0, (m.width-lipgloss.Width(panel))/2)
	y := max(0, (m.height-lipgloss.Height(panel))/2)
	return fullScreenView(m.width, m.height, lipgloss.NewCompositor(lipgloss.NewLayer(base), lipgloss.NewLayer(panel).X(x).Y(y)).Render())
}

func runSearchFilterForm(m searchFilterForm) (searchFilterForm, error) {
	if !isInteractive() {
		return m, fmt.Errorf("search requires a terminal")
	}
	result, err := tea.NewProgram(m).Run()
	if err != nil {
		return m, err
	}
	return result.(searchFilterForm), nil
}
