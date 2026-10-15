package cli

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/heartleo/zlib/internal/config"
)

func TestCodexBuiltInThemesAreAvailable(t *testing.T) {
	want := strings.Fields(`catppuccin-mocha gruvbox-dark dracula tokyo nord solarized-dark monokai-extended one-half-dark two-dark catppuccin-macchiato
		base16-ocean-dark sublime-snazzy github catppuccin-latte gruvbox-light solarized-light one-half-light monokai-extended-light base16-ocean-light coldark-cold`)
	slices.Sort(want)
	got := selectableThemes()
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("selectable themes differ:\ngot  %v\nwant %v", got, want)
	}
	if len(themes) != len(want) {
		t.Fatalf("theme map has %d entries, want %d", len(themes), len(want))
	}
	for _, name := range want {
		if !isSelectableTheme(name) {
			t.Fatalf("theme %q is listed but has no palette", name)
		}
	}
}

func TestThemeCommandSavesCodexTheme(t *testing.T) {
	isolateConfigHome(t)
	t.Setenv(themeEnvVar, "")
	previous := currentTheme
	t.Cleanup(func() { currentTheme = previous })
	if err := themeCmd.RunE(themeCmd, []string{"solarized-dark"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig()
	if err != nil || cfg.Theme != "solarized-dark" || configuredThemeName() != "solarized-dark" {
		t.Fatalf("saved theme = %q, load error = %v", cfg.Theme, err)
	}
}

func TestCodexThemePreviewSurfaceSeparatesFromBackground(t *testing.T) {
	for _, name := range []string{"dracula", "github"} {
		palette := codexThemes[name]
		if palette.Background == nil || reflect.DeepEqual(palette.Background, palette.Surface) {
			t.Fatalf("%s preview surface does not separate from its background", name)
		}
	}
}

func TestIsSelectableTheme(t *testing.T) {
	if isSelectableTheme("auto") {
		t.Errorf("auto must not be selectable")
	}
	if !isSelectableTheme("catppuccin-mocha") || isSelectableTheme("mocha") {
		t.Errorf("only the canonical Catppuccin Mocha name should be selectable")
	}
	if isSelectableTheme("nope") {
		t.Errorf("unknown theme should not be selectable")
	}
}

// isolateConfigHome points config-file lookups (os.UserHomeDir) at a temp dir
// so tests don't read or clobber the real ~/.config/zlib, and returns that dir.
// It covers both the Unix ($HOME) and Windows (%USERPROFILE%) home-resolution
// paths.
func isolateConfigHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

func TestConfiguredThemeNamePrefersEnvOverConfig(t *testing.T) {
	isolateConfigHome(t)
	t.Setenv(themeEnvVar, "dracula")
	if err := config.SaveConfig(config.Config{Theme: "catppuccin-latte"}); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if got := configuredThemeName(); got != "dracula" {
		t.Errorf("configuredThemeName() = %q, want env theme dracula", got)
	}
}

func TestConfiguredThemeNameDefaultsToCatppuccinMocha(t *testing.T) {
	isolateConfigHome(t)
	t.Setenv(themeEnvVar, "")

	if got := configuredThemeName(); got != defaultTheme {
		t.Errorf("configuredThemeName() = %q, want %s", got, defaultTheme)
	}
}

func TestThemeLongHelpFormatsAvailableOnOwnLine(t *testing.T) {
	if !strings.Contains(themeCmd.Long, "\n\nAvailable: ") {
		t.Errorf("theme help should put Available on its own line: %q", themeCmd.Long)
	}
	if !strings.HasSuffix(themeCmd.Long, "\n") {
		t.Errorf("theme help should end with newline: %q", themeCmd.Long)
	}
}

func TestLegacyAutoThemeIsUnset(t *testing.T) {
	isolateConfigHome(t)
	t.Setenv(themeEnvVar, "auto")
	for _, saved := range []string{"auto", "latte", "mocha", "gruvbox"} {
		if err := config.SaveConfig(config.Config{Theme: saved}); err != nil {
			t.Fatal(err)
		}
		want := saved
		switch saved {
		case "auto", "mocha":
			want = "catppuccin-mocha"
		case "latte":
			want = "catppuccin-latte"
		case "gruvbox":
			want = "gruvbox-dark"
		}
		if got := configuredThemeName(); got != want {
			t.Fatalf("saved %q: got %q, want %q", saved, got, want)
		}
		if themeOverride() != "" {
			t.Fatal("legacy auto must not override fixed theme")
		}
	}
	if err := themeCmd.RunE(themeCmd, []string{"auto"}); err == nil {
		t.Fatal("theme auto must be rejected")
	}
}
