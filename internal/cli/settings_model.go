package cli

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	keys "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
	"github.com/heartleo/zlib/internal/config"
)

const (
	kindleToKey   = "KINDLE_TO"
	kindleFromKey = "KINDLE_FROM"
	kindleHostKey = "KINDLE_SMTP_HOST"
	kindlePortKey = "KINDLE_SMTP_PORT"
)

var kindleSettingKeys = []string{kindleToKey, kindleFromKey, kindleHostKey, kindlePortKey}

type settingsProbeMsg struct {
	generation int
	domains    []string
}
type settingsModel struct {
	store          settingsStore
	values         []settingValue
	kindle         zlib.KindleConfig
	page           string
	cursor         int
	width, height  int
	input          textinput.Model
	key            string
	message        string
	domains        []string
	busy           bool
	spinner        spinner.Model
	generation     int
	cancel         context.CancelFunc
	ctx            context.Context
	probeReturn    string
	directory      string
	directories    []string
	discard        bool
	discardFocused bool
	editAction     int
	original       string
	returnPage     string
	returnCursor   int
}

func newSettingsModel(ctx context.Context, store settingsStore, page string) (settingsModel, error) {
	values, err := store.values()
	if err != nil {
		return settingsModel{}, err
	}
	kindle, err := config.LoadKindleConfigFrom(store.kindle)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return settingsModel{}, err
	}
	values = append(values, kindleSettingValues(kindle)...)
	input := textinput.New()
	input.Prompt = ""
	input.SetWidth(48)
	styles := input.Styles()
	styles.Focused.Text = lipgloss.NewStyle().Foreground(currentTheme.Title)
	styles.Cursor.Color = currentTheme.Accent
	input.SetStyles(styles)
	return settingsModel{store: store, values: values, kindle: kindle, page: page, input: input, ctx: ctx, width: 80, height: 24}, nil
}

func kindleSettingValues(cfg zlib.KindleConfig) []settingValue {
	values := []settingValue{
		{Key: kindleToKey, Value: cfg.To, Saved: cfg.To},
		{Key: kindleFromKey, Value: cfg.From, Saved: cfg.From},
		{Key: kindleHostKey, Value: cfg.SMTPHost, Saved: cfg.SMTPHost},
		{Key: kindlePortKey, Saved: strconv.Itoa(cfg.SMTPPort)},
	}
	if cfg.SMTPPort > 0 {
		values[3].Value = strconv.Itoa(cfg.SMTPPort)
	} else {
		values[3].Value, values[3].Saved = strconv.Itoa(defaultKindleSMTPPort), ""
	}
	if cfg.SMTPHost == "" {
		values[2].Value = defaultKindleSMTPHost
	}
	for i := range values {
		values[i].Source = "Kindle configuration"
		if values[i].Saved == "" {
			values[i].Source = "Default"
		}
	}
	return values
}

func isKindleSetting(key string) bool {
	switch key {
	case kindleToKey, kindleFromKey, kindleHostKey, kindlePortKey:
		return true
	}
	return false
}

func (m settingsModel) Init() tea.Cmd { return nil }
func (m settingsModel) value(key string) settingValue {
	for _, v := range m.values {
		if v.Key == key {
			return v
		}
	}
	return settingValue{Key: key}
}
func (m settingsModel) rows() []string {
	switch m.page {
	case "network":
		return []string{"Domain", "Proxy URL"}
	case "download":
		return []string{"Default directory", "Download retries"}
	case "kindle":
		return []string{"Kindle address", "From address", "SMTP host", "SMTP port"}
	case "domain":
		return m.domains
	case "browse":
		return append([]string{".."}, m.directories...)
	default:
		return []string{"Domain", "Proxy URL", "Default directory", "Download retries", "Kindle address", "From address", "SMTP host", "SMTP port"}
	}
}

