package cli

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	keys "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
	"github.com/heartleo/zlib/internal/config"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive shell",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !isInteractive() {
			return fmt.Errorf("zlib tui requires an interactive terminal")
		}
		return runShell()
	},
}

type shellEvent struct {
	generation int
	msg        tea.Msg
}
type shellText struct {
	text string
	err  error
}
type shellModel struct {
	input                                                           textinput.Model
	page                                                            tea.Model
	previous                                                        tea.Model
	kind, status, pending, paletteDraft, infoTitle                  string
	width, height, generation, selection, paletteOffset, infoOffset int
	typing                                                          bool
	signedIn                                                        bool
	download                                                        downloadModel
	kindlePath                                                      string
	loginForm                                                       *inlineForm
}

type shellCommand struct{ name, description string }

var shellCommands = []shellCommand{
	{"search", "Search books by title, author, or ISBN"},
	{"favorites", "Show saved books"},
	{"recommended", "Show recommended books"},
	{"most-popular", "Show popular books"},
	{"history", "Show download history"},
	{"profile", "Show profile and quota"},
	{"config", "Edit network, download, and Kindle settings"},
	{"theme", "Choose a theme"},
	{"doctor", "Find available domains"},
	{"login", "Log in with email and password"},
	{"logout", "Remove saved session"},
	{"kindle", "Send a file to Kindle"},
	{"quit", "Exit zlib"},
}

func newShellModel() shellModel {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "Search books or type / for commands"
	input.Focus()
	session, _ := config.LoadSession()
	return shellModel{input: input, width: 80, height: 24, typing: true, signedIn: len(session.Cookies) > 0}
}
func runShell() error              { _, err := tea.NewProgram(newShellModel()).Run(); return err }
func (m shellModel) Init() tea.Cmd { return textinput.Blink }

// Adapt existing standalone pages without allowing their QuitMsg to end the shell.
// Every async response is scoped to the page instance that started it.
func shellCmd(generation int, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			wrapped := make([]tea.Cmd, len(batch))
			for i, c := range batch {
				wrapped[i] = shellCmd(generation, c)
			}
			return tea.Batch(wrapped...)()
		}
		return shellEvent{generation, msg}
	}
}
func (m *shellModel) open(kind string, page tea.Model) tea.Cmd {
	if m.kind == "list" && m.page != nil && m.previous == nil {
		m.previous = m.page
	}
	if kind == "list" {
		if list, ok := page.(searchSelectModel); ok {
			list.inShell = true
			page = list
		} else if list, ok := page.(historySelectModel); ok {
			list.inShell = true
			page = list
		}
	}
	m.generation++
	m.kind = kind
	m.page = page
	m.typing = false
	m.input.Blur()
	m.status = ""
	page, _ = page.Update(tea.WindowSizeMsg{Width: m.width, Height: m.contentHeight()})
	m.page = page
	return shellCmd(m.generation, page.Init())
}
func (m *shellModel) home() {
	m.generation++
	m.page = nil
	m.kind = ""
	m.infoTitle = ""
	m.infoOffset = 0
	m.typing = true
	m.paletteOffset = 0
	m.input.Focus()
}

func (m *shellModel) openInfo(title, content string) {
	if m.kind == "list" && m.page != nil {
		m.previous = m.page
	}
	m.home()
	m.kind = "info"
	m.typing = false
	m.input.Blur()
	m.infoTitle = title
	m.status = content
}
func (m *shellModel) restore() {
	old := m.previous
	m.previous = nil
	if old == nil {
		m.home()
		return
	}
	m.generation++
	m.page = old
	m.kind = "list"
	m.typing = false
	m.resizePage()
}

func (m shellModel) contentHeight() int {
	return max(1, m.height-m.footerHeight())
}

func (m shellModel) footerHeight() int {
	if m.typing {
		return 2
	}
	if m.page != nil && m.kind != "list" {
		return 0
	}
	return 1
}

