package cli

import (
	"fmt"
	"image/color"
	"os"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/heartleo/zlib/internal/config"
	"github.com/spf13/cobra"
)

// Theme defines the color palette for the CLI.
type Theme struct {
	Accent  color.Color // borders, headers, titles, spinners
	Link    color.Color // ID columns, clickable elements
	Title   color.Color // book titles, primary content
	Success color.Color // ✓ checkmarks
	Error   color.Color // ✗ errors, cancellation
	Warning color.Color // remaining ≤ 3
	Info    color.Color // ▸ hints, default format color
	Muted   color.Color // secondary text, labels
	Surface color.Color // progress bar empty, subtle backgrounds
	ExtEPUB color.Color
	ExtPDF  color.Color
	ExtMOBI color.Color
	ExtFB2  color.Color
}

// Built-in themes
var themes = map[string]Theme{
	"mocha": {
		Accent:  lipgloss.Color("183"), // Mauve  #cba6f7
		Link:    lipgloss.Color("111"), // Blue   #89b4fa
		Title:   lipgloss.Color("189"), // Lavender #b4befe
		Success: lipgloss.Color("114"), // Green  #a6e3a1
		Error:   lipgloss.Color("204"), // Red    #f38ba8
		Warning: lipgloss.Color("223"), // Yellow #f9e2af
		Info:    lipgloss.Color("109"), // Teal   #94e2d5
		Muted:   lipgloss.Color("243"), // Overlay0 #6c7086
		Surface: lipgloss.Color("238"), // Surface0 #313244
		ExtEPUB: lipgloss.Color("114"), // Green
		ExtPDF:  lipgloss.Color("111"), // Blue
		ExtMOBI: lipgloss.Color("223"), // Yellow
		ExtFB2:  lipgloss.Color("183"), // Mauve
	},
	"dracula": {
		Accent:  lipgloss.Color("141"), // Purple  #bd93f9
		Link:    lipgloss.Color("117"), // Cyan    #8be9fd
		Title:   lipgloss.Color("231"), // Foreground #f8f8f2
		Success: lipgloss.Color("84"),  // Green   #50fa7b
		Error:   lipgloss.Color("210"), // Red     #ff5555
		Warning: lipgloss.Color("228"), // Yellow  #f1fa8c
		Info:    lipgloss.Color("117"), // Cyan
		Muted:   lipgloss.Color("61"),  // Comment #6272a4
		Surface: lipgloss.Color("236"), // Current #44475a
		ExtEPUB: lipgloss.Color("84"),
		ExtPDF:  lipgloss.Color("117"),
		ExtMOBI: lipgloss.Color("228"),
		ExtFB2:  lipgloss.Color("141"),
	},
	"tokyo": {
		Accent:  lipgloss.Color("75"),  // Blue    #7aa2f7
		Link:    lipgloss.Color("117"), // Cyan    #7dcfff
		Title:   lipgloss.Color("189"), // Fg      #a9b1d6
		Success: lipgloss.Color("108"), // Green   #9ece6a
		Error:   lipgloss.Color("203"), // Red     #f7768e
		Warning: lipgloss.Color("223"), // Yellow  #e0af68
		Info:    lipgloss.Color("73"),  // Teal    #73daca
		Muted:   lipgloss.Color("59"),  // Comment #565f89
		Surface: lipgloss.Color("236"), // Surface #24283b
		ExtEPUB: lipgloss.Color("108"),
		ExtPDF:  lipgloss.Color("117"),
		ExtMOBI: lipgloss.Color("223"),
		ExtFB2:  lipgloss.Color("75"),
	},
	"nord": {
		Accent:  lipgloss.Color("110"), // Frost   #81a1c1
		Link:    lipgloss.Color("110"), // Frost   #81a1c1
		Title:   lipgloss.Color("253"), // Snow    #eceff4
		Success: lipgloss.Color("108"), // Green   #a3be8c
		Error:   lipgloss.Color("174"), // Red     #bf616a
		Warning: lipgloss.Color("222"), // Yellow  #ebcb8b
		Info:    lipgloss.Color("73"),  // Frost   #88c0d0
		Muted:   lipgloss.Color("60"),  // Comment #4c566a
		Surface: lipgloss.Color("236"), // Polar   #3b4252
		ExtEPUB: lipgloss.Color("108"),
		ExtPDF:  lipgloss.Color("110"),
		ExtMOBI: lipgloss.Color("222"),
		ExtFB2:  lipgloss.Color("110"),
	},
	"gruvbox": {
		Accent:  lipgloss.Color("208"), // Orange  #fe8019
		Link:    lipgloss.Color("109"), // Blue    #83a598
		Title:   lipgloss.Color("223"), // Fg      #ebdbb2
		Success: lipgloss.Color("142"), // Green   #b8bb26
		Error:   lipgloss.Color("167"), // Red     #fb4934
		Warning: lipgloss.Color("214"), // Yellow  #fabd2f
		Info:    lipgloss.Color("108"), // Aqua    #8ec07c
		Muted:   lipgloss.Color("245"), // Gray    #928374
		Surface: lipgloss.Color("237"), // Bg1     #3c3836
		ExtEPUB: lipgloss.Color("142"), // Green
		ExtPDF:  lipgloss.Color("109"), // Blue
		ExtMOBI: lipgloss.Color("214"), // Yellow
		ExtFB2:  lipgloss.Color("175"), // Purple  #d3869b
	},
	"latte": { // Catppuccin Latte — light background palette
		Accent:  lipgloss.Color("98"),  // Mauve  #8839ef
		Link:    lipgloss.Color("27"),  // Blue   #1e66f5
		Title:   lipgloss.Color("60"),  // Text   #4c4f69 (dark, for light bg)
		Success: lipgloss.Color("28"),  // Green  #40a02b
		Error:   lipgloss.Color("160"), // Red    #d20f39
		Warning: lipgloss.Color("136"), // Yellow #df8e1d
		Info:    lipgloss.Color("30"),  // Teal   #179299
		Muted:   lipgloss.Color("247"), // Overlay0 #9ca0b0
		Surface: lipgloss.Color("254"), // Surface0 #ccd0da
		ExtEPUB: lipgloss.Color("28"),  // Green
		ExtPDF:  lipgloss.Color("27"),  // Blue
		ExtMOBI: lipgloss.Color("136"), // Yellow
		ExtFB2:  lipgloss.Color("98"),  // Mauve
	},
}

