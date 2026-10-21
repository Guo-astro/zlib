package cli

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

// Selected syntax palettes adapted to zlib's semantic UI roles.
var codexThemes = map[string]Theme{
	"base16-ocean-dark":      codexPalette("#c0c5ce", "#b48ead", "#8fa1b3", "#a3be8c", "#65737e", "#d08770", "#2b303b"),
	"base16-ocean-light":     codexPalette("#4f5b66", "#b48ead", "#d08770", "#a3be8c", "#a7adba", "#d08770", "#eff1f5"),
	"catppuccin-latte":       codexPalette("#4c4f69", "#8839ef", "#1e66f5", "#40a02b", "#7c7f93", "#fe640b", "#eff1f5"),
	"catppuccin-macchiato":   codexPalette("#cad3f5", "#c6a0f6", "#8aadf4", "#a6da95", "#939ab7", "#f5a97f", "#24273a"),
	"catppuccin-mocha":       codexPalette("#cdd6f4", "#cba6f7", "#89b4fa", "#a6e3a1", "#9399b2", "#fab387", "#1e1e2e"),
	"coldark-cold":           codexPalette("#111b27", "#a04900", "#7c00aa", "#116b00", "#3c526d", "#755f00", "#e3eaf2"),
	"dracula":                codexPalette("#f8f8f2", "#8be9fd", "#50fa7b", "#f1fa8c", "#6272a4", "#bd93f9", "#282a36"),
	"github":                 codexPalette("#333333", "#a71d5d", "#795da3", "#183691", "#969896", "#0086b3", "#ffffff"),
	"gruvbox-dark":           codexPalette("#fbf1c7", "#8ec07c", "#b8bb26", "#b8bb26", "#928374", "#d3869b", "#282828"),
	"gruvbox-light":          codexPalette("#282828", "#427b58", "#79740e", "#79740e", "#928374", "#8f3f71", "#fbf1c7"),
	"monokai-extended":       codexPalette("#f8f8f2", "#66d9ef", "#a6e22e", "#e6db74", "#75715e", "#be84ff", "#222222"),
	"monokai-extended-light": codexPalette("#49483e", "#0089b3", "#679c00", "#998f2f", "#75715e", "#684d99", "#fafafa"),
	"nord":                   codexPalette("#d8dee9", "#81a1c1", "#88c0d0", "#a3be8c", "#616e88", "#b48ead", "#2e3440"),
	"one-half-dark":          codexPalette("#dcdfe4", "#c678dd", "#61afef", "#98c379", "#5c6370", "#e5c07b", "#282c34"),
	"one-half-light":         codexPalette("#383a42", "#a626a4", "#0184bc", "#50a14f", "#a0a1a7", "#c18401", "#fafafa"),
	"solarized-dark":         codexPalette("#839496", "#268bd2", "#b58900", "#2aa198", "#586e75", "#6c71c4", "#002b36"),
	"solarized-light":        codexPalette("#657b83", "#268bd2", "#b58900", "#2aa198", "#93a1a1", "#6c71c4", "#fdf6e3"),
	"sublime-snazzy":         codexPalette("#f8f8f2", "#9aedfe", "#9aedfe", "#f3f99d", "#686868", "#f1f1f0", "#282a36"),
	"two-dark":               codexPalette("#abb2bf", "#c678dd", "#61afef", "#98c379", "#5c6370", "#d19a66", "#282c34"),
}

func codexPalette(fg, keyword, function, str, comment, number, background string) Theme {
	palette := Theme{
		Accent:  lipgloss.Color(keyword),
		Link:    lipgloss.Color(function),
		Title:   lipgloss.Color(fg),
		Success: lipgloss.Color("#a6e3a1"),
		Error:   lipgloss.Color("#f38ba8"),
		Info:    lipgloss.Color(function),
		Muted:   lipgloss.Color(comment),
	}
	// Status colors are fixed rather than taken from the palette: a palette's
	// "number" color made Warning blue in github.
	warnings := []string{"#f9e2af", "#fab387"}
	if background == "" {
		palette.Surface = lipgloss.Color("8")
	} else {
		palette.Background = lipgloss.Color(background)
		var r, g, b uint8
		if _, err := fmt.Sscanf(background, "#%02x%02x%02x", &r, &g, &b); err != nil {
			palette.Surface = palette.Background
		} else if int(r)*299+int(g)*587+int(b)*114 > 160000 {
			palette.Success = lipgloss.Color("#247a36")
			palette.Error = lipgloss.Color("#c43440")
			warnings = []string{"#8a5a00", "#b03500"}
			palette.Surface = lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r-r/10, g-g/10, b-b/10))
		} else {
			palette.Surface = lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r+(255-r)/8, g+(255-g)/8, b+(255-b)/8))
		}
	}
	palette = readableTheme(palette)

	// Pick distinct colors only after readableTheme has darkened or lightened
	// them: on a light background pastel syntax colors converge toward black.
	readable := func(hexes ...string) []color.Color {
		colors := make([]color.Color, len(hexes))
		for i, hex := range hexes {
			colors[i] = lipgloss.Color(hex)
			if palette.Background != nil {
				colors[i] = readableOn(colors[i], palette.Background, minTextContrast)
			}
		}
		return colors
	}
	// Warning must not read as a link or an error; amber first, then orange.
	palette.Warning = distinctFrom(readable(warnings...), palette.Info, palette.Link, palette.Error)
	// Several palettes reuse one color for two token kinds (gruvbox strings
	// and functions), so formats fall back to the status colors.
	ext := distinctColors(4, append(readable(str, function, number, keyword),
		palette.Warning, palette.Success, palette.Error, palette.Title)...)
	palette.ExtEPUB, palette.ExtPDF, palette.ExtMOBI, palette.ExtFB2 = ext[0], ext[1], ext[2], ext[3]
	return palette
}