func (m *shellModel) resizePage() {
	if m.page != nil {
		m.page, _ = m.page.Update(tea.WindowSizeMsg{Width: m.width, Height: m.contentHeight()})
	}
}

func (m shellModel) matches() []shellCommand {
	value, ok := strings.CutPrefix(m.input.Value(), "/")
	if !ok {
		return nil
	}
	if strings.Contains(value, " ") {
		return nil
	}
	var out []shellCommand
	for _, command := range shellCommands {
		if strings.HasPrefix(command.name, value) {
			out = append(out, command)
		}
	}
	if value != "" {
		for _, command := range shellCommands {
			if !strings.HasPrefix(command.name, value) && strings.Contains(command.name, value) {
				out = append(out, command)
			}
		}
	}
	return out
}
func (m shellModel) childInputActive() bool {
	switch page := m.page.(type) {
	case searchSelectModel:
		return page.state == searchStateFilter || page.confirmDownload
	case historySelectModel:
		return page.confirmDownload || page.state == historyStateConfirmDelete
	default:
		return false
	}
}
func (m shellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = size.Width, size.Height
		m.input.SetWidth(max(1, m.width-4))
		m.resizePage()
		m.paletteOffset = min(m.paletteOffset, max(0, len(m.matches())-m.paletteVisibleRows(m.contentHeight())))
		return m, nil
	}
	if event, ok := msg.(shellEvent); ok {
		if event.generation != m.generation {
			return m, nil
		}
		if _, ok := event.msg.(tea.QuitMsg); ok {
			cmd := m.completed()
			return m, cmd
		}
		if result, ok := event.msg.(shellText); ok {
			loginFinished := m.kind == "login-wait"
			if loginFinished {
				if result.err != nil {
					if m.loginForm != nil {
						form := *m.loginForm
						m.loginForm = nil
						form.accepted = false
						form.message = result.err.Error()
						form.fields[1].value = ""
						form.inputs[1].SetValue("")
						form.focus(1)
						return m, m.open("login", form)
					}
					m.home()
					m.status = result.err.Error()
					return m, nil
				}
				m.loginForm = nil
				m.signedIn = true
				m.previous = nil
				pending := m.pending
				m.pending = ""
				m.home()
				m.status = ""
				if pending != "" {
					m.input.SetValue(pending)
					m.input.CursorEnd()
					m.status = "Press Enter to continue."
				}
				return m, nil
			}
			if result.err != nil {
				m.status = result.err.Error()
			} else {
				m.status = result.text
			}
			m.kind = "info"
			return m, nil
		}
		if m.page != nil {
			var cmd tea.Cmd
			m.page, cmd = m.page.Update(event.msg)
			return m, shellCmd(m.generation, cmd)
		}
		return m, nil
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if m.typing {
			switch k.String() {
			case "esc":
				if m.paletteDraft != "" {
					m.input.SetValue(m.paletteDraft)
					m.input.CursorEnd()
					m.paletteDraft = ""
					m.selection = 0
					m.paletteOffset = 0
					return m, nil
				}
				m.input.SetValue("")
				m.selection = 0
				m.paletteOffset = 0
				if m.page != nil {
					m.typing = false
					m.input.Blur()
					m.resizePage()
				} else if m.previous != nil {
					m.restore()
				}
				return m, nil
			case "ctrl+c":
				if m.input.Value() != "" {
					m.input.SetValue("")
					return m, nil
				}
				if m.page == nil {
					return m, tea.Quit
				}
				m.typing = false
				m.resizePage()
				return m, nil
			case "ctrl+k":
				if !strings.HasPrefix(m.input.Value(), "/") {
					m.paletteDraft = m.input.Value()
				}
				m.input.SetValue("/")
				m.selection = 0
				m.paletteOffset = 0
				m.status = ""
				return m, nil
			case "up", "down":
				items := m.matches()
				if len(items) > 0 {
					m.selection = min(max(0, m.selection), len(items)-1)
					if k.String() == "up" {
						m.selection = (m.selection - 1 + len(items)) % len(items)
					} else {
						m.selection = (m.selection + 1) % len(items)
					}
					m.ensurePaletteVisible(len(items))
				}
				return m, nil
			case "tab":
				items := m.matches()
				if len(items) > 0 {
					m.input.SetValue("/" + items[min(m.selection, len(items)-1)].name + " ")
					m.input.CursorEnd()
				}
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.input.Value())
				if value == "" && !m.signedIn && m.page == nil && m.kind == "" {
					return m, m.dispatch("/login")
				}
				if strings.HasPrefix(value, "/") && !strings.Contains(value, " ") {
					items := m.matches()
					if len(items) > 0 && !slices.ContainsFunc(shellCommands, func(command shellCommand) bool { return command.name == strings.TrimPrefix(value, "/") }) {
						value = "/" + items[min(m.selection, len(items)-1)].name
					}
				}
				cmd := m.dispatch(value)
				return m, cmd
			}
		} else if m.kind == "info" {
			if k.String() == "esc" || k.String() == "q" {
				m.restore()
				m.status = ""
				return m, nil
			}
			lines := strings.Split(m.status, "\n")
			visible := max(1, m.height-6)
			switch k.String() {
			case "up":
				m.infoOffset--
			case "down":
				m.infoOffset++
			case "pgup":
				m.infoOffset -= visible
			case "pgdown":
				m.infoOffset += visible
			case "home":
				m.infoOffset = 0
			case "end":
				m.infoOffset = len(lines)
			}
			m.infoOffset = min(max(0, m.infoOffset), max(0, len(lines)-visible))
		}
		if !m.typing && (m.kind == "list" || m.kind == "info" || m.page == nil) && !m.childInputActive() {
			if k.String() == "/" || k.String() == "ctrl+k" {
				m.typing = true
				m.input.SetValue("/")
				m.selection = 0
				m.status = ""
				m.resizePage()
				return m, m.input.Focus()
			}
			if k.String() == "ctrl+f" {
				if list, ok := m.page.(searchSelectModel); ok && list.feed == nil && list.feedPage == nil {
					msg = tea.KeyPressMsg{Code: '/', Text: "/"}
				}
			}
		}
	}
	if m.typing {
		var cmd tea.Cmd
		before := m.input.Value()
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != before {
			m.selection = 0
			m.paletteOffset = 0
			m.status = ""
		}
		return m, cmd
	}
	if m.page != nil {
		var cmd tea.Cmd
		m.page, cmd = m.page.Update(msg)
		return m, shellCmd(m.generation, cmd)
	}
	return m, nil
}

