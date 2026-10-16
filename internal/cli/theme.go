package cli

import (
	"fmt"
	"image/color"
	"os"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/heartleo/zlib/internal/config"
	"github.com/spf13/cobra"
)

// Theme defines the color palette for the CLI.
type Theme struct {
	Background color.Color // preview background when a theme defines one
	Accent     color.Color // borders, headers, titles, spinners
	Link       color.Color // ID columns, clickable elements
	Title      color.Color // book titles, primary content
	Success    color.Color // ✓ checkmarks
	Error      color.Color // ✗ errors, cancellation
	Warning    color.Color // remaining ≤ 3
	Info       color.Color // ▸ hints, default format color
	Muted      color.Color // secondary text, labels
	Surface    color.Color // progress bar empty, subtle backgrounds
	ExtEPUB    color.Color
	ExtPDF     color.Color
	ExtMOBI    color.Color
	ExtFB2     color.Color
}

const defaultTheme = "catppuccin-mocha"

// The picker groups dark themes before light themes.
var themeOrder = []string{
	"catppuccin-mocha", "gruvbox-dark", "dracula", "tokyo", "nord",
	"solarized-dark", "monokai-extended", "one-half-dark", "two-dark",
	"catppuccin-macchiato", "base16-ocean-dark", "sublime-snazzy",
	"github", "catppuccin-latte", "gruvbox-light", "solarized-light",
	"one-half-light", "monokai-extended-light", "base16-ocean-light", "coldark-cold",
}

// Built-in themes
var themes = func() map[string]Theme {
	all := map[string]Theme{
		"tokyo": {
			Background: lipgloss.Color("#1a1b26"),
			Accent:     lipgloss.Color("#7aa2f7"), // Blue    #7aa2f7
			Link:       lipgloss.Color("#7dcfff"), // Cyan    #7dcfff
			Title:      lipgloss.Color("#a9b1d6"), // Fg      #a9b1d6
			Success:    lipgloss.Color("#9ece6a"), // Green   #9ece6a
			Error:      lipgloss.Color("#f7768e"), // Red     #f7768e
			Warning:    lipgloss.Color("#e0af68"), // Yellow  #e0af68
			Info:       lipgloss.Color("#73daca"), // Teal    #73daca
			Muted:      lipgloss.Color("#565f89"), // Comment #565f89
			Surface:    lipgloss.Color("#24283b"), // Surface #24283b
			ExtEPUB:    lipgloss.Color("#9ece6a"),
			ExtPDF:     lipgloss.Color("#7dcfff"),
			ExtMOBI:    lipgloss.Color("#e0af68"),
			ExtFB2:     lipgloss.Color("#7aa2f7"),
		},
	}
	for name, palette := range codexThemes {
		all[name] = palette
	}
	return all
}()

// currentTheme is the active theme, set during init via resolveTheme().
var currentTheme = themes[defaultTheme]

const themeEnvVar = "ZLIB_THEME"

// configuredThemeName resolves valid preferences in priority order.
func configuredThemeName() string {
	if env := recognizedTheme(os.Getenv(themeEnvVar)); env != "" {
		return env
	}
	if cfg, err := config.LoadConfig(); err == nil {
		if saved := recognizedTheme(cfg.Theme); saved != "" {
			return saved
		}
	}
	return defaultTheme
}

// Old saved preferences remain usable without adding aliases to the picker.
func recognizedTheme(name string) string {
	switch name {
	case "mocha":
		return "catppuccin-mocha"
	case "latte":
		return "catppuccin-latte"
	case "gruvbox":
		return "gruvbox-dark"
	}
	if isSelectableTheme(name) {
		return name
	}
	return ""
}

func resolveTheme() {
	currentTheme, _ = namedTheme(configuredThemeName())
}

// themeNames returns the picker order.
func themeNames() []string {
	return append(slices.Clone(themeOrder), customThemeNames()...)
}

func selectableThemes() []string { return themeNames() }

// isSelectableTheme reports whether name is a settable theme value.
func isSelectableTheme(name string) bool {
	_, ok := namedTheme(name)
	return ok
}

var themeCmd = &cobra.Command{
	Use:   "theme [name]",
	Short: "Choose or set a theme",
	Long:  fmt.Sprintf("Choose a theme from the list or set one by name. Without a terminal, show the current theme.\n\nAvailable: %s\n", strings.Join(selectableThemes(), ", ")),
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) == 0 {
			current := configuredThemeName()
			if !isInteractive() {
				lipgloss.Fprintf(cmd.OutOrStdout(), "Current theme: %s\n", colorBold(current))
				lipgloss.Fprintf(cmd.OutOrStdout(), "Available: %s\n", strings.Join(selectableThemes(), ", "))
				return nil
			}
			picker := newThemePicker(current, themeOverride())
			result, err := tea.NewProgram(picker).Run()
			if err != nil {
				return err
			}
			selected := result.(themePickerModel)
			if !selected.accepted {
				return nil
			}
			name = selected.selected()
		} else {
			name = strings.ToLower(args[0])
		}

		if !isSelectableTheme(name) {
			return fmt.Errorf("unknown theme %q, available: %s", name, strings.Join(selectableThemes(), ", "))
		}

		cfg, _ := config.LoadConfig()
		cfg.Theme = name
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save theme: %w", err)
		}

		resolveTheme()
		msg := colorBold(name)
		lipgloss.Fprintf(cmd.OutOrStdout(), "%s Theme set to %s\n", colorGreen(symbolSuccess), msg)
		lipgloss.Fprintf(cmd.OutOrStdout(), "%s Saved to: %s\n", colorFaint(symbolInfo), tildePath(config.ConfigPath()))
		if override := themeOverride(); override != "" && override != name {
			lipgloss.Fprintf(cmd.OutOrStdout(), "ZLIB_THEME=%s overrides this saved preference.\n", override)
		}
		return nil
	},
}

func themeOverride() string {
	return recognizedTheme(os.Getenv(themeEnvVar))
}
