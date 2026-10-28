package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// The release update reports in the style of uv: each finished step stays on
// screen as one line with its timing, while the step in progress is a single
// live line (a spinner, or a progress bar during the download) beneath them.

// updateStepLine words a finished step, or returns "" for events that only
// move the live line.
func updateStepLine(e updateEvent) string {
	switch e.kind {
	case updateChecked:
		return "Checked for updates " + colorFaint("in "+formatElapsed(e.elapsed))
	case updateDownloaded:
		return fmt.Sprintf("Downloaded %s %s", e.archive,
			colorFaint(fmt.Sprintf("(%s) in %s", formatBytes(e.written), formatElapsed(e.elapsed))))
	case updateVerified:
		return "Verified checksum " + colorFaint("(sha256)")
	case updateInstalled:
		return "Installed zlib " + e.version + " " + colorFaint("in "+formatElapsed(e.elapsed))
	}
	return ""
}

// printUpdateSummary prints the outcome after all steps.
func printUpdateSummary(out io.Writer, current string, result updateResult, newer bool) {
	switch {
	case result.updated:
		lipgloss.Fprintln(out, colorRed(" - ")+"zlib "+current)
		lipgloss.Fprintln(out, colorGreen(" + ")+"zlib "+result.version)
		lipgloss.Fprintln(out, colorFaint("Restart zlib to use the new version."))
	case newer:
		lipgloss.Fprintf(out, "%s This build (%s) is newer than the latest release (%s)\n", colorFaint(symbolInfo), current, result.version)
	default:
		lipgloss.Fprintf(out, "%s zlib %s is up to date\n", colorGreen(symbolSuccess), current)
	}
}

// runPlainUpdate prints each finished step as a line; it is used when output
// is not a terminal, so there is nothing to animate.
func runPlainUpdate(ctx context.Context, out io.Writer, u releaseUpdater, executable, current, goos, goarch string) (updateResult, error) {
	u.onEvent = func(e updateEvent) {
		if line := updateStepLine(e); line != "" {
			lipgloss.Fprintln(out, line)
		}
	}
	result, err := u.update(ctx, executable, current, goos, goarch)
	if errors.Is(err, context.Canceled) {
		return result, errCancelled
	}
	return result, err
}

// runInteractiveUpdate runs the update under an inline bubbletea program that
// draws the live line. Ctrl+C cancels the update; the updater then removes its
// temporary files before the program exits.
func runInteractiveUpdate(ctx context.Context, in io.Reader, out io.Writer, u releaseUpdater, executable, current, goos, goarch string) (updateResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p := tea.NewProgram(newUpdateProgressModel(cancel), tea.WithInput(in), tea.WithOutput(out))
	u.onEvent = func(e updateEvent) { p.Send(updateEventMsg(e)) }
	go func() {
		result, err := u.update(ctx, executable, current, goos, goarch)
		p.Send(updateFinishedMsg{result: result, err: err})
	}()
	final, err := p.Run()
	if err != nil {
		return updateResult{}, err
	}
	m := final.(updateProgressModel)
	if m.cancelled && !m.result.updated {
		return m.result, errCancelled
	}
	return m.result, m.err
}

type updateEventMsg updateEvent

type updateFinishedMsg struct {
	result updateResult
	err    error
}

// updateLinePrintedMsg follows each step line through the program, so the
// model can wait for every line before it quits.
type updateLinePrintedMsg struct{}

type updateProgressModel struct {
	spinner   spinner.Model
	bar       progress.Model
	activity  string // what the spinner line says
	event     updateEvent
	started   time.Time // download start, for the transfer rate
	width     int
	cancel    context.CancelFunc
	cancelled bool
	done      bool
	result    updateResult
	err       error
	// unprinted counts step lines handed to tea.Println but not yet written.
	// tea.Println runs as a command, so without this count the program could
	// quit before the last step ("Installed") reaches the terminal.
	unprinted int
}