func (m *shellModel) dispatch(value string) tea.Cmd {
	if value == "" {
		return nil
	}
	name, arg := "search", value
	if strings.HasPrefix(value, "/") {
		name, arg, _ = strings.Cut(strings.TrimPrefix(value, "/"), " ")
		arg = strings.TrimSpace(arg)
	}
	known := false
	for _, command := range shellCommands {
		if command.name == name {
			known = true
		}
	}
	if !known {
		m.status = "Unknown command: /" + name
		return nil
	}
	if name == "quit" {
		return tea.Quit
	}
	m.input.SetValue("")
	m.selection = 0
	m.paletteOffset = 0
	m.paletteDraft = ""
	switch name {
	case "login":
		m.loginForm = nil
		form := newInlineForm("Log in", "Log in", []formField{{label: "Email", value: rememberedLoginEmail()}, {label: "Password", secret: true}})
		form.validate = func(fields []formField) error {
			if err := requiredField("email")(fields[0].value); err != nil {
				return err
			}
			return requiredField("password")(fields[1].value)
		}
		return m.open("login", form)
	case "logout":
		if !m.signedIn {
			removed, err := clearSavedSession()
			if err != nil {
				m.status = "Could not log out: " + err.Error()
			} else if removed {
				m.status = "Logged out"
			} else {
				m.status = "Not logged in"
			}
			return nil
		}
		return m.open("logout", shellConfirm{title: "Log out?", detail: "Clear the saved session", action: "Log out"})
	case "theme":
		return m.open("theme", newThemePicker(configuredThemeName(), themeOverride()))
	case "config":
		store, err := newSettingsStore()
		if err != nil {
			m.status = err.Error()
			return nil
		}
		page := arg
		if page == "" {
			page = "home"
		}
		if page != "home" && page != "network" && page != "download" && page != "kindle" {
			m.status = "Use /config network, /config download, or /config kindle"
			return nil
		}
		model, err := newSettingsModel(context.Background(), store, page)
		if err != nil {
			m.status = err.Error()
			return nil
		}
		return m.open("config", model)
	case "doctor":
		m.openInfo("Available domains", "Checking domains…")
		return shellCmd(m.generation, func() tea.Msg {
			checks := checkAvailableDomains(context.Background(), zlib.KnownDomains(), os.Getenv(zlib.EnvProxy))
			var rows []string
			for _, c := range checks {
				rows = append(rows, c.Domain+"  Available")
			}
			if len(rows) == 0 {
				rows = append(rows, "No available domains")
			}
			return shellText{text: strings.Join(rows, "\n")}
		})
	case "kindle":
		return m.openKindle(arg)
	}
	c, err := loadClient()
	if err != nil {
		m.pending = value
		m.home()
		if m.signedIn {
			m.status = err.Error()
		} else {
			m.status = "Sign in to continue. Press Enter to sign in."
		}
		return nil
	}
	switch name {
	case "search":
		if arg == "" {
			return m.open("search-form", newInlineForm("Search books", "Search", []formField{
				{label: "Query", hint: "Title, author, or ISBN", validate: requiredField("query")},
				{label: "Formats", options: []string{"EPUB", "AZW", "AZW3", "MOBI", "PDF", "RTF", "TXT"}, hint: "No selection includes all formats"},
			}))
		}
		return m.open("list", newSearchSelectModel(arg, c, nil, false))
	case "favorites":
		return m.open("list", newSavedBooksSelectModel(c.SavedBooks).withBookDetails(c))
	case "recommended":
		return m.open("list", newBookFeedSelectModel("Recommended", c.RecommendedBooks).withBookDetails(c))
	case "most-popular":
		return m.open("list", newBookFeedSelectModel("Most popular", c.MostPopularBooks).withBookDetails(c))
	case "history":
		return m.open("list", newHistorySelectModel(c))
	case "profile":
		m.openInfo("Profile", "Loading profile…")
		return shellCmd(m.generation, func() tea.Msg { p, err := c.GetProfile(); return shellText{text: renderProfileCard(p), err: err} })
	}
	return nil
}

