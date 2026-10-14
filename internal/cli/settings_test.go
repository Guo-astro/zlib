package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

func TestSettingsCommandRoutesAndJSON(t *testing.T) {
	isolateConfigHome(t)
	t.Chdir(t.TempDir())
	previous := settingsEnvironment
	settingsEnvironment = map[string]string{zlib.EnvProxy: "http://reader:secret@localhost:7890"}
	t.Cleanup(func() { settingsEnvironment = previous })
	for _, tc := range []struct {
		args  []string
		count int
	}{
		{[]string{"config"}, 4}, {[]string{"config", "network"}, 2},
		{[]string{"config", "network", "domain"}, 1}, {[]string{"config", "network", "proxy"}, 1},
		{[]string{"config", "download"}, 2}, {[]string{"config", "download", "directory"}, 1},
		{[]string{"config", "download", "retries"}, 1},
	} {
		out, err := executeRootForTest(t, append(tc.args, "--json")...)
		if err != nil {
			t.Fatal(err)
		}
		var values []settingValue
		if err = json.Unmarshal([]byte(out), &values); err != nil || len(values) != tc.count {
			t.Fatalf("%v: %s, %v", tc.args, out, err)
		}
		if strings.Contains(out, "secret") || strings.Contains(out, "reader") {
			t.Fatal("credentials leaked")
		}
	}
	configCmd, _, _ := rootCmd.Find([]string{"config"})
	configCmd.PersistentFlags().Set("json", "false")
	configCmd.PersistentFlags().Lookup("json").Changed = false
}

func TestSettingsSaveValidationAndLiveReload(t *testing.T) {
	s := testSettingsStore(t)
	t.Setenv(zlib.EnvRetries, "")
	m, _ := newSettingsModel(t.Context(), s, "download")
	m.edit(zlib.EnvRetries)
	m.input.SetValue("-1")
	next, _ := m.Update(key("enter"))
	m = next.(settingsModel)
	if m.page != "edit" || !strings.Contains(m.message, "non-negative") {
		t.Fatal("invalid value saved")
	}
	m.input.SetValue("5")
	next, _ = m.Update(key("enter"))
	m = next.(settingsModel)
	if m.value(zlib.EnvRetries).Value != "5" || os.Getenv(zlib.EnvRetries) != "5" {
		t.Fatal("saved setting not active")
	}
	m.edit(zlib.EnvRetries)
	m.input.SetValue("")
	next, _ = m.Update(key("enter"))
	m = next.(settingsModel)
	if m.value(zlib.EnvRetries).Value != "3" || os.Getenv(zlib.EnvRetries) != "" {
		t.Fatal("reset failed")
	}
	if err := s.save(zlib.EnvDomain, "https://example.com"); err == nil {
		t.Fatal("untrusted domain saved")
	}
	if err := s.save(downloadDirEnvVar, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("nonexistent directory saved")
	}
}

func TestSettingsSaveFailureKeepsDraft(t *testing.T) {
	s := testSettingsStore(t)
	m, _ := newSettingsModel(t.Context(), s, "download")
	m.edit(zlib.EnvRetries)
	m.input.SetValue("8")
	if err := os.Mkdir(s.global, 0700); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(key("enter"))
	m = next.(settingsModel)
	if m.page != "edit" || m.input.Value() != "8" || !strings.Contains(m.message, "Could not save") {
		t.Fatal("failed save lost draft")
	}
}

func testSettingsStore(t *testing.T) settingsStore {
	t.Helper()
	dir := t.TempDir()
	return settingsStore{global: filepath.Join(dir, "global.env"), local: filepath.Join(dir, "local.env"), env: map[string]string{}}
}