func (m *settingsModel) edit(key string) tea.Cmd {
	m.returnPage = m.page
	m.returnCursor = m.cursor
	m.key = key
	m.page = "edit"
	m.message = ""
	m.discard = false
	m.editAction = 0
	v := m.value(key)
	value := v.Value
	if (v.Source == "Default" && !isKindleSetting(key)) || (key == zlib.EnvDomain && v.Source == "Saved session") {
		value = v.Saved
	}
	m.original = value
	m.input.SetValue(value)
	m.input.CursorEnd()
	m.input.SetWidth(max(1, min(56, m.width-8)))
	if key == zlib.EnvProxy {
		m.input.EchoMode = textinput.EchoPassword
	} else {
		m.input.EchoMode = textinput.EchoNormal
	}
	return m.input.Focus()
}

func (m *settingsModel) back() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.generation++
	m.busy = false
	m.cursor = 0
	m.message = ""
	switch m.page {
	case "domain":
		m.page = m.returnPage
		if m.page == "" {
			m.page = "network"
		}
		m.cursor = m.returnCursor
	case "edit":
		m.page = m.returnPage
		if m.page == "" {
			m.page = "home"
		}
		m.cursor = m.returnCursor
	case "browse":
		m.page = "edit"
		m.editAction = 0
		m.input.Focus()
	default:
		m.page = "home"
	}
}

func (m *settingsModel) probe(proxy string, page string) tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.cancel = cancel
	m.generation++
	generation := m.generation
	m.busy = true
	m.message = ""
	m.spinner = newLoadingSpinner()
	m.probeReturn = page
	candidates := zlib.KnownDomains()
	if d, err := zlib.ParseAllowedDomain(m.value(zlib.EnvDomain).Value); err == nil {
		candidates = append(candidates, d)
	}
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		results := checkAvailableDomains(ctx, candidates, proxy)
		seen := map[string]bool{}
		var domains []string
		for _, result := range results {
			u, err := url.Parse(result.FinalURL)
			if err != nil {
				continue
			}
			d := u.Scheme + "://" + u.Host
			if !seen[d] {
				seen[d] = true
				domains = append(domains, d)
			}
		}
		sort.Strings(domains)
		return settingsProbeMsg{generation, domains}
	})
}

func (m *settingsModel) effectiveProxy() string {
	v := m.value(zlib.EnvProxy)
	if v.Source == "Default" {
		return ""
	}
	return v.Value
}

func (m *settingsModel) save(value string) {
	if isKindleSetting(m.key) {
		m.saveKindle(value)
		return
	}
	if err := m.store.save(m.key, value); err != nil {
		m.message = "Could not save: " + err.Error()
		return
	}
	values, err := m.store.values()
	if err != nil {
		m.message = "Saved; could not reload settings: " + err.Error()
		return
	}
	m.values = append(values, kindleSettingValues(m.kindle)...)
	v := m.value(m.key)
	// Update only this process's derived settings. Preserve explicit overrides.
	if v.Override != "" {
		// Explicit process and working-directory settings keep precedence.
	} else if v.Source == "Global configuration" {
		_ = os.Setenv(m.key, v.Value)
	} else if v.Source == "Default" || v.Source == "Saved session" {
		_ = os.Unsetenv(m.key)
	}
	m.back()
	m.message = "Saved"
	if v.Override != "" {
		m.message = "Saved globally; overridden by " + v.Override
	}
	m.domains = nil
}

func (m *settingsModel) saveKindle(value string) {
	if strings.ContainsAny(value, "\r\n\x00") {
		m.message = "Enter a single-line value"
		return
	}
	value = strings.TrimSpace(value)
	cfg := m.kindle
	switch m.key {
	case kindleToKey:
		cfg.To = value
	case kindleFromKey:
		cfg.From = value
	case kindleHostKey:
		cfg.SMTPHost = value
	case kindlePortKey:
		if value == "" {
			cfg.SMTPPort = 0
		} else {
			port, err := strconv.Atoi(value)
			if err != nil || port < 1 || port > 65535 {
				m.message = "Enter a port from 1 to 65535"
				return
			}
			cfg.SMTPPort = port
		}
	}
	if cfg.SMTPHost == "" {
		cfg.SMTPHost = defaultKindleSMTPHost
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = defaultKindleSMTPPort
	}
	if err := config.SaveKindleConfigTo(m.store.kindle, cfg); err != nil {
		m.message = "Could not save: " + err.Error()
		return
	}
	m.kindle = cfg
	m.values = append(m.values[:4], kindleSettingValues(cfg)...)
	m.back()
	m.message = "Saved"
}

