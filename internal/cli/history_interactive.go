package cli

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

// — interactive history bubbletea model —

type historyState int

const (
	historyStateLoading historyState = iota
	historyStateResults
	historyStateConfirmDelete
	historyStateDeleting
)

type historySelectModel struct {
	client          *zlib.Client
	items           []zlib.DownloadHistoryItem
	page            int
	totalPages      int
	hasNext         bool
	cursor          int
	width, height   int
	state           historyState
	spinner         spinner.Model
	selected        *zlib.DownloadHistoryItem // set when user presses Enter
	err             error
	retryPage       int
	quitting        bool
	deleteError     error
	deleteFocused   bool
	confirmDownload bool
	downloadFocused bool
}

type historyDeletedMsg struct{ err error }

type historyResultMsg struct {
	items      []zlib.DownloadHistoryItem
	page       int
	totalPages int
	hasNext    bool
}

func newHistorySelectModel(c *zlib.Client) historySelectModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	return historySelectModel{
		client:  c,
		page:    1,
		state:   historyStateLoading,
		spinner: s,
	}
}

func (m historySelectModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchPage(1))
}

func (m historySelectModel) fetchPage(page int) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.DownloadHistory(page)
		if err != nil {
			return listLoadError{err, page}
		}
		// Fetch missing extension/size details
		var missingIDs []string
		for _, item := range result.Items {
			if id := historyCommandID(item); id != "" && item.Extension == "" {
				missingIDs = append(missingIDs, id)
			}
		}
		if len(missingIDs) > 0 {
			details := m.client.FetchBookDetails(missingIDs)
			for i, item := range result.Items {
				id := historyCommandID(item)
				if d, ok := details[id]; ok {
					if result.Items[i].Extension == "" {
						result.Items[i].Extension = d.Extension
					}
					if result.Items[i].Size == "" {
						result.Items[i].Size = d.Size
					}
				}
				_ = item
			}
		}
		return historyResultMsg{
			items:      result.Items,
			page:       result.Page,
			totalPages: result.TotalPages,
			hasNext:    historyHasNext(result),
		}
	}
}

func (m historySelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case historyDeletedMsg:
		m.state = historyStateResults
		m.deleteError = msg.err
		if msg.err != nil {
			return m, nil
		}
		page := m.page
		if len(m.items) == 1 && page > 1 {
			page--
		}
		m.state = historyStateLoading
		return m, tea.Batch(m.spinner.Tick, m.fetchPage(page))

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case listLoadError:
		m.err = msg.err
		m.retryPage = msg.page
		m.state = historyStateResults
		return m, nil

	case historyResultMsg:
		m.err = nil
		m.items = msg.items
		m.page = msg.page
		m.totalPages = msg.totalPages
		m.hasNext = msg.hasNext
		m.cursor = 0
		m.state = historyStateResults
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
					item := m.items[m.cursor]
					m.selected = &item
					return m, tea.Quit
				}
			}
			return m, nil
		}
		switch m.state {
		case historyStateConfirmDelete:
			if msg.String() == "left" || msg.String() == "right" || msg.String() == "tab" || msg.String() == "shift+tab" {
				m.deleteFocused = !m.deleteFocused
				return m, nil
			}
			if msg.String() == "y" || (msg.String() == "enter" && m.deleteFocused) {
				id := m.items[m.cursor].ID
				m.state = historyStateDeleting
				return m, func() tea.Msg { return historyDeletedMsg{m.client.DeleteDownloadHistory(id)} }
			}
			if msg.String() == "n" || msg.String() == "esc" || msg.String() == "ctrl+c" || msg.String() == "enter" {
				m.state = historyStateResults
			}
			return m, nil
		case historyStateDeleting:
			return m, nil
		case historyStateResults:
			switch msg.String() {
			case "r":
				page := m.page
				if m.err != nil {
					page = max(1, m.retryPage)
				}
				m.err = nil
				m.state = historyStateLoading
				return m, tea.Batch(m.spinner.Tick, m.fetchPage(page))
			case "d", "delete":
				if len(m.items) > 0 && m.items[m.cursor].ID != "" {
					m.deleteError = nil
					m.deleteFocused = false
					m.state = historyStateConfirmDelete
				}
			case "pgdown":
				m.cursor = min(max(0, len(m.items)-1), m.cursor+(searchSelectModel{height: m.height}).feedRows())
			case "pgup":
				m.cursor = max(0, m.cursor-(searchSelectModel{height: m.height}).feedRows())
			case "home":
				m.cursor = 0
			case "end":
				m.cursor = max(0, len(m.items)-1)
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.items)-1 {
					m.cursor++
				}
			case "left", "h":
				if m.page > 1 {
					m.state = historyStateLoading
					return m, tea.Batch(m.spinner.Tick, m.fetchPage(m.page-1))
				}
			case "right", "l":
				if m.hasNext {
					m.state = historyStateLoading
					return m, tea.Batch(m.spinner.Tick, m.fetchPage(m.page+1))
				}
			case "enter":
				if len(m.items) > 0 && m.cursor < len(m.items) {
					item := m.items[m.cursor]
					if item.DownloadURL != "" || historyCommandID(item) != "" {
						m.confirmDownload = true
						m.downloadFocused = false
						return m, nil
					}
				}
			case "q", "esc", "ctrl+c":
				m.err = nil
				m.quitting = true
				return m, tea.Quit
			}
		case historyStateLoading:
			if msg.String() == "q" || msg.String() == "esc" || msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		if m.state == historyStateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func historyHasNext(result zlib.DownloadHistoryResult) bool {
	if result.TotalPages > 0 {
		return result.Page < result.TotalPages
	}
	return len(result.Items) > 0
}