func TestSettingsSaveRoundTripAndReset(t *testing.T) {
	s := testSettingsStore(t)
	original := "# retain comment\nUNRELATED=keep\nZLIB_PROXY=old\nZLIB_PROXY=duplicate\n"
	if err := os.WriteFile(s.global, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	proxy := "http://user:p%40ss@localhost:7890"
	if err := s.save(zlib.EnvProxy, proxy); err != nil {
		t.Fatal(err)
	}
	v, err := readSettingsEnv(s.global)
	if err != nil {
		t.Fatal(err)
	}
	if v[zlib.EnvProxy] != proxy || v["UNRELATED"] != "keep" {
		t.Fatalf("round trip failed")
	}
	data, _ := os.ReadFile(s.global)
	if !strings.Contains(string(data), "# retain comment") || strings.Count(string(data), "ZLIB_PROXY=") != 1 {
		t.Fatal("comment lost or duplicate remained")
	}
	dir := filepath.Join(t.TempDir(), "spaces $dollar")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := s.save(downloadDirEnvVar, dir); err != nil {
		t.Fatal(err)
	}
	v, err = readSettingsEnv(s.global)
	if err != nil || v[downloadDirEnvVar] != dir {
		t.Fatalf("path round trip: %q, %v", v[downloadDirEnvVar], err)
	}
	if err := s.save(zlib.EnvProxy, ""); err != nil {
		t.Fatal(err)
	}
	v, _ = readSettingsEnv(s.global)
	if _, ok := v[zlib.EnvProxy]; ok {
		t.Fatal("reset did not remove key")
	}
}

func TestSettingsSourceAndRedaction(t *testing.T) {
	s := testSettingsStore(t)
	if err := s.save(zlib.EnvRetries, "2"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(s.local, []byte("ZLIB_DOWNLOAD_RETRIES=4\n"), 0600)
	s.env[zlib.EnvRetries] = "5"
	values, err := s.values()
	if err != nil {
		t.Fatal(err)
	}
	if values[3].Value != "5" || values[3].Saved != "2" || values[3].Source != "Process environment" {
		t.Fatalf("precedence: %+v", values[3])
	}
	delete(s.env, zlib.EnvRetries)
	values, _ = s.values()
	if values[3].Source != "Working-directory .env" {
		t.Fatal("local source lost")
	}
	if strings.Contains(displaySetting(zlib.EnvProxy, "http://user:password@localhost:7890"), "password") {
		t.Fatal("proxy password exposed")
	}
}

func TestSettingsNavigationCancelAndViewport(t *testing.T) {
	for _, page := range []string{"home", "network", "download", "domain"} {
		for _, size := range [][2]int{{80, 24}, {40, 16}, {32, 12}, {20, 5}} {
			m, err := newSettingsModel(t.Context(), testSettingsStore(t), page)
			if err != nil {
				t.Fatal(err)
			}
			m.width, m.height = size[0], size[1]
			for _, line := range strings.Split(m.View().Content, "\n") {
				if ansi.StringWidth(line) > size[0] {
					t.Fatal("view too wide")
				}
			}
			if len(strings.Split(m.View().Content, "\n")) > size[1] {
				t.Fatal("view too tall")
			}
		}
	}
	s := testSettingsStore(t)
	m, _ := newSettingsModel(t.Context(), s, "download")
	m.edit(zlib.EnvRetries)
	m.input.SetValue("7")
	next, _ := m.Update(key("esc"))
	m = next.(settingsModel)
	if !m.discard {
		t.Fatal("dirty form exited without confirmation")
	}
	next, _ = m.Update(key("y"))
	m = next.(settingsModel)
	if _, err := os.Stat(s.global); !os.IsNotExist(err) {
		t.Fatal("cancel wrote settings")
	}
	if m.page != "download" {
		t.Fatal("cancel did not return to parent")
	}
}

func TestSettingsStaleProbeAndDirectorySelection(t *testing.T) {
	m, _ := newSettingsModel(t.Context(), testSettingsStore(t), "domain")
	m.generation = 2
	next, _ := m.Update(settingsProbeMsg{generation: 1, domains: []string{"stale"}})
	m = next.(settingsModel)
	if len(m.domains) != 0 {
		t.Fatal("stale probe updated page")
	}
	m.edit(downloadDirEnvVar)
	dir := t.TempDir()
	m.browse(dir)
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = next.(settingsModel)
	if m.page != "edit" || m.input.Value() != dir {
		t.Fatal("directory not returned to editor")
	}
	if _, err := os.Stat(m.store.global); !os.IsNotExist(err) {
		t.Fatal("browse saved prematurely")
	}
}

func TestSettingsEditorsFitAndButtonsSave(t *testing.T) {
	for _, key := range []string{zlib.EnvProxy, downloadDirEnvVar, zlib.EnvRetries} {
		for _, size := range [][2]int{{32, 12}, {40, 16}, {60, 20}, {80, 24}} {
			m, _ := newSettingsModel(t.Context(), testSettingsStore(t), "home")
			m.width, m.height = size[0], size[1]
			m.edit(key)
			view := m.View().Content
			if len(strings.Split(view, "\n")) > size[1] {
				t.Fatal("editor exceeds height")
			}
			for _, line := range strings.Split(view, "\n") {
				if ansi.StringWidth(line) > size[0] {
					t.Fatalf("editor exceeds width: %q", line)
				}
			}
		}
	}
	t.Setenv(zlib.EnvRetries, "")
	m, _ := newSettingsModel(t.Context(), testSettingsStore(t), "download")
	m.edit(zlib.EnvRetries)
	m.input.SetValue("2")
	for range 3 {
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		m = next.(settingsModel)
	}
	if m.editAction != 4 {
		t.Fatal("Tab did not reach Save")
	}
	next, _ := m.Update(key("enter"))
	m = next.(settingsModel)
	if m.value(zlib.EnvRetries).Saved != "2" {
		t.Fatal("Save button failed")
	}
}

func TestSettingsProbeFailureAndCancel(t *testing.T) {
	m, _ := newSettingsModel(t.Context(), testSettingsStore(t), "domain")
	cmd := m.probe("http://127.0.0.1:1", "domain")
	msg := cmd()
	next, _ := m.Update(msg)
	m = next.(settingsModel)
	if m.busy || len(m.domains) != 0 || !strings.Contains(m.View().Content, "No available domains") {
		t.Fatal("failed probe has no recovery state")
	}
	m.probe("http://127.0.0.1:1", "domain")
	old := m.generation
	m.back()
	next, _ = m.Update(settingsProbeMsg{generation: old, domains: []string{"stale"}})
	m = next.(settingsModel)
	if m.page != "network" || len(m.domains) != 0 {
		t.Fatal("cancelled probe updated page")
	}
}
