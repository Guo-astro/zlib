package cli

import (
	"context"
	"fmt"
	"os"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/heartleo/zlib"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <book-id>",
	Short: "Download a book by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := destDirFlag(cmd)
		if err != nil {
			return err
		}
		sendToKindle, _ := cmd.Flags().GetBool("send-to-kindle")
		c := newClient()
		return runDownload(newDownloadModel(args[0], dir, c), sendToKindle, cmd)
	},
}

// runDownload executes the download described by m — through the bubbletea
// progress UI on an interactive terminal, or a plain text path otherwise (see
// isInteractive) — then prints the saved path and optionally delivers to
// Kindle. Shared by the download, search, and history commands.
func runDownload(m downloadModel, sendToKindle bool, cmd *cobra.Command) error {
	dir, proceed, err := chooseDownloadDirectory(cmd, m.dir)
	if err != nil {
		return err
	}
	if !proceed {
		return nil
	}
	m.dir = dir
	dm, err := runDownloadModel(m)
	if err != nil {
		return err
	}
	if dm.state == stateCancelled {
		return nil
	}
	if dm.err != nil {
		return dm.err
	}

	lipgloss.Fprintln(os.Stdout)
	lipgloss.Fprintf(os.Stdout, "%s Saved to: %s (%d bytes)\n",
		colorGreen(symbolSuccess),
		colorGreen(tildePath(dm.savedPath)),
		dm.savedSize,
	)
	if sendToKindle {
		if err := sendDownloadedFileToKindle(dm.savedPath); err != nil {
			return fmt.Errorf("send to kindle failed: %w", err)
		}
	}
	return nil
}

// runDownloadModel runs the download, choosing the bubbletea UI or the plain
// path based on whether the process is attached to a terminal.
func runDownloadModel(m downloadModel) (downloadModel, error) {
	if !isInteractive() {
		return m.runPlain()
	}
	result, err := tea.NewProgram(m).Run()
	if err != nil {
		return downloadModel{}, err
	}
	return result.(downloadModel), nil
}

// runPlain performs the download without the bubbletea UI, for non-terminal
// output. It mirrors the fetch-then-download flow of the UI model and records
// the result on the returned model.
func (m downloadModel) runPlain() (downloadModel, error) {
	downloadURL, name := m.directURL, m.bookName
	if downloadURL == "" {
		book, err := m.client.FetchBook(m.id)
		if err != nil {
			return m, fmt.Errorf("failed to fetch book: %w", err)
		}
		if book.DownloadURL == "" {
			return m, fmt.Errorf("no download URL available for book %s", m.id)
		}
		downloadURL, name = book.DownloadURL, book.Name
	}

	lipgloss.Fprintf(os.Stdout, "  Downloading: %s\n", name)
	result, err := m.client.Download(downloadURL, m.dir, nil)
	if err != nil {
		return m, err
	}
	m.savedPath = result.FilePath
	m.savedSize = result.Size
	m.state = stateComplete
	return m, nil
}

func init() {
	addDestDirFlag(downloadCmd)
	downloadCmd.Flags().Bool("send-to-kindle", false, "Send the downloaded file to Kindle.")
}

// — bubbletea model —

type downloadState int

const (
	stateFetching downloadState = iota
	stateDownloading
	stateComplete
	stateCancelled
)

type downloadModel struct {
	width, height int
	id            string
	dir           string
	client        *zlib.Client
	state         downloadState
	bookName      string
	directURL     string // set to skip fetch and download immediately
	spinner       spinner.Model
	progress      progress.Model
	written       int64
	total         int64
	savedPath     string
	savedSize     int64
	err           error
	downloadCh    chan tea.Msg
	cancel        context.CancelFunc
}

type fetchDoneMsg struct{ book zlib.Book }
type progressMsg struct{ written, total int64 }
type downloadDoneMsg struct {
	path string
	size int64
}
type errMsg struct{ err error }

func newDownloadModel(id, dir string, c *zlib.Client) downloadModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(currentTheme.Accent)

	p := progress.New(
		progress.WithDefaultBlend(),
		progress.WithWidth(40),
	)

	return downloadModel{
		id:       id,
		dir:      dir,
		client:   c,
		spinner:  s,
		progress: p,
	}
}

