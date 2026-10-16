package cli

import (
	"fmt"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
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
	searchStateFilter                     // editing search filters (/ or Ctrl+F)
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
	filter          searchFilterForm
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

func newSearchSelectModel(query string, c *zlib.Client, opts *zlib.SearchOptions, fullTitle bool) searchSelectModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	p := progress.New(progress.WithWidth(12), progress.WithColors(currentTheme.Accent, currentTheme.Info),
		progress.WithoutPercentage(), progress.WithFillCharacters('━', '─'))

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
	if m.state == searchStateFilter {
		if size, ok := msg.(tea.WindowSizeMsg); ok {
			m.width, m.height = size.Width, size.Height
		}
		if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "esc" && m.filter.picker < 0 {
			m.state = searchStateResults
			return m, nil
		}
		if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		next, _ := m.filter.Update(msg)
		m.filter = next.(searchFilterForm)
		if !m.filter.accepted {
			return m, nil
		}
		m.query = m.filter.query
		m.opts = m.filter.searchOptions()
		m.page, m.loadingPage = 1, 1
		m.totalPages, m.cursor, m.listOffset = 0, 0, 0
		m.err = nil
		m.state = searchStateLoading
		return m, m.loadPage(1)
	}
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()
		return m, nil

	case listLoadError:
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
		return m, m.queueSelectedDetail()

	case bookDetailReadyMsg:
		if msg.sequence == m.details.sequence && msg.ref == m.selectedBookRef() && m.state == searchStateResults {
			return m, m.details.fetch(m.client, msg.ref)
		}
		return m, nil

	case bookDetailResultMsg:
		m.details.store(msg)
		return m, nil

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
				m.state = searchStateFilter
				m.filter = newSearchFilterForm(m.query, m.opts)
				m.filter.width, m.filter.height = m.width, m.height
				return m, m.filter.Init()
			case "q", "esc", "ctrl+c":
				m.err = nil
				m.quitting = true
				return m, tea.Quit
			}
			m.ensureCursorVisible()
			if m.cursor != previousCursor {
				return m, m.queueSelectedDetail()
			}
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

	return m, nil
}

func (m searchSelectModel) View() tea.View {
	if m.state == searchStateFilter {
		return m.filter.View()
	}
	v := m.listView()
	if m.err != nil {
		title := m.feedTitle
		if title == "" {
			title = "Search: " + bookCell(m.query)
		}
		if m.feed == nil && m.feedPage == nil {
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
		v.Content = confirmationDialog(v.Content, m.width, m.height, "Download book?", book.Name, bookCommandID(book), "", "Download", m.downloadFocused, false)
		v.AltScreen = true
	}
	return fullScreenView(m.width, m.height, v.Content)
}

func (m searchSelectModel) listView() tea.View {
	if m.feed == nil && m.feedPage == nil {
		list := m
		list.feedTitle = "Search: " + bookCell(m.query)
		if m.state == searchStateLoading {
			return screenView(m.width, m.height, list.feedTitle, "", "  "+m.spinner.View(), "", uiKey("esc", "back"))
		}
		status := fmt.Sprintf("Page %d", m.page)
		if m.totalPages > 0 {
			status += fmt.Sprintf(" / %d", m.totalPages)
		}
		if m.state == searchStateResults && len(m.books) > 0 {
			if m.width >= 55 {
				status += "   " + m.progress.ViewAs(float64(m.cursor+1)/float64(len(m.books)))
			}
		}
		if m.state == searchStateResults && len(m.books) == 0 {
			searchKey := "/"
			exitLabel := "quit"
			if m.inShell {
				searchKey = "Ctrl+F"
				exitLabel = "back"
			}
			return screenView(m.width, m.height, list.feedTitle, status,
				"  No books found for this search.\n  Try another title, author, or ISBN.", "",
				uiKey(searchKey, "search"), uiKey("q", exitLabel))
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
	form, err := runSearchFilterForm(newSearchFilterForm("", opts))
	if err != nil {
		return "", err
	}
	if !form.accepted {
		return "", nil
	}
	query := form.query
	opts = form.searchOptions()

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
