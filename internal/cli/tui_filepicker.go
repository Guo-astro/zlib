package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type kindleFileModel struct {
	picker                              filepicker.Model
	input                               textinput.Model
	width, height                       int
	editing, accepted, confirm, focused bool
	path, message                       string
}

func newKindleFileModel(dir string) kindleFileModel {
	fp := filepicker.New()
	fp.CurrentDirectory = dir
	fp.AllowedTypes = kindleAllowedTypes
	fp.AutoHeight = false
	fp.SetHeight(16)
	fp.ShowPermissions = false
	fp.ShowSize = false
	fp.ShowHidden = true
	fp.KeyMap.GoToTop.SetKeys("home", "g")
	fp.KeyMap.GoToLast.SetKeys("end", "G")
	fp.Styles.Selected = lipgloss.NewStyle().Foreground(currentTheme.Title).Background(currentTheme.Surface).Bold(true)
	fp.Styles.Directory = lipgloss.NewStyle().Foreground(currentTheme.Link)
	fp.Styles.File = lipgloss.NewStyle().Foreground(currentTheme.Title)
	fp.Styles.Cursor = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	fp.Styles.DisabledFile = lipgloss.NewStyle().Foreground(currentTheme.Muted)
	fp.Styles.EmptyDirectory = lipgloss.NewStyle().Foreground(currentTheme.Muted).SetString("No files. Press / to enter another directory.")
	input := textinput.New()
	input.Prompt = "> "
	return kindleFileModel{picker: fp, input: input, width: 80, height: 24}
}
func (m kindleFileModel) Init() tea.Cmd { return m.picker.Init() }
func (m kindleFileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		m.picker.SetHeight(max(1, size.Height-8))
		m.input.SetWidth(max(1, size.Width-4))
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if m.confirm {
			switch k.String() {
			case "esc", "ctrl+c", "n":
				m.confirm = false
			case "left", "right", "tab", "shift+tab":
				m.focused = !m.focused
			case "enter":
				if m.focused {
					m.accepted = true
					return m, tea.Quit
				}
				m.confirm = false
			}
			return m, nil
		}
		if k.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.editing {
			switch k.String() {
			case "esc":
				m.editing = false
				return m, nil
			case "enter":
				dir, err := validateSetting(downloadDirEnvVar, m.input.Value())
				if err != nil || dir == "" {
					m.message = "Enter an existing directory"
					return m, nil
				}
				m.picker = newKindleFileModel(dir).picker
				m.picker.SetHeight(max(1, m.height-8))
				m.editing = false
				m.message = ""
				return m, m.picker.Init()
			}
		} else {
			switch k.String() {
			case "esc", "q":
				return m, tea.Quit
			case "/":
				m.editing = true
				m.input.SetValue(m.picker.CurrentDirectory)
				m.input.CursorEnd()
				return m, m.input.Focus()
			}
		}
	}
	if m.editing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	if ok, path := m.picker.DidSelectFile(msg); ok {
		m.path = path
		m.confirm = true
		m.focused = false
	}
	if ok, _ := m.picker.DidSelectDisabledFile(msg); ok {
		m.message = "Unsupported file type. Choose an EPUB, PDF or another Kindle-supported file."
	}
	return m, cmd
}
func (m kindleFileModel) View() tea.View {
	body := m.picker.View()
	subtitle := bookCell(m.picker.CurrentDirectory)
	if m.editing {
		body = m.input.View()
		subtitle = "Enter a directory path"
	}
	v := screenView(m.width, m.height, "Send to Kindle", subtitle, body, m.message, uiKey("esc", "cancel"), uiKey("enter", "open/select"), uiKey("←", "parent"), uiKey("/", "path"))
	if m.confirm {
		v.Content = confirmationDialog(v.Content, m.width, m.height, "Send to Kindle?", filepath.Base(m.path), "", "Send this file using your saved Kindle settings.", "Send", m.focused, false)
	}
	return v
}
func chooseKindleFile() (string, bool, error) {
	if !isInteractive() {
		return "", false, fmt.Errorf("provide a file: zlib kindle send <file>")
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", false, err
	}
	result, err := tea.NewProgram(newKindleFileModel(dir)).Run()
	if err != nil {
		return "", false, err
	}
	m := result.(kindleFileModel)
	return m.path, m.accepted, nil
}
