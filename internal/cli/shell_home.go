package cli

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	if width < 32 || height < 12 {
		return []string{ansi.Truncate(gradientText("ZLIB", true), width, "")}
	}
	return homeWordmark()
}

func homeWordmarkRows() []string {
	rows := make([]string, len(homeLetterforms[0]))
	for row := range rows {
		letters := make([]string, len(homeLetterforms))
		for letter := range homeLetterforms {
			letters[letter] = homeLetterforms[letter][row]
		}
		rows[row] = strings.Join(letters, " ")
	}
	return rows
}

// homeWordmark paints one ramp across the wordmark's columns, as Crush's
// ApplyForegroundGrad does, so every row shares a color at a given column.
func homeWordmark() []string {
	rows := homeWordmarkRows()
	ramp := lipgloss.Blend1D(ansi.StringWidth(rows[0]), currentTheme.Accent, wordmarkRampEnd(currentTheme))
	for i, row := range rows {
		rows[i] = paintRamp(row, ramp, true)
	}
	return rows
}

func gradientText(text string, bold bool) string {
	return paintRamp(text, lipgloss.Blend1D(ansi.StringWidth(text), currentTheme.Accent, wordmarkRampEnd(currentTheme)), bold)
}

// paintRamp colors each single-cell rune with the ramp entry for its column and
// leaves spaces unstyled.
func paintRamp(text string, ramp []color.Color, bold bool) string {
	var b strings.Builder
	for i, r := range []rune(text) {
		if r == ' ' || i >= len(ramp) {
			b.WriteRune(r)
			continue
		}
		b.WriteString(lipgloss.NewStyle().Foreground(ramp[i]).Bold(bold).Render(string(r)))
	}
	return b.String()
}

// Below this RGB distance a two-stop ramp reads as a flat color.
const wordmarkMinRampSpan = 48

// wordmarkRampEnd prefers Link, then Info and Success, so themes whose Link
// equals or nearly equals Accent still show a visible gradient.
func wordmarkRampEnd(t Theme) color.Color {
	best, bestSpan := t.Link, -1.0
	for _, candidate := range []color.Color{t.Link, t.Info, t.Success} {
		if candidate == nil {
			continue
		}
		span := colorDistance(t.Accent, candidate)
		if span >= wordmarkMinRampSpan {
			return candidate
		}
		if span > bestSpan {
			best, bestSpan = candidate, span
		}
	}
	return best
}

func colorDistance(a, b color.Color) float64 {
	if a == nil || b == nil {
		return 0
	}
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	dr, dg, db := float64(ar>>8)-float64(br>>8), float64(ag>>8)-float64(bg>>8), float64(ab>>8)-float64(bb>>8)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

func homeShortcutCell(shortcut homeShortcut, width int) string {
	name := "/" + shortcut.name
	styled := lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true).Render(name)
	if width < 30 {
		return ansi.Truncate(styled, width, "…")
	}
	gap := strings.Repeat(" ", max(2, 16-ansi.StringWidth(name)))
	label := lipgloss.NewStyle().Foreground(currentTheme.Muted).Render(shortcut.label)
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
		if available-len(rows) >= 3 {
			rows = append(rows, "")
		}
		if len(rows) < available {
			rows = append(rows, renderActionButton("Log in", true, false))
		}
		if len(rows) < available {
			rows = append(rows, lipgloss.NewStyle().Foreground(currentTheme.Muted).Render("Email and password"))
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
