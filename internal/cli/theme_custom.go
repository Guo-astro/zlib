package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/heartleo/zlib/internal/config"
)

type customThemeFile struct {
	Base   string            `json:"base"`
	Colors map[string]string `json:"colors"`
}

var themeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)
var themeColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func customThemeDir() string { return filepath.Join(filepath.Dir(config.ConfigPath()), "themes") }

func customThemePath(name string) string { return filepath.Join(customThemeDir(), name+".json") }

func customThemeNames() []string {
	entries, err := os.ReadDir(customThemeDir())
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".json")
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && themeNamePattern.MatchString(name) && !slices.Contains(themeOrder, name) {
			if _, err := loadCustomTheme(name); err == nil {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return names
}

func validateCustomThemeName(name string) error {
	if !themeNamePattern.MatchString(name) {
		return errors.New("use 1–40 lowercase letters, digits, or hyphens; start with a letter")
	}
	if slices.Contains(themeOrder, name) {
		return fmt.Errorf("%s is a built-in theme", name)
	}
	if _, err := os.Stat(customThemePath(name)); err == nil {
		return fmt.Errorf("%s already exists", name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func themeColor(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func themeColors(p Theme) map[string]string {
	return map[string]string{
		"background": themeColor(p.Background), "accent": themeColor(p.Accent),
		"link": themeColor(p.Link), "title": themeColor(p.Title),
		"success": themeColor(p.Success), "error": themeColor(p.Error),
		"warning": themeColor(p.Warning), "info": themeColor(p.Info),
		"muted": themeColor(p.Muted), "surface": themeColor(p.Surface),
		"epub": themeColor(p.ExtEPUB), "pdf": themeColor(p.ExtPDF),
		"mobi": themeColor(p.ExtMOBI), "fb2": themeColor(p.ExtFB2),
	}
}

var themeColorSlots = []string{"background", "surface", "title", "accent", "muted", "link", "success", "error", "warning", "info", "epub", "pdf", "mobi", "fb2"}

func paletteFromColors(base Theme, colors map[string]string) (Theme, error) {
	values := themeColors(base)
	for name, value := range colors {
		if _, ok := values[name]; !ok {
			return Theme{}, fmt.Errorf("unknown color %q", name)
		}
		if !themeColorPattern.MatchString(value) {
			return Theme{}, fmt.Errorf("%s must be a #RRGGBB color", name)
		}
		values[name] = strings.ToLower(value)
	}
	c := func(key string) color.Color { return lipgloss.Color(values[key]) }
	return Theme{Background: c("background"), Surface: c("surface"), Title: c("title"), Accent: c("accent"), Muted: c("muted"), Link: c("link"), Success: c("success"), Error: c("error"), Warning: c("warning"), Info: c("info"), ExtEPUB: c("epub"), ExtPDF: c("pdf"), ExtMOBI: c("mobi"), ExtFB2: c("fb2")}, nil
}

func loadCustomTheme(name string) (Theme, error) {
	if !themeNamePattern.MatchString(name) || slices.Contains(themeOrder, name) {
		return Theme{}, fmt.Errorf("invalid custom theme %q", name)
	}
	file, err := readCustomThemeFile(name)
	if err != nil {
		return Theme{}, err
	}
	base, ok := themes[file.Base]
	if !ok || !slices.Contains(themeOrder, file.Base) {
		return Theme{}, fmt.Errorf("unknown base theme %q", file.Base)
	}
	return paletteFromColors(base, file.Colors)
}

func readCustomThemeFile(name string) (customThemeFile, error) {
	if !themeNamePattern.MatchString(name) || slices.Contains(themeOrder, name) {
		return customThemeFile{}, fmt.Errorf("invalid custom theme %q", name)
	}
	data, err := os.ReadFile(customThemePath(name))
	if err != nil {
		return customThemeFile{}, err
	}
	var file customThemeFile
	if err := json.Unmarshal(data, &file); err != nil {
		return customThemeFile{}, err
	}
	return file, nil
}

func saveCustomTheme(name, base string, palette Theme) error {
	if !themeNamePattern.MatchString(name) || slices.Contains(themeOrder, name) {
		return fmt.Errorf("invalid custom theme name %q", name)
	}
	if _, ok := themes[base]; !ok || !slices.Contains(themeOrder, base) {
		return fmt.Errorf("unknown base theme %q", base)
	}
	if _, err := paletteFromColors(themes[base], themeColors(palette)); err != nil {
		return err
	}
	if err := os.MkdirAll(customThemeDir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(customThemeFile{Base: base, Colors: themeColors(palette)}, "", "  ")
	if err != nil {
		return err
	}
	path := customThemePath(name)
	if _, err := os.Stat(path); err == nil {
		return os.WriteFile(path, data, 0o600)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temp, err := os.CreateTemp(customThemeDir(), ".theme-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func namedTheme(name string) (Theme, bool) {
	if palette, ok := themes[name]; ok {
		return palette, true
	}
	palette, err := loadCustomTheme(name)
	return palette, err == nil
}