func (m *settingsModel) browse(path string) {
	p, err := expandTilde(path)
	if err == nil {
		p, err = filepath.Abs(p)
	}
	if err != nil {
		m.message = "Invalid directory"
		return
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		m.message = "Cannot open directory"
		return
	}
	m.directory = p
	m.directories = nil
	for _, entry := range entries {
		if entry.IsDir() {
			m.directories = append(m.directories, entry.Name())
		}
	}
	m.page = "browse"
	m.cursor = 0
	m.message = ""
}

func (m settingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.busy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.SetWidth(max(1, min(56, m.width-8)))
		return m, nil
	case settingsProbeMsg:
		if msg.generation != m.generation {
			return m, nil
		}
		m.busy = false
		m.cancel = nil
		if m.probeReturn == "edit" {
			m.message = fmt.Sprintf("Proxy test: %d available domains", len(msg.domains))
			return m, nil
		}
		m.domains = msg.domains
		m.cursor = 0
		for i, d := range m.domains {
			if d == m.value(zlib.EnvDomain).Value {
				m.cursor = i
			}
		}
		m.message = fmt.Sprintf("%d available domains · r refresh", len(m.domains))
		return m, nil
	case tea.KeyPressMsg:
		k := msg.String()
		if k == "ctrl+c" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
		if m.discard {
			if k == "left" || k == "right" || k == "tab" {
				m.discardFocused = !m.discardFocused
				return m, nil
			}
			if k == "y" || (k == "enter" && m.discardFocused) {
				m.discard = false
				m.back()
			} else if k == "n" || k == "esc" || k == "enter" {
				m.discard = false
			}
			return m, nil
		}
		if k == "esc" {
			if m.page == "home" {
				return m, tea.Quit
			}
			if m.page == "edit" && m.input.Value() != m.original {
				m.discard = true
				m.discardFocused = false
				return m, nil
			}
			m.back()
			return m, nil
		}
		if m.page == "edit" {
			if k == "tab" || k == "shift+tab" {
				delta := 1
				if k == "shift+tab" {
					delta = -1
				}
				m.editAction = (m.editAction + delta + 5) % 5
				if m.key != downloadDirEnvVar && m.key != zlib.EnvProxy && m.editAction == 1 {
					m.editAction = (m.editAction + delta + 5) % 5
				}
				if m.editAction == 0 {
					return m, m.input.Focus()
				}
				m.input.Blur()
				return m, nil
			}
			if k == "enter" && m.editAction != 0 {
				switch m.editAction {
				case 1:
					if m.key == downloadDirEnvVar {
						k = "ctrl+b"
					} else if m.key == zlib.EnvProxy {
						k = "ctrl+t"
					} else {
						return m, nil
					}
				case 2:
					k = "ctrl+r"
				case 3:
					if m.input.Value() != m.original {
						m.discard = true
						m.discardFocused = false
					} else {
						m.back()
					}
					return m, nil
				}
			}
			switch k {
			case "enter":
				m.save(m.input.Value())
				return m, nil
			case "ctrl+r":
				m.input.SetValue("")
				m.message = "Default selected; Enter saves"
				return m, nil
			case "ctrl+b":
				if m.key == downloadDirEnvVar {
					p := m.input.Value()
					if p == "" {
						p, _ = os.Getwd()
					}
					m.browse(p)
				}
				return m, nil
			case "ctrl+t":
				if m.key == zlib.EnvProxy {
					v, err := validateSetting(m.key, m.input.Value())
					if err != nil {
						m.message = err.Error()
						return m, nil
					}
					if v == "" {
						m.message = "Enter a proxy URL to test"
						return m, nil
					}
					return m, m.probe(v, "edit")
				}
				return m, nil
			}
			if m.editAction != 0 {
				return m, nil
			}
			var cmd tea.Cmd
			before := m.input.Value()
			m.input, cmd = m.input.Update(msg)
			if m.busy && m.input.Value() != before {
				if m.cancel != nil {
					m.cancel()
				}
				m.generation++
				m.busy = false
				m.message = "Proxy changed; test again"
			}
			return m, cmd
		}
		if k == "q" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
		if m.page == "domain" && k == "r" {
			return m, m.probe(m.effectiveProxy(), "domain")
		}
		rows := m.rows()
		count := len(rows)
		switch k {
		case "up", "k", "shift+tab":
			m.cursor = max(0, m.cursor-1)
		case "down", "j", "tab":
			m.cursor = min(max(0, count-1), m.cursor+1)
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = max(0, count-1)
		case "pgdown":
			m.cursor = min(max(0, count-1), m.cursor+max(1, m.height-10))
		case "pgup":
			m.cursor = max(0, m.cursor-max(1, m.height-10))
		case "space":
			if m.page == "browse" {
				m.input.SetValue(m.directory)
				m.page = "edit"
				m.editAction = 0
				return m, m.input.Focus()
			}
		case "enter":
			if m.busy {
				return m, nil
			}
			switch m.page {
			case "home":
				switch m.cursor {
				case 0:
					m.returnPage = "home"
					m.returnCursor = m.cursor
					m.page = "domain"
					m.cursor = 0
					return m, m.probe(m.effectiveProxy(), "domain")
				case 1:
					return m, m.edit(zlib.EnvProxy)
				case 2:
					return m, m.edit(downloadDirEnvVar)
				case 3:
					return m, m.edit(zlib.EnvRetries)
				default:
					return m, m.edit(kindleSettingKeys[m.cursor-4])
				}
			case "network":
				if m.cursor == 0 {
					m.returnPage = "network"
					m.returnCursor = m.cursor
					m.page = "domain"
					m.cursor = 0
					return m, m.probe(m.effectiveProxy(), "domain")
				}
				return m, m.edit(zlib.EnvProxy)
			case "download":
				if m.cursor == 0 {
					return m, m.edit(downloadDirEnvVar)
				}
				return m, m.edit(zlib.EnvRetries)
			case "kindle":
				return m, m.edit(kindleSettingKeys[m.cursor])
			case "domain":
				if count > 0 {
					m.key = zlib.EnvDomain
					m.save(rows[m.cursor])
				}
			case "browse":
				if count > 0 {
					m.browse(filepath.Join(m.directory, rows[m.cursor]))
				}
			}
		}
		return m, nil
	}
	if m.page == "edit" {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m settingsModel) settingRows(width, available int) []string {
	labels := m.rows()
	start := 0
	switch m.page {
	case "download":
		start = 2
	case "kindle":
		start = 4
	}
	labelWidth := min(18, max(12, width-14))
	valueWidth := max(1, width-labelWidth-4)
	section := lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true)
	lines := make([]string, 0, len(labels)+5)
	selected := 0
	for i, label := range labels {
		if m.page == "home" && (i == 0 || i == 2 || i == 4) {
			if i > 0 {
				lines = append(lines, "")
			}
			group := []string{"NETWORK", "DOWNLOADS", "KINDLE"}[i/2]
			lines = append(lines, section.Render("  "+group))
		}
		v := m.values[start+i]
		value := displaySetting(v.Key, v.Value)
		if value == "" {
			if v.Key == kindleToKey || v.Key == kindleFromKey {
				value = "Add address"
			} else {
				value = "Automatic"
			}
		}
		name := ansi.Truncate(label, labelWidth, "…")
		row := name + strings.Repeat(" ", max(0, labelWidth-ansi.StringWidth(name))+2)
		row += ansi.Truncate(value, valueWidth, "…")
		if i == m.cursor {
			selected = len(lines)
		}
		lines = append(lines, styledListRow(row, i == m.cursor, width, currentTheme))
	}
	if len(lines) <= available {
		return lines
	}
	from := max(0, selected-available+1)
	return lines[from:min(len(lines), from+available)]
}

