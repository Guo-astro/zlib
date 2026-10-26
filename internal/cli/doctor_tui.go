package cli

import (
	"context"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zlib "github.com/heartleo/zlib/internal/zlib"
)

type doctorProbeFunc func(context.Context, []string, string) []zlib.DomainCheck

type doctorResultMsg struct {
	sequence uint64
	checks   []zlib.DomainCheck
}

type doctorDomainItem struct {
	check   zlib.DomainCheck
	current bool
}

func (i doctorDomainItem) Title() string {
	if i.current {
		return i.check.Domain + "  · Current domain"
	}
	return i.check.Domain
}
func (i doctorDomainItem) Description() string { return "" }
func (i doctorDomainItem) FilterValue() string { return i.check.Domain }

type doctorPageModel struct {
	list       list.Model
	spinner    spinner.Model
	probe      doctorProbeFunc
	candidates []string
	proxy      string
	current    string
	ctx        context.Context
	cancel     context.CancelFunc
	sequence   uint64
	loading    bool
	width      int
	height     int
}

func newDoctorPageModel(candidates []string, proxy, current string, probe doctorProbeFunc) doctorPageModel {
	if probe == nil {
		probe = checkAvailableDomains
	}
	items := list.New(nil, doctorListDelegate(80), 80, 18)
	items.SetShowTitle(false)
	items.SetShowHelp(false)
	items.SetShowStatusBar(false)
	items.SetShowFilter(false)
	items.SetFilteringEnabled(false)
	items.DisableQuitKeybindings()
	items.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(currentTheme.Muted)
	items.Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	items.Styles.InactivePaginationDot = lipgloss.NewStyle().Foreground(currentTheme.Muted)
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	ctx, cancel := context.WithCancel(context.Background())
	return doctorPageModel{
		list: items, spinner: s, probe: probe, candidates: candidates, proxy: proxy,
		current: current, ctx: ctx, cancel: cancel, sequence: 1, loading: true, width: 80, height: 24,
	}
}

func doctorListDelegate(width int) list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(1)
	delegate.Styles.NormalTitle = lipgloss.NewStyle().Foreground(currentTheme.Title).PaddingLeft(2)
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(currentTheme.Accent).
		PaddingLeft(1).
		Foreground(currentTheme.Accent).
		Background(currentTheme.Surface).
		Bold(true).
		Width(max(1, width-2))
	return delegate
}

func (m doctorPageModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.check())
}

func (m doctorPageModel) check() tea.Cmd {
	ctx, candidates, proxy, sequence := m.ctx, m.candidates, m.proxy, m.sequence
	return func() tea.Msg {
		return doctorResultMsg{sequence: sequence, checks: m.probe(ctx, candidates, proxy)}
	}
}

func (m *doctorPageModel) stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

func (m doctorPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(max(1, msg.Width), max(1, msg.Height-6))
		m.list.SetDelegate(doctorListDelegate(max(1, msg.Width)))
		return m, nil
	case doctorResultMsg:
		if msg.sequence != m.sequence || !m.loading {
			return m, nil
		}
		m.loading = false
		available := availableDomains(msg.checks)
		items := make([]list.Item, 0, len(available))
		selected := 0
		for _, check := range available {
			if check.Domain == m.current {
				selected = len(items)
			}
			items = append(items, doctorDomainItem{check: check, current: check.Domain == m.current})
		}
		cmd := m.list.SetItems(items)
		m.list.Select(selected)
		return m, cmd
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.stop()
			return m, tea.Quit
		case "r":
			m.stop()
			m.ctx, m.cancel = context.WithCancel(context.Background())
			m.sequence++
			m.loading = true
			m.list.SetItems(nil)
			return m, tea.Batch(m.spinner.Tick, m.check())
		}
	}
	if !m.loading && len(m.list.Items()) > 0 {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m doctorPageModel) View() tea.View {
	body := ""
	switch {
	case m.loading:
		return screenView(m.width, m.height, "Doctor", "", m.spinner.View(), "",
			uiKey("esc", "back"))
	case len(m.list.Items()) == 0:
		body = "No available EAPI domains."
	default:
		body = m.list.View()
	}
	return screenView(m.width, m.height, "Doctor", "", body, "",
		uiKey("↑/↓", "select"), uiKey("r", "check again"), uiKey("esc", "back"))
}