func (m historySelectModel) View() tea.View {
	if m.err != nil {
		return screenView(m.width, m.height, "Download history", "Could not load history", bookCell(m.err.Error()), "", uiKey("esc", "back"), uiKey("r", "retry"))
	}
	list := searchSelectModel{feedTitle: "Download history", width: m.width, height: m.height, cursor: m.cursor, spinner: m.spinner, state: searchStateResults}
	if m.state == historyStateLoading {
		list.state = searchStateLoading
	}
	column := "Format / size"
	for _, item := range m.items {
		if item.Date != "" {
			column = "Downloaded"
			break
		}
	}
	for _, item := range m.items {
		detail := item.Date
		if column == "Format / size" {
			detail = strings.TrimSpace(strings.ToUpper(item.Extension) + " " + item.Size)
		}
		list.books = append(list.books, zlib.Book{ID: item.ID, Hash: item.Hash, URL: item.URL, Name: item.Name, Authors: []string{detail}, Extension: item.Extension, Size: item.Size})
	}
	status := fmt.Sprintf("Page %d", m.page)
	if m.totalPages > 0 {
		status += fmt.Sprintf(" / %d", m.totalPages)
	}
	v := tea.NewView(list.feedViewWith(column, "No history on this page.", status, "←/→ page · "))
	v.AltScreen = true
	lines := strings.Split(v.Content, "\n")
	width := m.width
	if width == 0 {
		width = 80
	}
	help := "↑↓ select · ←→ page · enter download · d delete · q quit"
	if width < 70 {
		help = "↑↓ ←→ · enter · d delete · q"
	}
	if m.state == historyStateConfirmDelete {
		help = "Delete selected history? y / n"
	} else if m.state == historyStateDeleting {
		help = "Deleting history…"
	} else if m.deleteError != nil {
		help = "Delete failed; d retry, q quit"
		if len(lines) > 1 {
			lines[1] = bookCell(m.deleteError.Error())
		}
	}
	if len(lines) > 1 {
		lines[len(lines)-1] = help
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], max(1, width), "…")
	}
	v.Content = strings.Join(lines, "\n")
	if m.state == historyStateConfirmDelete {
		v.Content = m.deleteDialog(v.Content)
	}
	if m.confirmDownload {
		book := m.items[m.cursor]
		v.Content = confirmationDialog(v.Content, m.width, m.height, "Download book?", book.Name, historyCommandID(book), "Download the selected book.", "Download", m.downloadFocused, false)
	}
	return v
}

func (m historySelectModel) deleteDialog(background string) string {
	book := m.items[m.cursor]
	return confirmationDialog(background, m.width, m.height, "Delete history?", book.Name, book.ID, "Local files are kept.", "Delete", m.deleteFocused, true)
}