func (m *shellModel) chooseDirectory() tea.Cmd {
	dir := os.Getenv(downloadDirEnvVar)
	if dir == "" {
		dir = "."
	}
	return m.open("directory", newDownloadDirectoryModel(dir))
}
func (m *shellModel) completed() tea.Cmd {
	switch page := m.page.(type) {
	case searchSelectModel:
		if page.selectedID != "" {
			id := page.selectedID
			page.selectedID = ""
			page.confirmDownload = false
			m.previous = page
			client := page.client
			if client == nil {
				var err error
				client, err = loadClient()
				if err != nil {
					m.status = err.Error()
					m.restore()
					return nil
				}
			}
			m.download = newDownloadModel(id, "", client)
			return m.chooseDirectory()
		}
	case historySelectModel:
		if page.selected != nil {
			item := *page.selected
			page.selected = nil
			page.confirmDownload = false
			m.previous = page
			m.download = newDownloadModel(historyCommandID(item), "", page.client)
			return m.chooseDirectory()
		}
	case downloadDirectoryModel:
		if page.accepted {
			m.download.dir = page.dir
			return m.open("download", m.download)
		}
		m.restore()
		return nil
	case downloadModel:
		m.restore()
		if page.err != nil {
			m.status = page.err.Error()
		} else if page.state == stateCancelled {
			m.status = "Download cancelled"
		} else {
			m.status = "Saved to: " + page.savedPath
		}
		return nil
	case themePickerModel:
		if page.accepted {
			cfg, _ := config.LoadConfig()
			cfg.Theme = page.selected()
			if err := config.SaveConfig(cfg); err != nil {
				m.status = err.Error()
			} else {
				m.status = "Theme saved: " + cfg.Theme
			}
			resolveTheme()
		}
	case inlineForm:
		if !page.accepted {
			m.restore()
			return nil
		}
		switch m.kind {
		case "search-form":
			c, err := loadClient()
			if err != nil {
				m.pending = "/search " + page.fields[0].value
				m.home()
				if m.signedIn {
					m.status = err.Error()
				} else {
					m.status = "Sign in to continue. Press Enter to sign in."
				}
				return nil
			}
			opts := &zlib.SearchOptions{}
			for _, ext := range page.fields[1].selected {
				opts.Extensions = append(opts.Extensions, zlib.Extension(ext))
			}
			return m.open("list", newSearchSelectModel(page.fields[0].value, c, opts, false))
		case "login":
			fields := page.fields
			m.loginForm = new(page)
			m.home()
			m.kind = "login-wait"
			m.typing = false
			m.status = "Logging in…"
			return shellCmd(m.generation, func() tea.Msg { return shellLogin(fields) })
		case "kindle-config":
			return m.saveKindle(page.fields)
		}
	case kindleFileModel:
		if page.accepted {
			return m.sendKindle(page.path)
		}
	case kindleSendModel:
		m.restore()
		if page.err != nil {
			m.status = page.err.Error()
		} else {
			m.status = "Kindle send finished"
		}
		return nil
	case shellConfirm:
		if page.confirmed {
			switch m.kind {
			case "logout":
				_, err := clearSavedSession()
				if err != nil {
					m.restore()
					m.status = "Could not log out: " + err.Error()
					return nil
				}
				m.signedIn = false
				m.previous = nil
				m.pending = ""
				m.home()
				m.status = ""
				return nil
			case "kindle-confirm":
				return m.sendKindle(m.kindlePath)
			}
		}
	}
	m.restore()
	return nil
}

