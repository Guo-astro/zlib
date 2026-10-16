package cli

import (
	"fmt"
	"strings"

	keys "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// inlineForm keeps login and search fields visible while editing.
type inlineForm struct {
	formModel
	inputs  []textinput.Model
	choices []*huh.MultiSelect[string]
}

func newInlineForm(title, action string, fields []formField) inlineForm {
	m := inlineForm{formModel: newFormModel(title, action, fields)}
	for i, f := range fields {
		input := textinput.New()
		input.Prompt = "> "
		input.Placeholder = f.hint
		input.KeyMap.Paste.SetKeys("ctrl+v", "ctrl+shift+v", "shift+insert")
		input.SetWidth(60)
		if f.secret {
			input.EchoMode = textinput.EchoPassword
		}
		input.SetValue(f.value)
		m.inputs = append(m.inputs, input)
		if len(f.options) == 0 {
			m.choices = append(m.choices, nil)
			continue
		}
		options := make([]huh.Option[string], 0, len(f.options))
		for _, option := range f.options {
			options = append(options, huh.NewOption(option, option))
		}
		choice := huh.NewMultiSelect[string]().Title(f.label).Description(f.hint).
			Filterable(false).Options(options...).Value(&m.fields[i].selected).
			Width(76).Height(len(options) + 2)
		choice.WithTheme(huh.ThemeFunc(func(dark bool) *huh.Styles {
			styles := huh.ThemeBase(dark)
			styles.Focused.Base = lipgloss.NewStyle().BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(currentTheme.Accent)
			styles.Blurred.Base = lipgloss.NewStyle().BorderLeft(true).BorderStyle(lipgloss.HiddenBorder())
			for _, field := range []*huh.FieldStyles{&styles.Focused, &styles.Blurred} {
				field.Title = lipgloss.NewStyle().Foreground(currentTheme.Title).Bold(true)
				field.Description = lipgloss.NewStyle().Foreground(currentTheme.Muted)
				field.MultiSelectSelector = lipgloss.NewStyle().Foreground(currentTheme.Accent).SetString("› ")
				field.SelectedPrefix = lipgloss.NewStyle().Foreground(currentTheme.Success).SetString("[✓] ")
				field.UnselectedPrefix = lipgloss.NewStyle().Foreground(currentTheme.Muted).SetString("[ ] ")
				field.SelectedOption = lipgloss.NewStyle().Foreground(currentTheme.Title).Bold(true)
				field.UnselectedOption = lipgloss.NewStyle().Foreground(currentTheme.Title)
			}
			return styles
		}))
		choice.WithKeyMap(huh.NewDefaultKeyMap())
		m.choices = append(m.choices, choice)
	}
	m.focus(0)
	return m
}

func (m inlineForm) Init() tea.Cmd { return m.focus(m.cursor) }

func (m *inlineForm) focus(index int) tea.Cmd {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	for _, choice := range m.choices {
		if choice != nil {
			choice.Blur()
		}
	}
	m.cursor = (index + len(m.fields) + 1) % (len(m.fields) + 1)
	if m.cursor < len(m.fields) && len(m.fields[m.cursor].options) == 0 {
		return m.inputs[m.cursor].Focus()
	}
	if m.cursor < len(m.fields) && m.choices[m.cursor] != nil {
		return m.choices[m.cursor].Focus()
	}
	return nil
}

func (m inlineForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		for i := range m.inputs {
			m.inputs[i].SetWidth(max(1, size.Width-4))
		}
		for i, choice := range m.choices {
			if choice != nil {
				hint := m.fields[i].hint
				if size.Width < 48 {
					hint = "Empty = all formats"
				}
				choice.Description(hint)
				choice.Width(max(1, size.Width-2))
				choice.Height(min(len(m.fields[i].options)+2, max(3, size.Height-12)))
			}
		}
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "ctrl+c":
			return m, tea.Quit
		case "tab", "enter":
			if k.String() == "enter" && m.cursor == len(m.fields) {
				cmd := m.submit()
				if !m.accepted {
					cmd = m.focus(m.cursor)
				}
				return m, cmd
			}
			return m, m.focus(m.cursor + 1)
		case "shift+tab":
			return m, m.focus(m.cursor - 1)
		case "ctrl+s":
			cmd := m.submit()
			if !m.accepted {
				cmd = m.focus(m.cursor)
			}
			return m, cmd
		}
		if m.cursor < len(m.fields) && len(m.fields[m.cursor].options) > 0 {
			_, cmd := m.choices[m.cursor].Update(msg)
			return m, cmd
		}
	}
	if m.cursor < len(m.fields) && len(m.fields[m.cursor].options) == 0 {
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		m.fields[m.cursor].value = m.inputs[m.cursor].Value()
		if m.inputs[m.cursor].Err != nil {
			m.message = "Unable to paste. Try your terminal's Paste command."
		}
		return m, cmd
	}
	return m, nil
}

func (m inlineForm) View() tea.View {
	var lines []string
	for i, f := range m.fields {
		if len(f.options) == 0 {
			lines = append(lines, f.label, m.inputs[i].View())
		} else {
			lines = append(lines, m.choices[i].View())
		}
		lines = append(lines, "")
	}
	focused := m.cursor == len(m.fields)
	if m.action == "Search" {
		button := lipgloss.NewStyle().Foreground(currentTheme.Title).
			Background(currentTheme.Surface).Padding(0, 2)
		if focused {
			button = button.Background(currentTheme.Accent).Foreground(currentTheme.Surface).Bold(true)
		}
		lines = append(lines, button.Render(m.action))
	} else {
		lines = append(lines, selectionRow(m.action, focused, m.width))
	}
	bindings := []keys.Binding{uiKey("esc", "cancel"), uiKey("tab", "next"), uiKey("shift+tab", "previous"), uiKey("enter", "continue"), uiKey("ctrl+s", strings.ToLower(m.action))}
	if m.cursor < len(m.fields) && len(m.fields[m.cursor].options) > 0 {
		bindings = append(bindings, uiKey("↑/↓", "format"), uiKey("space", "toggle"), uiKey("ctrl+a", "all / none"))
	}
	return screenView(m.width, m.height, m.title, "", strings.Join(lines, "\n"), m.message, bindings...)
}

func runInlineForm(m inlineForm) ([]formField, bool, error) {
	if !isInteractive() {
		return nil, false, fmt.Errorf("%s requires a terminal", strings.ToLower(m.title))
	}
	m.focus(0)
	result, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, false, err
	}
	final := result.(inlineForm)
	return final.fields, final.accepted, nil
}
