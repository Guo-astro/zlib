package cli

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zlib "github.com/heartleo/zlib"
)

// — interactive search bubbletea model —

type searchState int

const (
	searchStateInput   searchState = iota // waiting for query input (handled outside model)
	searchStateLoading                    // fetching a page
	searchStateResults                    // showing results
	searchStateFilter                     // editing the query inline (/)
)

type searchSelectModel struct {
	client          *zlib.Client
	query           string
	feedTitle       string
	showBookCard    bool
	feed            func() ([]zlib.Book, error)
	feedPage        func(int) (zlib.SearchResult, error)
	opts            *zlib.SearchOptions
	books           []zlib.Book
	page            int
	totalPages      int
	cursor          int
	state           searchState
	spinner         spinner.Model
	selectedID      string // set when user presses Enter
	confirmDownload bool
	downloadFocused bool
	err             error
	retryPage       int
	quitting        bool
	fullTitle       bool // do not truncate titles
	width           int  // terminal width, from tea.WindowSizeMsg
	height          int
	input           textinput.Model // inline query editor, shown in searchStateFilter
}

func newBookFeedSelectModel(title string, load func() ([]zlib.Book, error)) searchSelectModel {
	m := newSearchSelectModel("", nil, nil, false)
	m.feedTitle = title
	m.feed = load
	return m
}

func newSavedBooksSelectModel(load func(int) (zlib.SearchResult, error)) searchSelectModel {
	m := newSearchSelectModel("", nil, nil, false)
	m.feedTitle = "Favorites"
	m.feedPage = load
	return m
}

// messages
type searchResultMsg struct {
	books      []zlib.Book
	page       int
	totalPages int
}

type listLoadError struct {
	err  error
	page int
}

func newSearchSelectModel(query string, c *zlib.Client, opts *zlib.SearchOptions, fullTitle bool) searchSelectModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(currentTheme.Accent)

	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "refine search…"
	styles := ti.Styles()
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	styles.Focused.Text = lipgloss.NewStyle().Foreground(currentTheme.Title)
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(currentTheme.Muted)
	styles.Blurred = styles.Focused
	styles.Cursor.Color = currentTheme.Accent
	ti.SetStyles(styles)

	return searchSelectModel{
		client:    c,
		query:     query,
		opts:      opts,
		page:      1,
		state:     searchStateLoading,
		spinner:   s,
		fullTitle: fullTitle,
		input:     ti,
	}
}

func (m searchSelectModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchPage(1))
}

func (m searchSelectModel) fetchPage(page int) tea.Cmd {
	return func() tea.Msg {
		if m.feedPage != nil {
			result, err := m.feedPage(page)
			if err != nil {
				return listLoadError{err, page}
			}
			return searchResultMsg{books: result.Books, page: result.Page, totalPages: result.TotalPages}
		}
		if m.feed != nil {
			books, err := m.feed()
			if err != nil {
				return listLoadError{err, page}
			}
			return searchResultMsg{books: books, page: 1, totalPages: 1}
		}
		result, err := m.client.Search(m.query, page, 25, m.opts)
		if err != nil {
			return listLoadError{err, page}
		}
		return searchResultMsg{
			books:      result.Books,
			page:       result.Page,
			totalPages: result.TotalPages,
		}
	}
}

