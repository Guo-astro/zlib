package cli

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
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

const searchReducedMotion = false

type searchSelectModel struct {
	client          *zlib.Client
	query           string
	feedTitle       string
	showBookCard    bool
	feed            func() ([]zlib.Book, error)
	feedPage        func(int) (zlib.SearchResult, error)
	opts            *zlib.SearchOptions
	books           []zlib.Book
	details         bookDetailState
	page            int
	loadingPage     int
	totalPages      int
	cursor          int
	listOffset      int
	listAction      string
	state           searchState
	spinner         spinner.Model
	progress        progress.Model
	revealRows      int
	revealing       bool
	cardFlash       int
	motionSeq       uint64
	reducedMotion   bool
	selectedID      string // set when user presses Enter
	confirmDownload bool
	downloadFocused bool
	err             error
	retryPage       int
	quitting        bool
	fullTitle       bool // do not truncate titles
	inShell         bool
	width           int // terminal width, from tea.WindowSizeMsg
	height          int
	input           textinput.Model // inline query editor, shown in searchStateFilter
}

func newBookFeedSelectModel(title string, load func() ([]zlib.Book, error)) searchSelectModel {
	m := newSearchSelectModel("", nil, nil, false)
	m.feedTitle = title
	m.feed = load
	m.showBookCard = true
	return m
}

func newSavedBooksSelectModel(load func(int) (zlib.SearchResult, error)) searchSelectModel {
	m := newSearchSelectModel("", nil, nil, false)
	m.feedTitle = "Favorites"
	m.feedPage = load
	return m
}

func (m searchSelectModel) withBookDetails(c *zlib.Client) searchSelectModel {
	m.client = c
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

type searchMotionMsg struct{ sequence uint64 }

func searchMotionTick(sequence uint64) tea.Cmd {
	return tea.Tick(55*time.Millisecond, func(time.Time) tea.Msg { return searchMotionMsg{sequence} })
}

func (m *searchSelectModel) startSearchMotion() tea.Cmd {
	if m.feed != nil || m.feedPage != nil || m.reducedMotion || len(m.books) == 0 {
		m.revealing = false
		m.cardFlash = 0
		return nil
	}
	m.motionSeq++
	m.cardFlash = 3
	return searchMotionTick(m.motionSeq)
}

func newSearchSelectModel(query string, c *zlib.Client, opts *zlib.SearchOptions, fullTitle bool) searchSelectModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	p := progress.New(progress.WithWidth(12), progress.WithColors(currentTheme.Accent, currentTheme.Info),
		progress.WithoutPercentage(), progress.WithFillCharacters('━', '─'))

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
		client:        c,
		query:         query,
		opts:          opts,
		page:          1,
		loadingPage:   1,
		state:         searchStateLoading,
		spinner:       s,
		progress:      p,
		reducedMotion: searchReducedMotion,
		fullTitle:     fullTitle,
		input:         ti,
	}
}

func (m searchSelectModel) Init() tea.Cmd {
	return m.loadPage(1)
}

