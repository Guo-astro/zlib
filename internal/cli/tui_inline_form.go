package cli

import (
	"fmt"
	"slices"
	"strings"

	keys "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// inlineForm keeps login and search fields visible while editing.
type inlineForm struct {
	formModel
	inputs []textinput.Model
}

func newInlineForm(title, action string, fields []formField) inlineForm {
	m := inlineForm{formModel: newFormModel(title, action, fields)}
	for _, f := range fields {
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
	}
	m.focus(0)
	return m
}

func (m inlineForm) Init() tea.Cmd { return m.focus(m.cursor) }

func (m *inlineForm) focus(index int) tea.Cmd {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.cursor = (index + len(m.fields) + 1) % (len(m.fields) + 1)
	m.optionCursor = 0
	if m.cursor < len(m.fields) && len(m.fields[m.cursor].options) == 0 {
		return m.inputs[m.cursor].Focus()
	}
	return nil
}

func (m inlineForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		for i := range m.inputs {
			m.inputs[i].SetWidth(max(1, size.Width-4))
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
			f := &m.fields[m.cursor]
			switch k.String() {
			case "left", "up":
				m.optionCursor = max(0, m.optionCursor-1)
			case "right", "down":
				m.optionCursor = min(len(f.options)-1, m.optionCursor+1)
			case "space":
				value := f.options[m.optionCursor]
				if i := slices.Index(f.selected, value); i >= 0 {
					f.selected = slices.Delete(f.selected, i, i+1)
				} else {
					f.selected = append(f.selected, value)
				}
			}
			return m, nil
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
		lines = append(lines, f.label)
		if len(f.options) == 0 {
			lines = append(lines, m.inputs[i].View())
		} else {
			var row string
			for j, option := range f.options {
				mark := "[ ]"
				if slices.Contains(f.selected, option) {
					mark = "[x]"
				}
				item := mark + " " + option
				if m.cursor == i && m.optionCursor == j {
					item = ">" + item
				}
				if len(row)+len(item)+2 > m.width && row != "" {
					lines = append(lines, row)
					row = ""
				}
				row += item + "  "
			}
			lines = append(lines, row, f.hint)
		}
		lines = append(lines, "")
	}
	lines = append(lines, selectionRow(m.action, m.cursor == len(m.fields), m.width))
	bindings := []keys.Binding{uiKey("esc", "cancel"), uiKey("tab", "next"), uiKey("shift+tab", "previous"), uiKey("enter", "continue"), uiKey("ctrl+s", strings.ToLower(m.action))}
	if m.cursor < len(m.fields) && len(m.fields[m.cursor].options) > 0 {
		bindings = append(bindings, uiKey("←/→", "format"), uiKey("space", "toggle"))
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
