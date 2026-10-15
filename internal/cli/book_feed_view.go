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
	if m.feed == nil || m.showBookCard {
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
	showCard := m.feed == nil || m.showBookCard
	if showCard {
		lines = append(lines, selectedBookCard(m.books[cursor], width), "")
	}
	count := m.feedRows()
	start := max(0, cursor-count+1)
	end := min(len(m.books), start+count)
	numberWidth := len(fmt.Sprint(len(m.books)))
	titleNeed, detailNeed := len("Title"), ansi.StringWidth(column)
	for _, book := range m.books {
		titleNeed = max(titleNeed, ansi.StringWidth(bookCell(book.Name)))
		detailNeed = max(detailNeed, ansi.StringWidth(bookCell(strings.Join(book.Authors, ", "))))
	}
	titleWidth, authorWidth, gapWidth := bookColumnWidths(width, numberWidth, titleNeed, detailNeed)
	gap := strings.Repeat(" ", gapWidth)
	cell := func(s string, w int) string {
		s = ansi.Truncate(bookCell(s), w, "…")
		return s + strings.Repeat(" ", max(0, w-ansi.StringWidth(s)))
	}
	row := func(number, title, author string) string {
		s := "  " + strings.Repeat(" ", max(0, numberWidth-ansi.StringWidth(number))) + number + gap + cell(title, titleWidth)
		if authorWidth > 0 {
			s += gap + cell(author, authorWidth)
		}
		return s + strings.Repeat(" ", max(0, width-ansi.StringWidth(s)))
	}
	lines = append(lines, muted.Render(row("#", "Title", column)))
	for i := start; i < end; i++ {
		book := m.books[i]
		s := row(fmt.Sprint(i+1), book.Name, strings.Join(book.Authors, ", "))
		if i == cursor {
			s = styledListRow(s[2:], true, width, currentTheme)
		} else {
			s = styledListRow(s[2:], false, width, currentTheme)
		}
		lines = append(lines, s)
	}
	for range count - (end - start) {
		lines = append(lines, "")
	}
	book := m.books[cursor]
	if !showCard {
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

// Measure the whole page, not the selected row, to keep columns stationary.
// Spend spare cells on content before truncating either column.
func bookColumnWidths(width, numberWidth, titleNeed, detailNeed int) (int, int, int) {
	gap := 3
	if 2+numberWidth+2*gap+titleNeed+detailNeed > width {
		gap = 2
	}
	available := max(1, width-2-numberWidth-2*gap)
	if titleNeed+detailNeed <= available {
		return titleNeed, detailNeed, gap
	}
	titleMin, detailMin := min(16, titleNeed), min(8, detailNeed)
	if available < titleMin+detailMin {
		return min(titleNeed, max(1, width-2-numberWidth-gap)), 0, gap
	}
	// Reserve a readable detail column, favor titles, then redistribute leftovers.
	detail := min(detailNeed, max(detailMin, available/3))
	title := min(titleNeed, available-detail)
	detail = min(detailNeed, available-title)
	return title, detail, gap
}

// Reserve the same space for every book so selection never moves the list.
func selectedBookCard(book zlib.Book, width int) string {
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