func (m searchSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(max(1, msg.Width-3))
		return m, nil

	case listLoadError:
		m.err = msg.err
		m.retryPage = msg.page
		m.state = searchStateResults
		return m, nil

	case searchResultMsg:
		m.err = nil
		m.books = msg.books
		m.page = msg.page
		m.totalPages = msg.totalPages
		m.cursor = 0
		m.state = searchStateResults
		return m, nil

	case tea.KeyPressMsg:
		if m.err != nil && msg.String() != "r" && msg.String() != "esc" && msg.String() != "q" && msg.String() != "ctrl+c" {
			return m, nil
		}
		if m.confirmDownload {
			switch msg.String() {
			case "left", "right", "tab", "shift+tab":
				m.downloadFocused = !m.downloadFocused
			case "esc", "ctrl+c", "n":
				m.confirmDownload = false
			case "enter":
				m.confirmDownload = false
				if m.downloadFocused {
					m.selectedID = bookCommandID(m.books[m.cursor])
					return m, tea.Quit
				}
			}
			return m, nil
		}
		switch m.state {
		case searchStateResults:
			switch msg.String() {
			case "r":
				page := m.page
				if m.err != nil {
					page = max(1, m.retryPage)
				}
				m.err = nil
				m.state = searchStateLoading
				return m, tea.Batch(m.spinner.Tick, m.fetchPage(page))
			case "pgdown", "pgup", "home", "end":
				if len(m.books) > 0 {
					switch msg.String() {
					case "pgdown":
						m.cursor = min(len(m.books)-1, m.cursor+m.feedRows())
					case "pgup":
						m.cursor = max(0, m.cursor-m.feedRows())
					case "home":
						m.cursor = 0
					case "end":
						m.cursor = len(m.books) - 1
					}
				}
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.books)-1 {
					m.cursor++
				}
			case "left", "h":
				if m.feed == nil && m.page > 1 {
					m.state = searchStateLoading
					return m, tea.Batch(m.spinner.Tick, m.fetchPage(m.page-1))
				}
			case "right", "l":
				if m.feed == nil && m.page < m.totalPages {
					m.state = searchStateLoading
					return m, tea.Batch(m.spinner.Tick, m.fetchPage(m.page+1))
				}
			case "enter":
				if len(m.books) > 0 && m.cursor < len(m.books) {
					id := bookCommandID(m.books[m.cursor])
					if id != "" {
						m.confirmDownload = true
						m.downloadFocused = false
						return m, nil
					}
				}
			case "/":
				if m.feed != nil || m.feedPage != nil {
					return m, nil
				}
				// Enter inline query editing, pre-filled with the current
				// query so it can be tweaked rather than retyped.
				m.state = searchStateFilter
				m.input.SetValue(m.query)
				m.input.CursorEnd()
				return m, m.input.Focus()
			case "q", "esc", "ctrl+c":
				m.err = nil
				m.quitting = true
				return m, tea.Quit
			}
		case searchStateFilter:
			switch msg.String() {
			case "enter":
				q := m.input.Value()
				m.input.Blur()
				if strings.TrimSpace(q) == "" || q == m.query {
					// Nothing to re-run; return to the current results.
					m.state = searchStateResults
					return m, nil
				}
				m.query = q
				m.page = 1
				m.totalPages = 0
				m.cursor = 0
				m.state = searchStateLoading
				return m, tea.Batch(m.spinner.Tick, m.fetchPage(1))
			case "esc":
				m.input.Blur()
				m.state = searchStateResults
				return m, nil
			case "ctrl+c":
				m.input.Blur()
				m.quitting = true
				return m, tea.Quit
			}
			// Any other key edits the query; fall through to input.Update below.
		case searchStateLoading:
			if msg.String() == "q" || msg.String() == "esc" || msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		if m.state == searchStateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// While editing, route remaining messages (typed keys, cursor blink) to
	// the text input so it updates and keeps blinking.
	if m.state == searchStateFilter {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m searchSelectModel) View() tea.View {
	v := m.listView()
	if m.err != nil {
		title := m.feedTitle
		if title == "" {
			title = "Search: " + bookCell(m.query)
		}
		v = screenView(m.width, m.height, title, "Could not load books", bookCell(m.err.Error()), "", uiKey("esc", "close"), uiKey("r", "retry"))
	}
	if m.confirmDownload {
		book := m.books[m.cursor]
		v.Content = confirmationDialog(v.Content, m.width, m.height, "Download book?", book.Name, bookCommandID(book), "Download the selected book.", "Download", m.downloadFocused, false)
		v.AltScreen = true
	}
	return v
}

func (m searchSelectModel) listView() tea.View {
	if m.feed == nil && m.feedPage == nil {
		list := m
		list.feedTitle = "Search: " + bookCell(m.query)
		status := fmt.Sprintf("Page %d", m.page)
		if m.totalPages > 0 {
			status += fmt.Sprintf(" / %d", m.totalPages)
		}
		if m.state == searchStateFilter {
			status = m.input.View()
		}
		v := tea.NewView(list.feedViewWith("Author", "No books found.", status, "/ search · ←/→ page · "))
		v.AltScreen = true
		return v
	}
	if m.feed != nil {
		v := tea.NewView(m.feedView())
		v.AltScreen = true
		return v
	}
	status := fmt.Sprintf("Page %d", m.page)
	if m.totalPages > 0 {
		status += fmt.Sprintf(" / %d", m.totalPages)
	}
	v := tea.NewView(m.feedViewWith("Author", "No favorites on this page.", status, "←/→ page · "))
	v.AltScreen = true
	return v
}

func interactiveSearch(c *zlib.Client, opts *zlib.SearchOptions, fullTitle bool) (string, error) {
	var query string
	var selectedExts []string

	// Pre-select extensions from CLI --ext flags
	if opts != nil {
		for _, e := range opts.Extensions {
			selectedExts = append(selectedExts, string(e))
		}
	}

	fields, accepted, err := runInlineForm(newInlineForm("Search books", "Search", []formField{
		{label: "Query", hint: "Title, author, or ISBN", validate: requiredField("query")},
		{label: "Formats", options: []string{"EPUB", "AZW", "AZW3", "MOBI", "PDF", "RTF", "TXT"}, selected: selectedExts, hint: "Choose formats or leave all unchecked"},
	}))
	if err != nil {
		return "", err
	}
	if !accepted {
		return "", nil
	}
	query, selectedExts = fields[0].value, fields[1].selected
	if strings.TrimSpace(query) == "" {
		return "", nil
	}

	// Merge selected extensions into opts
	if len(selectedExts) > 0 || opts != nil {
		if opts == nil {
			opts = &zlib.SearchOptions{}
		}
		opts.Extensions = nil
		for _, e := range selectedExts {
			opts.Extensions = append(opts.Extensions, zlib.Extension(e))
		}
	}

	if c == nil {
		c = newClient()
	}
	p := tea.NewProgram(newSearchSelectModel(query, c, opts, fullTitle))
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	sm := result.(searchSelectModel)
	if sm.err != nil {
		return "", sm.err
	}
	return sm.selectedID, nil
}
