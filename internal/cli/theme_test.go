package cli

import (
	"strings"
	"testing"

	"github.com/heartleo/zlib/internal/config"
)

func TestIsSelectableTheme(t *testing.T) {
	if isSelectableTheme("auto") {
		t.Errorf("auto must not be selectable")
	}
	if !isSelectableTheme("mocha") {
		t.Errorf("mocha should be selectable")
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
	if err := config.SaveConfig(config.Config{Theme: "latte"}); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if got := configuredThemeName(); got != "dracula" {
		t.Errorf("configuredThemeName() = %q, want env theme dracula", got)
	}
}

func TestConfiguredThemeNameDefaultsToMocha(t *testing.T) {
	isolateConfigHome(t)
	t.Setenv(themeEnvVar, "")

	if got := configuredThemeName(); got != "mocha" {
		t.Errorf("configuredThemeName() = %q, want mocha", got)
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
	for _, saved := range []string{"auto", "latte"} {
		if err := config.SaveConfig(config.Config{Theme: saved}); err != nil {
			t.Fatal(err)
		}
		want := saved
		if saved == "auto" {
			want = "mocha"
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
