package cli

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

// Fixed letterforms keep the wordmark steady while the terminal resizes.
var homeLetterforms = [4][5]string{
	{"█████", "   ██", "  ██ ", " ██  ", "█████"},
	{"█    ", "█    ", "█    ", "█    ", "█████"},
	{"█████", "  █  ", "  █  ", "  █  ", "█████"},
	{"████ ", "█   █", "████ ", "█   █", "████ "},
}

type homeShortcut struct{ name, label string }

var signedInShortcuts = []homeShortcut{
	{"search", "Search books"}, {"favorites", "Saved books"},
	{"recommended", "Recommended"}, {"history", "Download history"},
	{"config", "Settings"}, {"kindle", "Send to Kindle"},
}

func homeLogo(width, height int) []string {
	accent := lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true)
	version := lipgloss.NewStyle().Foreground(currentTheme.Muted)
	if width < 32 || height < 12 {
		if width < 4 || height < 2 {
			return []string{ansi.Truncate(accent.Render("ZLIB"), width, "")}
		}
		label := ansi.Truncate(zlib.Version, max(1, width-2), "…")
		blockWidth := max(4, ansi.StringWidth(label))
		name := accent.Render("ZLIB") + strings.Repeat(" ", blockWidth-4)
		return []string{name, strings.Repeat(" ", blockWidth-ansi.StringWidth(label)) + version.Render(label)}
	}
	rows := make([]string, len(homeLetterforms[0]))
	colors := []lipgloss.Style{
		lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true),
		lipgloss.NewStyle().Foreground(currentTheme.Link).Bold(true),
		lipgloss.NewStyle().Foreground(currentTheme.Title).Bold(true),
		lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true),
	}
	for row := range rows {
		letters := make([]string, len(homeLetterforms))
		for letter := range homeLetterforms {
			letters[letter] = colors[letter].Render(homeLetterforms[letter][row])
		}
		rows[row] = strings.Join(letters, " ")
	}
	wordmarkWidth := ansi.StringWidth(rows[0])
	label := ansi.Truncate(zlib.Version, wordmarkWidth, "…")
	meta := strings.Repeat(" ", wordmarkWidth-ansi.StringWidth(label)) + version.Render(label)
	if height >= 16 {
		rows = append(rows, strings.Repeat(" ", wordmarkWidth))
	}
	return append(rows, meta)
}

func homeShortcutCell(shortcut homeShortcut, width int) string {
	name := "/" + shortcut.name
	styled := lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true).Render(name)
	if width < 30 {
		return ansi.Truncate(styled, width, "…")
	}
	gap := strings.Repeat(" ", max(2, 16-ansi.StringWidth(name)))
	label := lipgloss.NewStyle().Foreground(currentTheme.Title).Render(shortcut.label)
	return ansi.Truncate(styled+gap+label, width, "…")
}

func (m shellModel) homeShortcutRows() []string {
	shortcuts := signedInShortcuts
	if m.width >= 76 {
		rows := make([]string, 0, len(shortcuts)/2)
		for i := 0; i < len(shortcuts); i += 2 {
			left := homeShortcutCell(shortcuts[i], 33)
			left += strings.Repeat(" ", max(0, 33-ansi.StringWidth(left)))
			right := homeShortcutCell(shortcuts[i+1], 33)
			right += strings.Repeat(" ", max(0, 33-ansi.StringWidth(right)))
			rows = append(rows, left+"    "+right)
		}
		return rows
	}
	rows := make([]string, 0, len(shortcuts))
	contentWidth := 0
	for _, shortcut := range shortcuts {
		contentWidth = max(contentWidth, 16+ansi.StringWidth(shortcut.label))
	}
	rowWidth := min(contentWidth, max(1, m.width-4))
	for _, shortcut := range shortcuts {
		row := homeShortcutCell(shortcut, rowWidth)
		rows = append(rows, row+strings.Repeat(" ", max(0, rowWidth-ansi.StringWidth(row))))
	}
	return rows
}

func (m shellModel) homeView(available int) string {
	if available == 0 {
		return ""
	}
	place := func(line string) string {
		padding := max(0, (m.width-ansi.StringWidth(line))/2)
		return strings.Repeat(" ", padding) + line
	}
	rows := homeLogo(m.width, available)
	if !m.signedIn {
		if available-len(rows) >= 3 {
			rows = append(rows, "", lipgloss.NewStyle().Foreground(currentTheme.Title).Bold(true).Render("Sign in to Z-Library"))
		}
		if len(rows) < available {
			rows = append(rows, lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true).Render("› Log in"))
		}
		if len(rows) < available {
			rows = append(rows, lipgloss.NewStyle().Foreground(currentTheme.Muted).Render("Email and password"))
		}
		if available-len(rows) >= 2 {
			rows = append(rows, "", lipgloss.NewStyle().Foreground(currentTheme.Muted).Render("Search and saved books require sign-in"))
		}
	} else if !strings.HasPrefix(m.input.Value(), "/") && available-len(rows) >= 4 {
		shortcuts := m.homeShortcutRows()
		gap := 1
		if available >= 16 {
			gap = 2
		}
		room := max(0, available-len(rows)-gap-1)
		if room > 0 {
			for range gap {
				rows = append(rows, "")
			}
			rows = append(rows, shortcuts[:min(len(shortcuts), room)]...)
		}
	}
	if len(rows) > available {
		rows = rows[:available]
	}
	top := max(0, (available-len(rows))/2)
	lines := make([]string, 0, top+len(rows))
	for range top {
		lines = append(lines, "")
	}
	for _, row := range rows {
		lines = append(lines, place(row))
	}
	return strings.Join(lines, "\n")
}