func newUpdateProgressModel(cancel context.CancelFunc) updateProgressModel {
	bar := progress.New(
		progress.WithColors(currentTheme.Accent),
		// The empty part uses a thinner rule so the bar still reads without color.
		progress.WithFillCharacters('━', '─'),
		progress.WithoutPercentage(),
	)
	bar.EmptyColor = currentTheme.Surface
	return updateProgressModel{
		spinner:  newLoadingSpinner(),
		bar:      bar,
		activity: "Checking for updates",
		cancel:   cancel,
	}
}

func (m updateProgressModel) Init() tea.Cmd { return m.spinner.Tick }

func (m updateProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" && !m.cancelled {
			m.cancelled = true
			m.activity = "Cancelling"
			m.cancel()
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(msg)
		return m, cmd
	case updateEventMsg:
		return m.handleEvent(updateEvent(msg))
	case updateFinishedMsg:
		m.done, m.result, m.err = true, msg.result, msg.err
		return m, m.quitWhenPrinted()
	case updateLinePrintedMsg:
		m.unprinted--
		return m, m.quitWhenPrinted()
	}
	return m, nil
}

func (m updateProgressModel) quitWhenPrinted() tea.Cmd {
	if m.done && m.unprinted == 0 {
		return tea.Quit
	}
	return nil
}

func (m updateProgressModel) handleEvent(e updateEvent) (tea.Model, tea.Cmd) {
	if m.cancelled {
		return m, nil
	}
	var cmds []tea.Cmd
	if line := updateStepLine(e); line != "" {
		m.unprinted++
		cmds = append(cmds, tea.Sequence(tea.Println(line), func() tea.Msg { return updateLinePrintedMsg{} }))
	}
	switch e.kind {
	case updateChecked:
		m.activity = "Preparing download"
	case updateDownloadStarted:
		m.event, m.started, m.activity = e, time.Now(), ""
	case updateDownloadProgress:
		m.event = e
		if e.total > 0 {
			cmds = append(cmds, m.bar.SetPercent(float64(e.written)/float64(e.total)))
		}
	case updateDownloaded:
		m.event, m.activity = updateEvent{}, "Verifying checksum"
	case updateVerified:
		m.activity = "Installing"
	}
	return m, tea.Batch(cmds...)
}

func (m updateProgressModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	if m.activity != "" || m.event.archive == "" {
		return tea.NewView(m.spinner.View() + m.activity + "…")
	}
	return tea.NewView(m.downloadLine())
}

// downloadLine is the live transfer line:
//
//	⠋ zlib v0.0.13 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 4.2 MiB/9.6 MiB 3.1 MiB/s
func (m updateProgressModel) downloadLine() string {
	e := m.event
	label := "zlib " + releaseVersionFromArchive(e.archive)
	size := formatBytes(e.written)
	if e.total > 0 {
		size += "/" + formatBytes(e.total)
	}
	stats := colorFaint(size)
	if rate := transferRate(e.written, time.Since(m.started)); rate != "" {
		stats += " " + colorFaint(rate)
	}
	head := m.spinner.View() + label
	if e.total <= 0 {
		return head + " " + stats
	}
	width := m.width
	if width <= 0 {
		width = 80
	}
	m.bar.SetWidth(min(40, max(10, width-ansi.StringWidth(head)-ansi.StringWidth(stats)-2)))
	return head + " " + m.bar.View() + " " + stats
}

// releaseVersionFromArchive returns "v0.0.13" for "zlib_0.0.13_linux_x86_64.tar.gz".
func releaseVersionFromArchive(archive string) string {
	parts := strings.Split(archive, "_")
	if len(parts) < 2 {
		return archive
	}
	return "v" + parts[1]
}

// formatBytes formats n with binary units, as uv and most package managers do.
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value, suffix := float64(n)/unit, "KiB"
	for _, s := range []string{"MiB", "GiB"} {
		if value < unit {
			break
		}
		value, suffix = value/unit, s
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}

// transferRate is the average rate so far, or "" before there is enough data
// for a stable figure.
func transferRate(written int64, elapsed time.Duration) string {
	if written <= 0 || elapsed < 200*time.Millisecond {
		return ""
	}
	return formatBytes(int64(float64(written)/elapsed.Seconds())) + "/s"
}

// formatElapsed prints a step duration the way uv does: 182ms, 2.41s, 1m 05s.
func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.2fs", d.Seconds())
	default:
		return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
}
