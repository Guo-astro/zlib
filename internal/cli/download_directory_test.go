package cli

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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

func TestDownloadDirectoryDialogKeepsBookListBehindIt(t *testing.T) {
	m := newShellModel()
	list := newBookFeedSelectModel("Recommended", nil)
	m.open("list", list)
	directory := newDownloadDirectoryModel(t.TempDir())
	m.open("directory", directory)
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 40, Height: 16}} {
		m, _ = shellStep(m, size)
		view := ansi.Strip(m.View().Content)
		if (size.Width >= 56 && !strings.Contains(view, "Search:")) || !strings.Contains(view, "Save to") || !strings.Contains(view, "Choose another folder") {
			t.Fatalf("%dx%d: directory choice is not over the book list: %q", size.Width, size.Height, view)
		}
		for _, line := range strings.Split(m.View().Content, "\n") {
			if ansi.StringWidth(line) != size.Width {
				t.Fatalf("%dx%d: dialog row width %d", size.Width, size.Height, ansi.StringWidth(line))
			}
		}
	}
	m, _ = shellStep(m, key("right"))
	m, _ = shellStep(m, key("enter"))
	browse := ansi.Strip(m.View().Content)
	if m.page.(downloadDirectoryModel).picker.page != "browse" || !strings.Contains(browse, "Choose a folder") || !strings.Contains(browse, "Space use") || !strings.Contains(browse, "╰") {
		t.Fatal("folder browser did not stay in the dialog")
	}
	m, _ = shellStep(m, tea.WindowSizeMsg{Width: 40, Height: 12})
	if !strings.Contains(ansi.Strip(m.View().Content), "Choose a folder") {
		t.Fatal("compact folder browser lost its current mode")
	}
}

func TestDownloadDirectoryDialogAcceptsTypedFolder(t *testing.T) {
	dir := t.TempDir()
	m := directoryKey(newDownloadDirectoryModel("."), 'e')
	if !m.editing || !strings.Contains(ansi.Strip(m.View().Content), "Choose another folder") {
		t.Fatal("path editor did not open inside the dialog")
	}
	m.picker.input.SetValue(dir)
	m = directoryKey(m, tea.KeyEnter)
	if !m.accepted || m.dir != dir {
		t.Fatalf("typed folder was not selected: %q", m.dir)
	}
}
