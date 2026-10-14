package cli

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

func bookCell(s string) string {
	return strings.Join(strings.Fields(ansi.Strip(s)), " ")
}

func (m searchSelectModel) feedRows() int {
	height := m.height
	if height == 0 {
		height = 24
	}
	if m.feed == nil && m.feedPage == nil {
		return max(1, height-13)
	}
	return max(1, height-12)
}

func (m searchSelectModel) feedView() string {
	return m.feedViewWith("Author", "No books found.", "", "")
}

func (m searchSelectModel) feedViewWith(column, empty, status, navigation string) string {
	width, height := m.width, m.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	if width < 30 || height < 14 {
		return ansi.Truncate("Enlarge terminal · q quit", max(1, width), "…")
	}
	accent := lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(currentTheme.Muted)
	title := strings.TrimPrefix(m.feedTitle, "Browse ")
	if title != "" {
		title = strings.ToUpper(title[:1]) + title[1:]
	}
	lines := []string{accent.Render(title), status}
	if m.state == searchStateLoading {
		return strings.Join(append(lines, m.spinner.View()+" Loading books…", "", "q quit"), "\n")
	}
	if len(m.books) == 0 {
		help := navigation + "q quit"
		if m.state == searchStateFilter {
			help = "enter search · esc cancel"
		}
		return strings.Join(append(lines, empty, "", ansi.Truncate(help, width, "…")), "\n")
	}
	cursor := min(max(0, m.cursor), len(m.books)-1)
	searchCard := m.feed == nil && m.feedPage == nil
	if searchCard {
		lines = append(lines, searchBookCard(m.books[cursor], width), "")
	}
	count := m.feedRows()
	start := max(0, cursor-count+1)
	end := min(len(m.books), start+count)
	// Keep titles readable on narrow terminals; authors remain in the preview.
	authorWidth := 0
	if width >= 70 {
		authorWidth = min(28, width/4)
	}
	numberWidth := len(fmt.Sprint(len(m.books)))
	titleWidth := width - numberWidth - 3
	if authorWidth > 0 {
		titleWidth -= authorWidth + 2
	}
	cell := func(s string, w int) string {
		s = ansi.Truncate(bookCell(s), w, "…")
		return s + strings.Repeat(" ", max(0, w-ansi.StringWidth(s)))
	}
	row := func(number, title, author string) string {
		s := "  " + cell(number, numberWidth) + " " + cell(title, titleWidth)
		if authorWidth > 0 {
			s += "  " + cell(author, authorWidth)
		}
		return s
	}
	lines = append(lines, muted.Render(row("#", "Title", column)))
	for i := start; i < end; i++ {
		book := m.books[i]
		s := row(fmt.Sprint(i+1), book.Name, strings.Join(book.Authors, ", "))
		if i == cursor {
			s = "> " + s[2:]
			s = lipgloss.NewStyle().Foreground(currentTheme.Title).Background(currentTheme.Surface).Bold(true).Render(s)
		}
		lines = append(lines, s)
	}
	for range count - (end - start) {
		lines = append(lines, "")
	}
	book := m.books[cursor]
	if !searchCard {
		lines = append(lines, "")
		preview := strings.Split(ansi.Wrap(bookCell(book.Name), width, ""), "\n")
		for i := range 3 {
			line := ""
			if i < len(preview) {
				line = preview[i]
				if i == 2 && len(preview) > 3 {
					line = ansi.Truncate(line+"…", width, "…")
				}
			}
			lines = append(lines, accent.Render(line))
		}
		lines = append(lines, muted.Render(ansi.Truncate(bookCell(strings.Join(book.Authors, ", ")), width, "…")))
		lines = append(lines, ansi.Truncate(feedBookMetadata(book), width, "…"), "")
	}
	help := "↑/↓ select · PgUp/PgDn scroll · Home/End · enter download · q quit"
	if width < 70 {
		help = "↑/↓ · enter download · q quit"
	}
	lines[0] = accent.Render(ansi.Truncate(title, max(1, width-20), "…")) + muted.Render(fmt.Sprintf("  %d / %d", cursor+1, len(m.books)))
	if navigation != "" {
		help = "↑/↓ select · ←/→ page · enter download · q quit"
		if width < 70 {
			help = "↑↓ · ←→ page · enter · q quit"
		}
	}
	if strings.Contains(navigation, "/ search") {
		help = "↑/↓ select · ←/→ page · / search · enter download · q quit"
		if width < 70 {
			help = "↑↓ ←→ · / search · enter · q"
		}
	}
	if m.state == searchStateFilter {
		help = "enter search · esc cancel"
	}
	lines = append(lines, muted.Render(help))
	return strings.Join(lines, "\n")
}

// Reserve the same space for every book so selection never moves the list.
func searchBookCard(book zlib.Book, width int) string {
	inner := max(1, width-4)
	accent := lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(currentTheme.Muted)
	title := strings.Split(ansi.Wrap(bookCell(book.Name), inner, ""), "\n")
	var lines []string
	for i := range 2 {
		line := ""
		if i < len(title) {
			line = title[i]
			if i == 1 && len(title) > 2 {
				line = ansi.Truncate(line, max(1, inner-1), "") + "…"
			}
		}
		lines = append(lines, accent.Render(line))
	}
	lines = append(lines, ansi.Truncate(bookCell(strings.Join(book.Authors, ", ")), inner, "…"))
	var details []string
	for _, value := range []string{strings.ToUpper(book.Extension), book.Size, book.Year, book.Language, book.Publisher} {
		if value = bookCell(value); value != "" {
			details = append(details, value)
		}
	}
	lines = append(lines, muted.Render(ansi.Truncate(strings.Join(details, " · "), inner, "…")))
	lines = append(lines, muted.Render(ansi.Truncate("ID: "+bookCommandID(book), inner, "…")))
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(currentTheme.Accent).
		Padding(0, 1).Width(width - 2).Render(strings.Join(lines, "\n"))
}

func feedBookMetadata(book zlib.Book) string {
	parts := []string{"ID: " + bookCommandID(book)}
	for _, value := range []string{strings.ToUpper(book.Extension), book.Size} {
		if value != "" {
			parts = append(parts, bookCell(value))
		}
	}
	return strings.Join(parts, " · ")
}
