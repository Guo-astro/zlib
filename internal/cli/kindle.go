package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/heartleo/zlib"
	"github.com/heartleo/zlib/internal/config"
	"github.com/spf13/cobra"
)

const kindleSMTPPasswordEnv = "ZLIB_SMTP_PWD"
const defaultKindleSMTPHost = "smtp.gmail.com"
const defaultKindleSMTPPort = 587

// Kindle-supported file extensions for the file picker.
var kindleAllowedTypes = []string{".epub", ".pdf", ".txt", ".doc", ".docx", ".html", ".htm", ".rtf", ".jpg", ".jpeg", ".png", ".bmp", ".gif"}

var kindleCmd = &cobra.Command{
	Use:   "kindle",
	Short: "Kindle delivery",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadKindleConfig()
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load kindle config: %w", err)
		}

		if cfg.SMTPHost == "" {
			cfg.SMTPHost = defaultKindleSMTPHost
		}
		if cfg.SMTPPort == 0 {
			cfg.SMTPPort = defaultKindleSMTPPort
		}
		portStr := strconv.Itoa(cfg.SMTPPort)

		form := newFormModel("Kindle settings", "Save", []formField{
			{label: "Kindle address", value: cfg.To, hint: "Your Send to Kindle address", validate: requiredField("Kindle address")},
			{label: "From address", value: cfg.From, hint: "An address approved in your Amazon settings", validate: requiredField("from address")},
			{label: "SMTP host", value: cfg.SMTPHost, hint: "Outgoing mail server", validate: requiredField("SMTP host")},
			{label: "SMTP port", value: portStr, hint: "Usually 587 (STARTTLS) or 465 (TLS)", validate: func(s string) error {
				port, err := strconv.Atoi(strings.TrimSpace(s))
				if err != nil || port < 1 || port > 65535 {
					return fmt.Errorf("enter a port from 1 to 65535")
				}
				return nil
			}},
		})
		form.validate = func(fields []formField) error {
			draft := cfg
			draft.To = fields[0].value
			draft.From = fields[1].value
			draft.SMTPHost = fields[2].value
			draft.SMTPPort, _ = strconv.Atoi(strings.TrimSpace(fields[3].value))
			return draft.Validate()
		}
		fields, accepted, err := runForm(form)
		if err != nil {
			return err
		}
		if !accepted {
			return nil
		}
		cfg.To, cfg.From, cfg.SMTPHost, portStr = fields[0].value, fields[1].value, fields[2].value, fields[3].value
		port, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil || port <= 0 {
			return fmt.Errorf("invalid SMTP port: %s", portStr)
		}
		cfg.SMTPPort = port

		if err := cfg.Validate(); err != nil {
			return err
		}
		if err := config.SaveKindleConfig(cfg); err != nil {
			return fmt.Errorf("failed to save kindle config: %w", err)
		}

		lipgloss.Fprintf(os.Stdout, "%s Saved to: %s\n", colorGreen(symbolSuccess), tildePath(config.KindleConfigPath()))
		lipgloss.Fprintf(os.Stdout, "%s SMTP password via: export %s=...\n", colorFaint(symbolInfo), kindleSMTPPasswordEnv)
		return nil
	},
}

var kindleSendCmd = &cobra.Command{
	Use:   "send [file]",
	Short: "Send a local file to Kindle",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var filePath string
		if len(args) > 0 {
			expanded, err := expandTilde(args[0])
			if err != nil {
				return err
			}
			filePath = expanded
		} else {
			selected, accepted, err := chooseKindleFile()
			if err != nil {
				return err
			}
			if !accepted {
				return nil
			}
			filePath = selected
		}
		return sendDownloadedFileToKindle(filePath)
	},
}

func init() {
	kindleCmd.AddCommand(kindleSendCmd)
}

func sendDownloadedFileToKindle(filePath string) error {
	cfg, err := config.LoadKindleConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("kindle config not found; run: zlib kindle")
		}
		return fmt.Errorf("failed to load kindle config: %w", err)
	}

	password := os.Getenv(kindleSMTPPasswordEnv)
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("kindle smtp password not set; export %s", kindleSMTPPasswordEnv)
	}

	// Run the spinner UI on a terminal; send directly otherwise so the
	// command still works (and exits) when driven by a script or pipe.
	if isInteractive() {
		result, err := tea.NewProgram(newKindleSendModel(filePath, cfg, password)).Run()
		if err != nil {
			return err
		}
		if sm := result.(kindleSendModel); sm.err != nil {
			return sm.err
		}
	} else {
		lipgloss.Fprintln(os.Stdout, "  Sending to Kindle...")
		if err := zlib.SendToKindle(filePath, cfg, password); err != nil {
			return err
		}
	}

	lipgloss.Fprintln(os.Stdout)
	lipgloss.Fprintf(os.Stdout, "%s Email sent to Kindle address: %s\n", colorGreen(symbolSuccess), colorCyan(cfg.To))
	lipgloss.Fprintf(os.Stdout, "%s Final delivery depends on Amazon approval and file compatibility.\n", colorFaint(symbolInfo))
	return nil
}

// — kindle send model (spinner during SMTP send) —

type kindleSendModel struct {
	width, height int
	filePath      string
	cfg           zlib.KindleConfig
	password      string
	spinner       spinner.Model
	err           error
	done          bool
}

type kindleSendDoneMsg struct{}

func newKindleSendModel(filePath string, cfg zlib.KindleConfig, password string) kindleSendModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(currentTheme.Accent)
	return kindleSendModel{
		filePath: filePath,
		cfg:      cfg,
		password: password,
		spinner:  s,
	}
}

func (m kindleSendModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.sendCmd())
}

func (m kindleSendModel) sendCmd() tea.Cmd {
	return func() tea.Msg {
		if err := zlib.SendToKindle(m.filePath, m.cfg, m.password); err != nil {
			return errMsg{err}
		}
		return kindleSendDoneMsg{}
	}
}

func (m kindleSendModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, tea.Quit
	case kindleSendDoneMsg:
		m.done = true
		return m, tea.Quit
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("stopped waiting; email delivery may still complete")
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m kindleSendModel) View() tea.View {
	return screenView(m.width, m.height, "Send to Kindle", bookCell(m.filePath), m.viewContent(), "To: "+bookCell(m.cfg.To), uiKey("ctrl+c", "stop waiting"))
}

func (m kindleSendModel) viewContent() string {
	return fmt.Sprintf("  %s Sending to Kindle...\n", m.spinner.View())
}
