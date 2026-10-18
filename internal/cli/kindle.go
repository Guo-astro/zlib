package cli

import (
	"fmt"
	"os"
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
	Short: "Pick a file and send it to Kindle",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStandaloneShellCommand("/kindle")
	},
}

var kindleConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Set up Kindle delivery",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStandaloneShellCommand("/kindle config")
	},
}

var kindleSendCmd = &cobra.Command{
	Use:   "send [file]",
	Short: "Send a file to Kindle",
	Long:  "Send a local file to Kindle. Without a file, open the file picker.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if isInteractive() {
			if len(args) == 0 {
				return runStandaloneShellCommand("/kindle send")
			}
			path, err := expandTilde(args[0])
			if err != nil {
				return err
			}
			return runStandaloneShellCommand("/kindle send " + path)
		}
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
	kindleCmd.AddCommand(kindleConfigCmd, kindleSendCmd)
}

func sendDownloadedFileToKindle(filePath string) error {
	cfg, err := config.LoadKindleConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("kindle config not found; run: zlib kindle config")
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
