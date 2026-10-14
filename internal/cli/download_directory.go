package cli

import (
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

// This choice is local to one download; it never writes configuration.
type downloadDirectoryModel struct {
	picker                    settingsModel
	dir                       string
	editing, change, accepted bool
}

func newDownloadDirectoryModel(dir string) downloadDirectoryModel {
	input := textinput.New()
	input.SetValue(dir)
	return downloadDirectoryModel{dir: dir, picker: settingsModel{
		input: input, width: 80, height: 24, page: "edit", key: downloadDirEnvVar,
	}}
}

func (m downloadDirectoryModel) Init() tea.Cmd { return nil }

func (m downloadDirectoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.picker.width, m.picker.height = size.Width, size.Height
		m.picker.input.SetWidth(max(1, min(56, size.Width-8)))
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		k := key.String()
		if k == "ctrl+c" {
			return m, tea.Quit
		}
		if m.picker.page == "browse" {
			if k == "q" || k == "esc" {
				m.picker.page = "edit"
				m.editing, m.change = false, false
				return m, nil
			}
			if k == "space" {
				dir, err := validateSetting(downloadDirEnvVar, m.picker.directory)
				if err != nil {
					m.picker.message = err.Error()
					return m, nil
				}
				m.dir = dir
				m.picker.input.SetValue(dir)
				m.picker.page = "edit"
				m.editing, m.change = false, false
				return m, nil
			}
		} else if !m.editing {
			switch k {
			case "left", "right", "up", "down", "tab", "shift+tab":
				m.change = !m.change
			case "esc":
				m.accepted = true
				return m, tea.Quit
			case "enter":
				if !m.change {
					m.accepted = true
					return m, tea.Quit
				}
				m.picker.browse(m.dir)
				return m, nil
			}
			return m, nil
		} else {
			switch k {
			case "esc":
				m.editing = false
				m.change = false
				return m, nil
			case "ctrl+b":
				m.picker.browse(m.picker.input.Value())
				return m, nil
			case "enter":
				if strings.TrimSpace(m.picker.input.Value()) == "" {
					m.picker.message = "Enter a folder path"
					return m, nil
				}
				dir, err := validateSetting(downloadDirEnvVar, m.picker.input.Value())
				if err != nil {
					m.picker.message = err.Error()
					return m, nil
				}
				m.dir, m.accepted = dir, true
				return m, tea.Quit
			}
		}
	}
	if m.picker.page == "browse" {
		next, cmd := m.picker.Update(msg)
		m.picker = next.(settingsModel)
		return m, cmd
	}
	if m.editing {
		var cmd tea.Cmd
		m.picker.input, cmd = m.picker.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m downloadDirectoryModel) View() tea.View {
	w, h := m.picker.width, m.picker.height
	if m.picker.page == "browse" {
		rows := m.picker.rows()
		visible := max(1, h-7)
		start := max(0, m.picker.cursor-visible+1)
		var lines []string
		for i := start; i < min(len(rows), start+visible); i++ {
			lines = append(lines, selectionRow(rows[i], i == m.picker.cursor, w))
		}
		return screenView(w, h, "Choose a folder", bookCell(m.picker.directory), strings.Join(lines, "\n"), m.picker.message, uiKey("esc", "back"), uiKey("space", "select folder"), uiKey("enter", "open"))
	}
	dir, _ := filepath.Abs(m.dir)
	if m.editing {
		return screenView(w, h, "Choose another folder", "For this download only", "Folder path\n"+m.picker.input.View(), m.picker.message, uiKey("esc", "back"), uiKey("enter", "download here"), uiKey("ctrl+b", "browse"))
	}
	body := selectionRow("Download here", !m.change, w) + "\n" + selectionRow("Choose another folder…", m.change, w)
	return screenView(w, h, "Save to", bookCell(dir), body, m.picker.message, uiKey("↑/↓", "select"), uiKey("enter", "continue"), uiKey("esc", "download here"), uiKey("ctrl+c", "cancel"))
}
func chooseDownloadDirectory(cmd *cobra.Command, dir string) (string, bool, error) {
	if !isInteractive() || cmd.Flags().Changed("dir") {
		return dir, true, nil
	}
	result, err := tea.NewProgram(newDownloadDirectoryModel(dir)).Run()
	if err != nil {
		return "", false, err
	}
	m := result.(downloadDirectoryModel)
	return m.dir, m.accepted, nil
}