func (m searchSelectModel) loadPage(page int) tea.Cmd {
	if m.reducedMotion {
		return m.fetchPage(page)
	}
	return tea.Batch(m.spinner.Tick, m.fetchPage(page))
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
		result, err := m.client.Search(m.query, page, defaultSearchPageSize, m.opts)
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
		m.ensureCursorVisible()
		return m, nil

	case listLoadError:
		m.motionSeq++
		m.revealing = false
		m.cardFlash = 0
		m.err = msg.err
		m.retryPage = msg.page
		m.loadingPage = 0
		m.state = searchStateResults
		return m, nil

	case searchResultMsg:
		m.err = nil
		m.books = msg.books
		m.page = msg.page
		m.totalPages = msg.totalPages
		m.cursor = 0
		m.listOffset = 0
		m.loadingPage = 0
		m.state = searchStateResults
		m.revealing = m.feed == nil && m.feedPage == nil && !m.reducedMotion && len(m.books) > 4
		m.revealRows = min(4, len(m.books))
		detail := m.queueSelectedDetail()
		if m.reducedMotion {
			return m, detail
		}
		bar := m.progress.SetPercent(1 / float64(max(1, len(m.books))))
		return m, tea.Batch(bar, m.startSearchMotion(), detail)

	case bookDetailReadyMsg:
		if msg.sequence == m.details.sequence && msg.ref == m.selectedBookRef() && m.state == searchStateResults {
			return m, m.details.fetch(m.client, msg.ref)
		}
		return m, nil

	case bookDetailResultMsg:
		m.details.store(msg)
		return m, nil

	case searchMotionMsg:
		if msg.sequence != m.motionSeq || m.state != searchStateResults {
			return m, nil
		}
		if m.revealing {
			m.revealRows = min(len(m.books), m.revealRows+6)
			m.revealing = m.revealRows < len(m.books)
		}
		m.cardFlash = max(0, m.cardFlash-1)
		if m.revealing || m.cardFlash > 0 {
			return m, searchMotionTick(m.motionSeq)
		}
		return m, nil

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.err != nil && m.state != searchStateFilter && msg.String() != "r" && msg.String() != "/" && msg.String() != "esc" && msg.String() != "q" && msg.String() != "ctrl+c" {
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
			previousCursor := m.cursor
			switch msg.String() {
			case "r":
				page := m.page
				if m.err != nil {
					page = max(1, m.retryPage)
				}
				m.err = nil
				m.state = searchStateLoading
				m.loadingPage = page
				return m, m.loadPage(page)
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
					m.loadingPage = m.page - 1
					return m, m.loadPage(m.loadingPage)
				}
			case "right", "l":
				if m.feed == nil && m.page < m.totalPages {
					m.state = searchStateLoading
					m.loadingPage = m.page + 1
					return m, m.loadPage(m.loadingPage)
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
			m.ensureCursorVisible()
			if m.cursor != previousCursor && m.feed == nil && m.feedPage == nil {
				m.revealing = false
				bar := m.progress.SetPercent(float64(m.cursor+1) / float64(len(m.books)))
				return m, tea.Batch(bar, m.startSearchMotion(), m.queueSelectedDetail())
			}
			if m.cursor != previousCursor {
				return m, m.queueSelectedDetail()
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
				m.listOffset = 0
				m.err = nil
				m.state = searchStateLoading
				m.loadingPage = 1
				m.motionSeq++
				m.revealing = false
				return m, m.loadPage(1)
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
		if m.state == searchStateFilter {
			v = screenView(m.width, m.height, title, "Search failed", m.input.View(), bookCell(m.err.Error()), uiKey("enter", "search"), uiKey("esc", "cancel"))
		} else if m.feed == nil && m.feedPage == nil {
			searchKey := "/"
			if m.inShell {
				searchKey = "ctrl+f"
			}
			v = screenView(m.width, m.height, title, fmt.Sprintf("Could not load page %d", m.retryPage), bookCell(m.err.Error()), "Retry or change the search", uiKey("r", "retry"), uiKey(searchKey, "search"), uiKey("esc", "back"))
		} else {
			v = screenView(m.width, m.height, title, "Could not load books", bookCell(m.err.Error()), "", uiKey("r", "retry"), uiKey("esc", "back"))
		}
	}
	if m.confirmDownload {
		book := m.books[m.cursor]
		v.Content = confirmationDialog(v.Content, m.width, m.height, "Download book?", book.Name, bookCommandID(book), "Download the selected book.", "Download", m.downloadFocused, false)
		v.AltScreen = true
	}
	return fullScreenView(m.width, m.height, v.Content)
}

func (m searchSelectModel) listView() tea.View {
	if m.feed == nil && m.feedPage == nil {
		list := m
		list.feedTitle = "Search: " + bookCell(m.query)
		status := fmt.Sprintf("Page %d", m.page)
		if m.state == searchStateLoading && m.loadingPage > 0 {
			status = fmt.Sprintf("Page %d", m.loadingPage)
		}
		if m.totalPages > 0 {
			status += fmt.Sprintf(" / %d", m.totalPages)
		}
		if m.state == searchStateResults && len(m.books) > 0 {
			if m.width >= 55 {
				bar := m.progress.View()
				if m.reducedMotion {
					bar = m.progress.ViewAs(float64(m.cursor+1) / float64(len(m.books)))
				}
				status += "   " + bar
			}
		}
		if m.state == searchStateFilter {
			status = m.input.View()
		}
		return fullScreenView(m.width, m.height, list.feedViewWith("Author", "No books found.", status, "/ search · ←/→ page · "))
	}
	if m.feed != nil {
		return fullScreenView(m.width, m.height, m.feedView())
	}
	status := fmt.Sprintf("Page %d", m.page)
	if m.totalPages > 0 {
		status += fmt.Sprintf(" / %d", m.totalPages)
	}
	return fullScreenView(m.width, m.height, m.feedViewWith("Author", "No favorites on this page.", status, "←/→ page · "))
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