func (m settingsModel) selectedSetting() settingValue {
	index := m.cursor
	switch m.page {
	case "download":
		index += 2
	case "kindle":
		index += 4
	}
	if index >= 0 && index < len(m.values) {
		return m.values[index]
	}
	return settingValue{}
}

func (m settingsModel) settingsListView(width, height int, title string) tea.View {
	if height < 7 {
		return fullScreenView(width, height, title+"\nEnlarge terminal")
	}
	helpModel := help.New()
	helpModel.SetWidth(width)
	helpModel.Styles.ShortKey = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	helpModel.Styles.ShortDesc = lipgloss.NewStyle().Foreground(currentTheme.Muted)
	footer := helpModel.ShortHelpView([]keys.Binding{uiKey("esc", "back"), uiKey("enter", "edit"), uiKey("↑/↓", "select")})
	lines := []string{lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true).Render(title), ""}
	reserved := 1
	if m.message != "" || m.selectedSetting().Override != "" {
		reserved = 2
	}
	lines = append(lines, m.settingRows(width, max(1, height-3-reserved))...)
	if m.message != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(currentTheme.Muted).Render(m.message))
	} else if override := m.selectedSetting().Override; override != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(currentTheme.Muted).Render("Using "+override))
	}
	lines = append(lines, "", footer)
	return fullScreenView(width, height, strings.Join(lines, "\n"))
}