// currentTheme is the active theme, set during init via resolveTheme().
var currentTheme = themes["mocha"]

const themeEnvVar = "ZLIB_THEME"

// configuredThemeName resolves valid preferences in priority order. Legacy auto
// values are treated as unset; installations without a fixed theme use mocha.
func configuredThemeName() string {
	if env := os.Getenv(themeEnvVar); isSelectableTheme(env) {
		return env
	}
	if cfg, err := config.LoadConfig(); err == nil && isSelectableTheme(cfg.Theme) {
		return cfg.Theme
	}
	return "mocha"
}

func resolveTheme() { currentTheme = themes[configuredThemeName()] }

// themeNames returns sorted list of available theme names.
func themeNames() []string {
	names := make([]string, 0, len(themes))
	for k := range themes {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func selectableThemes() []string { return themeNames() }

// isSelectableTheme reports whether name is a settable theme value.
func isSelectableTheme(name string) bool {
	_, ok := themes[name]
	return ok
}

var themeCmd = &cobra.Command{
	Use:   "theme [name]",
	Short: "Show or set color theme",
	Long:  fmt.Sprintf("Choose a theme from a list, or set it directly by name.\nWithout a terminal, show the current theme.\n\nAvailable: %s\n", strings.Join(selectableThemes(), ", ")),
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

		currentTheme = themes[configuredThemeName()]
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
	value := os.Getenv(themeEnvVar)
	if isSelectableTheme(value) {
		return value
	}
	return ""
}
