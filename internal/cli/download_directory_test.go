package cli

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func directoryKey(m downloadDirectoryModel, code rune) downloadDirectoryModel {
	next, _ := m.Update(tea.KeyPressMsg{Code: code})
	return next.(downloadDirectoryModel)
}

func TestDownloadDirectorySkipAndCancel(t *testing.T) {
	for _, key := range []rune{tea.KeyEnter, tea.KeyEscape} {
		m := directoryKey(newDownloadDirectoryModel("."), key)
		if !m.accepted || m.dir != "." {
			t.Fatalf("skip changed destination: %+v", m)
		}
	}
	m := newDownloadDirectoryModel(".")
	next, _ := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if next.(downloadDirectoryModel).accepted {
		t.Fatal("cancel accepted download")
	}
}

func TestDownloadDirectoryEditAndBrowse(t *testing.T) {
	m := directoryKey(newDownloadDirectoryModel("."), tea.KeyRight)
	m = directoryKey(m, tea.KeyEnter)
	want, _ := filepath.Abs(".")
	if m.picker.page != "browse" || m.picker.directory != want {
		t.Fatal("picker did not open at current download directory")
	}
	dir := t.TempDir()
	m.picker.browse(dir)
	m = directoryKey(m, tea.KeySpace)
	if m.accepted || m.picker.page != "edit" {
		t.Fatal("browse must return to destination confirmation")
	}
	m = directoryKey(m, tea.KeyRight)
	m = directoryKey(m, tea.KeyEnter)
	if m.picker.page != "browse" || m.picker.directory != dir {
		t.Fatal("picker did not reopen at selected directory")
	}
	m = directoryKey(m, tea.KeyEscape)
	m = directoryKey(m, tea.KeyEnter)
	if !m.accepted || m.dir != dir {
		t.Fatalf("directory not selected: %q", m.dir)
	}
}
