package cli

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

type profileLoadedMsg struct {
	profile zlib.UserProfile
	err     error
}

type profilePageModel struct {
	client   *zlib.Client
	viewport viewport.Model
	spinner  spinner.Model
	profile  zlib.UserProfile
	loaded   bool
	err      error
	width    int
	height   int
}

func newProfilePageModel(client *zlib.Client) profilePageModel {
	return profilePageModel{client: client, spinner: newLoadingSpinner(), viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(18)), width: 80, height: 24}
}

func (m profilePageModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.load())
}

func (m profilePageModel) load() tea.Cmd {
	return func() tea.Msg {
		profile, err := m.client.GetProfile()
		return profileLoadedMsg{profile: profile, err: err}
	}
}

func (m profilePageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if !m.loaded {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewport.SetWidth(max(1, msg.Width))
		m.viewport.SetHeight(max(1, msg.Height-6))
		m.updateContent()
		return m, nil
	case profileLoadedMsg:
		m.profile, m.err, m.loaded = msg.profile, msg.err, true
		m.updateContent()
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			if m.err != nil {
				m.err, m.loaded = nil, false
				m.updateContent()
				return m, tea.Batch(m.spinner.Tick, m.load())
			}
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *profilePageModel) updateContent() {
	content := ""
	if m.err != nil {
		content = "Could not load profile: " + m.err.Error() + "\n\nPress r to retry."
	} else if m.loaded {
		content = renderProfilePage(m.profile, m.width)
	}
	m.viewport.SetContent(content)
}

func (m profilePageModel) View() tea.View {
	if !m.loaded {
		return screenView(m.width, m.height, "Profile", "", m.spinner.View(), "", uiKey("esc", "back"))
	}
	status := ""
	if m.viewport.TotalLineCount() > m.viewport.Height() {
		status = fmt.Sprintf("%d%%", int(m.viewport.ScrollPercent()*100))
	}
	return screenView(m.width, m.height, "Profile", "Account and download quota", m.viewport.View(), status,
		uiKey("↑/↓", "scroll"), uiKey("r", "retry"), uiKey("esc", "back"))
}

func renderProfilePage(profile zlib.UserProfile, width int) string {
	width = max(1, width)
	section := lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true)
	primary := lipgloss.NewStyle().Foreground(currentTheme.Title)
	muted := lipgloss.NewStyle().Foreground(currentTheme.Muted)
	remaining := lipgloss.NewStyle().Foreground(remainingColor(profile.DailyRemaining)).Bold(true)
	rows := []string{}
	if profile.Name != "" {
		rows = append(rows, primary.Bold(true).Render(bookCell(profile.Name)), "")
	}
	rows = append(rows, section.Render("DAILY DOWNLOADS"), "")
	if !profile.QuotaKnown {
		rows = append(rows, muted.Render("Quota unavailable"))
	} else {
		limits := profile.DownloadLimit
		bar := progress.New(progress.WithWidth(min(40, max(1, width-2))),
			progress.WithColors(remainingColor(limits.DailyRemaining), currentTheme.Surface),
			progress.WithoutPercentage(), progress.WithFillCharacters('━', '─'))
		fraction := 0.0
		if limits.DailyAllowed > 0 {
			fraction = float64(limits.DailyAmount) / float64(limits.DailyAllowed)
		}
		rows = append(rows, bar.ViewAs(fraction),
			primary.Render(fmt.Sprintf("%d / %d used", limits.DailyAmount, limits.DailyAllowed)),
			"",
			"Remaining  "+remaining.Render(fmt.Sprint(limits.DailyRemaining)))
		if limits.DailyReset != "" {
			rows = append(rows, "Resets at  "+primary.Render(bookCell(limits.DailyReset)))
		}
	}
	rows = append(rows, "", section.Render("ACCOUNT"))
	field := func(label, value string) {
		if value == "" {
			return
		}
		if width < 32 {
			rows = append(rows, muted.Render(label))
			for _, line := range strings.Split(ansi.Hardwrap(bookCell(value), width, false), "\n") {
				rows = append(rows, primary.Render(line))
			}
			return
		}
		styledLabel := muted.Width(16).Render(label)
		for i, line := range strings.Split(ansi.Hardwrap(bookCell(value), width-18, false), "\n") {
			if i == 0 {
				rows = append(rows, styledLabel+"  "+primary.Render(line))
			} else {
				rows = append(rows, strings.Repeat(" ", 18)+primary.Render(line))
			}
		}
	}
	field("Email", profile.Email)
	field("Kindle email", profile.KindleEmail)
	if profile.Premium != nil {
		if *profile.Premium {
			field("Plan", "Premium")
		} else {
			field("Plan", "Standard")
		}
	}
	if profile.Confirmed != nil {
		if *profile.Confirmed {
			field("Email confirmed", "Yes")
		} else {
			field("Email confirmed", "No")
		}
	}
	if profile.DonationsActive != nil {
		if *profile.DonationsActive {
			field("Donation", "Active")
		} else {
			field("Donation", "Inactive")
		}
	}
	field("Donation expires", profile.DonationsExpire)
	if len(profile.PersonalDomains) > 0 {
		rows = append(rows, "", section.Render("PERSONAL DOMAINS"))
		for _, domain := range profile.PersonalDomains {
			rows = append(rows, primary.Render(bookCell(domain)))
		}
	}
	return strings.Join(rows, "\n")
}