func shellLogin(fields []formField) shellText {
	domain, err := resolveLoginDomain("")
	if err != nil {
		return shellText{err: err}
	}
	domain, err = resolveEAPIDomain(context.Background(), domain)
	if err != nil {
		return shellText{err: err}
	}
	c := zlib.NewClient()
	c.SetDomain(domain)
	if err = c.LoginEAPI(fields[0].value, fields[1].value); err != nil {
		return shellText{err: err}
	}
	err = config.SaveSession(config.Session{Cookies: c.Cookies(), Domain: c.Domain(), Mode: string(zlib.ClientModeEAPI)})
	if err == nil {
		_ = rememberLoginEmail(fields[0].value)
	}
	return shellText{text: "Logged in", err: err}
}

func (m shellModel) View() tea.View {
	if m.kind == "info" && !m.typing {
		lines := strings.Split(m.status, "\n")
		visible := max(1, m.height-6)
		start := min(max(0, m.infoOffset), max(0, len(lines)-visible))
		end := min(len(lines), start+visible)
		status := ""
		if len(lines) > visible {
			status = fmt.Sprintf("%d–%d of %d", start+1, end, len(lines))
		}
		return screenView(m.width, m.height, m.infoTitle, "", strings.Join(lines[start:end], "\n"), status,
			uiKey("↑/↓", "scroll"), uiKey("/", "commands"), uiKey("esc", "back"))
	}
	available := m.contentHeight()
	palette := m.paletteLines(available)
	contentRows := max(0, available-len(palette))
	var body string
	if m.page != nil {
		body = m.page.View().Content
		body = strings.ReplaceAll(body, "/ search", "Ctrl+F search")
	} else if m.kind == "info" || m.kind == "login-wait" {
		body = "zlib\n\n" + m.status
	} else {
		body = m.homeView(contentRows)
	}
	lines := strings.Split(body, "\n")
	if len(lines) > contentRows {
		lines = lines[:contentRows]
	}
	for len(lines) < contentRows {
		lines = append(lines, "")
	}
	lines = append(lines, palette...)
	if m.typing {
		lines = append(lines, m.input.View())
	}
	if m.footerHeight() > 0 {
		lines = append(lines, m.footerView())
	}
	if len(lines) > m.height {
		lines = lines[:max(1, m.height)]
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], max(1, m.width), "…")
	}
	return fullScreenView(m.width, m.height, strings.Join(lines, "\n"))
}