// newDirectDownloadModel skips the fetch phase and downloads from a known URL.
func newDirectDownloadModel(bookName, downloadURL, dir string, c *zlib.Client) downloadModel {
	m := newDownloadModel("", dir, c)
	m.bookName = bookName
	m.directURL = downloadURL
	return m
}

func (m downloadModel) Init() tea.Cmd {
	if m.directURL != "" {
		// Emit a synthetic fetchDoneMsg to skip the HTTP fetch.
		book := zlib.Book{DownloadURL: m.directURL, Name: m.bookName}
		return func() tea.Msg { return fetchDoneMsg{book} }
	}
	return tea.Batch(m.spinner.Tick, m.fetchCmd())
}

func (m downloadModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		book, err := m.client.FetchBook(m.id)
		if err != nil {
			return errMsg{fmt.Errorf("failed to fetch book: %w", err)}
		}
		if book.DownloadURL == "" {
			return errMsg{fmt.Errorf("no download URL available for book %s", m.id)}
		}
		return fetchDoneMsg{book}
	}
}

// waitForMsg returns a Cmd that reads the next message from the download channel.
func waitForMsg(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m downloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.progress.SetWidth(max(1, min(60, msg.Width-8)))
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, tea.Quit

	case fetchDoneMsg:
		m.state = stateDownloading
		m.bookName = msg.book.Name
		ch := make(chan tea.Msg, 64)
		m.downloadCh = ch
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		go func() {
			emit := func(msg tea.Msg) {
				select {
				case ch <- msg:
				case <-ctx.Done():
				}
			}
			result, err := m.client.DownloadWithContext(ctx, msg.book.DownloadURL, m.dir, func(w, t int64) {
				emit(progressMsg{w, t})
			})
			if err != nil {
				emit(errMsg{err})
			} else {
				emit(downloadDoneMsg{result.FilePath, result.Size})
			}
			close(ch)
		}()
		return m, waitForMsg(ch)

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			if m.cancel != nil {
				m.cancel()
			}
			m.state = stateCancelled
			return m, tea.Quit
		}

	case progressMsg:
		m.written = msg.written
		m.total = msg.total
		var pct float64
		if m.total > 0 {
			pct = float64(m.written) / float64(m.total)
		}
		progressCmd := m.progress.SetPercent(pct)
		return m, tea.Batch(progressCmd, waitForMsg(m.downloadCh))

	case downloadDoneMsg:
		m.savedPath = msg.path
		m.savedSize = msg.size
		m.state = stateComplete
		return m, tea.Quit

	case progress.FrameMsg:
		pm, cmd := m.progress.Update(msg)
		m.progress = pm
		return m, cmd

	case spinner.TickMsg:
		if m.state == stateFetching {
			sm, cmd := m.spinner.Update(msg)
			m.spinner = sm
			return m, cmd
		}
	}
	return m, nil
}

func (m downloadModel) View() tea.View {
	return screenView(m.width, m.height, "Download", bookCell(m.bookName), m.viewContent(), "Directory: "+bookCell(m.dir), uiKey("esc", "cancel"))
}

func (m downloadModel) viewContent() string {
	switch m.state {
	case stateFetching:
		return fmt.Sprintf("  %s Fetching book %s...\n",
			m.spinner.View(), colorCyan(m.id))
	case stateDownloading, stateComplete, stateCancelled:
		writtenMB := float64(m.written) / 1024 / 1024
		totalMB := float64(m.total) / 1024 / 1024
		var sizeInfo string
		if m.total > 0 {
			sizeInfo = fmt.Sprintf("  %.1f MB / %.1f MB", writtenMB, totalMB)
		} else {
			sizeInfo = fmt.Sprintf("  %.1f MB", writtenMB)
		}
		bar := m.progress.View()
		if m.state == stateComplete {
			bar = m.progress.ViewAs(1.0)
		}
		out := fmt.Sprintf("  Downloading: %s\n  %s%s\n",
			m.bookName, bar, colorFaint(sizeInfo))
		if m.state == stateCancelled {
			out += fmt.Sprintf("\n  %s %s\n",
				colorRed(symbolError),
				colorRed("Download cancelled (incomplete file removed)"))
		}
		return out
	}
	return ""
}