func settingsCompactView(width, height int, title, subtitle string, lines []string, status string, controls []keys.Binding) tea.View {
	if height < 7 {
		return fullScreenView(width, height, title+"\nEnlarge terminal")
	}
	helpModel := help.New()
	helpModel.SetWidth(width)
	helpModel.Styles.ShortKey = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	helpModel.Styles.ShortDesc = lipgloss.NewStyle().Foreground(currentTheme.Muted)
	content := []string{lipgloss.NewStyle().Foreground(currentTheme.Accent).Bold(true).Render(title)}
	if subtitle != "" {
		content = append(content, lipgloss.NewStyle().Foreground(currentTheme.Muted).Render(subtitle))
	}
	content = append(content, "")
	content = append(content, lines...)
	if status != "" {
		content = append(content, "", lipgloss.NewStyle().Foreground(currentTheme.Muted).Render(status))
	}
	content = append(content, "", helpModel.ShortHelpView(controls))
	return fullScreenView(width, height, strings.Join(content, "\n"))
}

func (m settingsModel) View() tea.View {
	w, h := max(1, m.width), max(1, m.height)
	titles := map[string]string{"home": "Settings", "network": "Settings / Network", "download": "Settings / Downloads", "kindle": "Settings / Kindle", "domain": "Settings / Domain", "browse": "Settings / Choose directory", "edit": "Settings / Edit"}
	if m.busy {
		return screenView(w, h, titles[m.page], "", m.spinner.View(), "", uiKey("esc", "back"))
	}
	subtitle := ""
	status := m.message
	controls := []keys.Binding{uiKey("esc", "back"), uiKey("enter", "open"), uiKey("↑/↓", "select"), uiKey("q", "quit")}
	var lines []string
	if m.page == "home" || m.page == "network" || m.page == "download" || m.page == "kindle" {
		return m.settingsListView(w, h, titles[m.page])
	} else if m.page == "edit" {
		v := m.value(m.key)
		labels := map[string]string{zlib.EnvProxy: "Proxy URL", downloadDirEnvVar: "Download directory", zlib.EnvRetries: "Download retries", kindleToKey: "Kindle address", kindleFromKey: "From address", kindleHostKey: "SMTP host", kindlePortKey: "SMTP port"}
		if v.Override != "" {
			subtitle = "Using " + v.Override
		}
		lines = append(lines, labels[m.key], m.input.View(), "")
		if isKindleSetting(m.key) {
			lines = append(lines, "SMTP password: ZLIB_SMTP_PWD", "")
		}
		actions := []string{"", "Reset", "Cancel", "Save"}
		if m.key == downloadDirEnvVar {
			actions[0] = "Browse"
		} else if m.key == zlib.EnvProxy {
			actions[0] = "Test"
		}
		var buttons []string
		for i, label := range actions {
			if label == "" {
				continue
			}
			style := lipgloss.NewStyle().Foreground(currentTheme.Title)
			if m.editAction == i+1 {
				style = style.Background(currentTheme.Accent).Foreground(currentTheme.Surface).Bold(true)
			}
			buttons = append(buttons, style.Render("["+label+"]"))
		}
		lines = append(lines, strings.Join(buttons, " "))
		controls = []keys.Binding{uiKey("esc", "cancel"), uiKey("enter", "save"), uiKey("tab", "actions")}
		if m.key == downloadDirEnvVar {
			controls = append(controls, uiKey("ctrl+b", "browse"))
		}
		if m.key == zlib.EnvProxy {
			controls = append(controls, uiKey("ctrl+t", "test"))
		}
	} else {
		rows := m.rows()
		values := make([]string, len(rows))
		labelNeed, valueNeed := 0, 0
		for i, label := range rows {
			switch m.page {
			case "network":
				values[i] = displaySetting(m.values[i].Key, m.values[i].Value)
			case "download":
				values[i] = displaySetting(m.values[i+2].Key, m.values[i+2].Value)
			case "domain":
				values[i] = "Available"
				if label == m.value(zlib.EnvDomain).Value {
					values[i] += " (current)"
				}
			}
			labelNeed = max(labelNeed, ansi.StringWidth(bookCell(label)))
			valueNeed = max(valueNeed, ansi.StringWidth(bookCell(values[i])))
		}
		// The shared allocator reserves a leading column gap; compensate here
		// because settings have no numeric column.
		lw, vw, gap := bookColumnWidths(w+3, 0, labelNeed, valueNeed)
		if gap == 2 {
			lw, vw, gap = bookColumnWidths(w+2, 0, labelNeed, valueNeed)
		}
		visible := max(1, h-10)
		start := max(0, m.cursor-visible+1)
		for i := start; i < min(len(rows), start+visible); i++ {
			label := rows[i]
			if values[i] != "" && vw > 0 {
				label = ansi.Truncate(bookCell(label), lw, "…")
				label += strings.Repeat(" ", max(0, lw-ansi.StringWidth(label))+gap) + ansi.Truncate(bookCell(values[i]), vw, "…")
			}
			lines = append(lines, styledListRow(label, i == m.cursor, w, currentTheme))
		}
		if m.page == "browse" {
			subtitle = bookCell(m.directory)
			controls = []keys.Binding{uiKey("esc", "back"), uiKey("space", "use folder"), uiKey("enter", "open")}
		}
		if m.page == "domain" {
			subtitle = "Current: " + m.value(zlib.EnvDomain).Value
			controls = []keys.Binding{uiKey("esc", "back"), uiKey("enter", "save"), uiKey("r", "refresh")}
			if !m.busy && len(rows) == 0 {
				lines = append(lines, "No available domains. Press r to retry.")
			}
		}
		if m.page == "network" || m.page == "download" {
			i := m.cursor
			if m.page == "download" {
				i += 2
			}
			v := m.values[i]
			lines = append(lines, "", "Source: "+v.Source, "Saved: "+displaySetting(v.Key, v.Saved))
		}
	}
	v := screenView(w, h, titles[m.page], subtitle, strings.Join(lines, "\n"), status, controls...)
	if m.page == "edit" {
		v = settingsCompactView(w, h, titles[m.page], subtitle, lines, status, controls)
	}
	if m.discard {
		v.Content = confirmationDialog(v.Content, w, h, "Discard changes?", m.key, "", "Unsaved changes will be lost.", "Discard", m.discardFocused, true)
	}
	return v
}
