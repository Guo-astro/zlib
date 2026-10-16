package cli

import (
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
		m.picker.input.SetWidth(max(1, min(56, size.Width-10)))
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
			case "e", "/":
				m.editing = true
				m.picker.message = ""
				m.picker.input.SetValue(m.dir)
				return m, m.picker.input.Focus()
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

func (m downloadDirectoryModel) View() tea.View { return m.dialogView("") }

// The shell supplies its selected book list as the backdrop. Direct commands
// use a blank download screen behind the same dialog.
func (m downloadDirectoryModel) dialogView(background string) tea.View {
	w, h := max(1, m.picker.width), max(1, m.picker.height)
	if w < 32 || h < 16 {
		if m.picker.page == "browse" {
			rows := m.picker.rows()
			visible := max(1, h-7)
			start := max(0, m.picker.cursor-visible+1)
			var lines []string
			for i := start; i < min(len(rows), start+visible); i++ {
				lines = append(lines, selectionRow(rows[i], i == m.picker.cursor, w))
			}
			return screenView(w, h, "Choose a folder", bookCell(m.picker.directory), strings.Join(lines, "\n"), m.picker.message,
				uiKey("esc", "back"), uiKey("space", "use folder"), uiKey("enter", "open"))
		}
		if m.editing {
			return screenView(w, h, "Choose another folder", "For this download", m.picker.input.View(), m.picker.message,
				uiKey("esc", "back"), uiKey("enter", "download here"), uiKey("ctrl+b", "browse"))
		}
		focused := 1
		if m.change {
			focused = 2
		}
		body := strings.Join(actionButtonRows([]string{"Download here", "Choose another folder"}, focused, w), "\n")
		return screenView(w, h, "Save to", bookCell(m.dir), body, m.picker.message,
			uiKey("↑/↓", "select"), uiKey("enter", "continue"), uiKey("esc", "download here"))
	}
	inner := min(60, w-10)
	base := lipgloss.NewStyle().Foreground(currentTheme.Title).Background(currentTheme.Surface)
	muted := lipgloss.NewStyle().Foreground(currentTheme.Muted).Background(currentTheme.Surface)
	dir, _ := filepath.Abs(m.dir)
	lines := []string{base.Bold(true).Render("Save to"), muted.Render("For this download"), "", muted.Render("Folder"), ansi.Truncate(bookCell(dir), inner-4, "…"), ""}
	controls := "↑/↓ select · Enter continue · Esc use this folder"
	if m.picker.page == "browse" {
		lines[0] = base.Bold(true).Render("Choose a folder")
		lines[3] = muted.Render("Current folder")
		lines[4] = ansi.Truncate(bookCell(m.picker.directory), inner-4, "…")
		rows := m.picker.rows()
		visible := h - 12
		if m.picker.message != "" {
			visible -= 2
		}
		visible = max(1, min(10, visible))
		start := max(0, m.picker.cursor-visible+1)
		for i := start; i < min(len(rows), start+visible); i++ {
			lines = append(lines, selectionRow(rows[i], i == m.picker.cursor, inner))
		}
		controls = "↑/↓ select · Enter open · Space use folder · Esc back"
	} else if m.editing {
		lines[0] = base.Bold(true).Render("Choose another folder")
		lines = append(lines, ansi.Truncate(m.picker.input.View(), inner, "…"))
		controls = "Enter download here · Ctrl+B browse · Esc back"
	} else {
		focused := 1
		if m.change {
			focused = 2
		}
		lines = append(lines, actionButtonRows([]string{"Download here", "Choose another folder"}, focused, inner)...)
	}
	if m.picker.message != "" {
		lines = append(lines, "", muted.Render(ansi.Truncate(m.picker.message, inner, "…")))
	}
	if w < 56 {
		switch {
		case m.picker.page == "browse":
			controls = "Enter open · Space use · Esc"
		case m.editing:
			controls = "Enter save · Ctrl+B browse · Esc"
		default:
			controls = "↑/↓ · Enter · Esc"
		}
	}
	lines = append(lines, "", muted.Render(ansi.Truncate(controls, inner, "…")))
	panel := base.Width(inner).Padding(1, 1).Border(lipgloss.RoundedBorder()).BorderForeground(currentTheme.Accent).Render(strings.Join(lines, "\n"))
	if background == "" {
		background = screenView(w, h, "Download", "", "", "").Content
	}
	dim := lipgloss.NewStyle().Foreground(currentTheme.Muted).Render(ansi.Strip(background))
	x, y := max(0, (w-lipgloss.Width(panel))/2), max(0, (h-lipgloss.Height(panel))/2)
	return fullScreenView(w, h, lipgloss.NewCompositor(lipgloss.NewLayer(dim), lipgloss.NewLayer(panel).X(x).Y(y)).Render())
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