func (m shellModel) paletteLines(available int) []string {
	if !m.typing || !strings.HasPrefix(m.input.Value(), "/") || strings.Contains(m.input.Value(), " ") {
		return nil
	}
	items := m.matches()
	muted := lipgloss.NewStyle().Foreground(currentTheme.Muted)
	if len(items) == 0 {
		return []string{muted.Render("  No matching commands")}
	}
	if available <= 1 {
		return []string{styledListRow("/"+items[min(m.selection, len(items)-1)].name, true, m.width, currentTheme)}
	}
	count := min(len(items), m.paletteVisibleRows(available))
	selected := min(max(0, m.selection), len(items)-1)
	start := min(max(0, m.paletteOffset), len(items)-count)
	if selected < start {
		start = selected
	} else if selected >= start+count {
		start = selected - count + 1
	}
	end := min(len(items), start+count)
	lines := make([]string, 0, count)
	nameWidth := 0
	for _, item := range items[start:end] {
		nameWidth = max(nameWidth, len(item.name)+1)
	}
	nameWidth = min(nameWidth, max(1, m.width/2))
	for i := start; i < end; i++ {
		item := items[i]
		name := "/" + item.name
		row := name
		if m.width >= 32 {
			row += strings.Repeat(" ", max(1, nameWidth-len(name)+2)) + item.description
		}
		lines = append(lines, styledListRow(row, i == selected, m.width, currentTheme))
	}
	return lines
}

func (m shellModel) paletteVisibleRows(available int) int {
	return min(6, max(1, available))
}

func (m *shellModel) ensurePaletteVisible(count int) {
	rows := min(count, m.paletteVisibleRows(m.contentHeight()))
	m.paletteOffset = min(max(0, m.paletteOffset), max(0, count-rows))
	if m.selection < m.paletteOffset {
		m.paletteOffset = m.selection
	} else if m.selection >= m.paletteOffset+rows {
		m.paletteOffset = m.selection - rows + 1
	}
}

func (m shellModel) footerView() string {
	if m.status != "" && m.kind != "info" && m.kind != "login-wait" {
		return lipgloss.NewStyle().Foreground(currentTheme.Accent).Render(m.status)
	}
	h := help.New()
	h.SetWidth(max(1, m.width))
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(currentTheme.Muted)
	bindings := []keys.Binding{uiKey("esc", "back")}
	switch {
	case m.typing && strings.HasPrefix(m.input.Value(), "/"):
		bindings = []keys.Binding{uiKey("↑/↓", "cycle"), uiKey("tab", "complete"), uiKey("enter", "run"), uiKey("esc", "close")}
	case m.typing:
		if !m.signedIn && m.input.Value() == "" && m.page == nil && m.kind == "" {
			bindings = []keys.Binding{uiKey("enter", "sign in"), uiKey("/", "commands"), uiKey("esc", "clear")}
		} else {
			bindings = []keys.Binding{uiKey("enter", "search"), uiKey("/", "commands"), uiKey("esc", "clear")}
		}
	case m.kind == "list":
		bindings = []keys.Binding{uiKey("/", "commands"), uiKey("ctrl+f", "refine"), uiKey("esc", "back")}
	case m.kind == "info" || m.page == nil && m.kind == "":
		bindings = []keys.Binding{uiKey("/", "commands"), uiKey("esc", "back")}
	}
	return h.ShortHelpView(bindings)
}
