package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestFormatBytes(t *testing.T) {
	for n, want := range map[int64]string{
		0:                      "0 B",
		1023:                   "1023 B",
		1024:                   "1.0 KiB",
		1536:                   "1.5 KiB",
		10 * 1024 * 1024:       "10.0 MiB",
		3 * 1024 * 1024 * 1024: "3.0 GiB",
	} {
		if got := formatBytes(n); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestFormatElapsed(t *testing.T) {
	for d, want := range map[time.Duration]string{
		182 * time.Millisecond:  "182ms",
		2410 * time.Millisecond: "2.41s",
		65 * time.Second:        "1m 05s",
	} {
		if got := formatElapsed(d); got != want {
			t.Errorf("formatElapsed(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestTransferRateWaitsForStableSample(t *testing.T) {
	if got := transferRate(1<<20, 100*time.Millisecond); got != "" {
		t.Errorf("rate after 100ms = %q, want none", got)
	}
	if got := transferRate(2<<20, time.Second); got != "2.0 MiB/s" {
		t.Errorf("rate = %q, want 2.0 MiB/s", got)
	}
}

func TestReleaseVersionFromArchive(t *testing.T) {
	if got := releaseVersionFromArchive("zlib_0.0.13_linux_x86_64.tar.gz"); got != "v0.0.13" {
		t.Errorf("version = %q", got)
	}
}

// recordUpdate runs a release update against a fixture server and returns
// the events it emitted, in order.
func recordUpdate(t *testing.T, current string, badChecksum bool) ([]updateEvent, updateResult, error) {
	t.Helper()
	archive := "zlib_0.0.8_linux_x86_64.tar.gz"
	payload := testReleaseArchive(t, "linux", "zlib", bytes.Repeat([]byte("x"), 256<<10))
	server := testReleaseServer(t, archive, payload, badChecksum)
	t.Cleanup(server.Close)
	executable := filepath.Join(t.TempDir(), "zlib")
	if err := os.WriteFile(executable, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	var events []updateEvent
	updater := releaseUpdater{client: server.Client(), apiURL: server.URL + "/latest", onEvent: func(e updateEvent) { events = append(events, e) }}
	result, err := updater.update(t.Context(), executable, current, "linux", "amd64")
	return events, result, err
}

func eventKinds(events []updateEvent) []updateEventKind {
	var kinds []updateEventKind
	for _, e := range events {
		if e.kind != updateDownloadProgress && (len(kinds) == 0 || kinds[len(kinds)-1] != e.kind) {
			kinds = append(kinds, e.kind)
		}
	}
	return kinds
}

func TestReleaseUpdateReportsEveryStep(t *testing.T) {
	events, result, err := recordUpdate(t, "v0.0.7", false)
	if err != nil || !result.updated {
		t.Fatalf("update = %+v, %v", result, err)
	}
	want := []updateEventKind{updateChecked, updateDownloadStarted, updateDownloaded, updateVerified, updateInstalled}
	if got := eventKinds(events); !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	var downloaded updateEvent
	for _, e := range events {
		if e.kind == updateDownloaded {
			downloaded = e
		}
		if e.kind == updateDownloadProgress && (e.written <= 0 || (e.total > 0 && e.written > e.total)) {
			t.Errorf("progress event out of range: %+v", e)
		}
	}
	if downloaded.written <= 0 || downloaded.written != downloaded.total {
		t.Errorf("downloaded = %+v, want written equal to total", downloaded)
	}
	if events[0].version != "v0.0.8" || downloaded.archive != "zlib_0.0.8_linux_x86_64.tar.gz" {
		t.Errorf("checked %q, downloaded %q", events[0].version, downloaded.archive)
	}
}

func TestReleaseUpdateUpToDateOnlyChecks(t *testing.T) {
	events, result, err := recordUpdate(t, "v0.0.8", false)
	if err != nil || result.updated {
		t.Fatalf("update = %+v, %v", result, err)
	}
	if got := eventKinds(events); !slices.Equal(got, []updateEventKind{updateChecked}) {
		t.Fatalf("events = %v, want only checked", got)
	}
}

func TestReleaseUpdateBadChecksumStopsBeforeVerifyAndInstall(t *testing.T) {
	events, _, err := recordUpdate(t, "v0.0.7", true)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v", err)
	}
	for _, kind := range eventKinds(events) {
		if kind == updateVerified || kind == updateInstalled {
			t.Fatalf("events after a checksum mismatch include %v", kind)
		}
	}
}

// localReleaseFixture serves a release built for the running platform, as
// runReleaseUpdate uses runtime.GOOS and runtime.GOARCH.
func localReleaseFixture(t *testing.T) (releaseUpdater, string) {
	t.Helper()
	archive, binary, err := releaseArtifact("v0.0.8", runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	server := testReleaseServer(t, archive, testReleaseArchive(t, runtime.GOOS, binary, []byte("new executable")), false)
	t.Cleanup(server.Close)
	executable := filepath.Join(t.TempDir(), binary)
	if err := os.WriteFile(executable, []byte("old executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	return releaseUpdater{client: server.Client(), apiURL: server.URL + "/latest"}, executable
}

func TestPlainReleaseUpdateOutput(t *testing.T) {
	updater, executable := localReleaseFixture(t)
	var out bytes.Buffer
	if err := runReleaseUpdate(t.Context(), nil, &out, false, updater, executable, "v0.0.7"); err != nil {
		t.Fatal(err)
	}
	text := ansi.Strip(out.String())
	archive, _, _ := releaseArtifact("v0.0.8", runtime.GOOS, runtime.GOARCH)
	for _, want := range []string{"Checked for updates in ", "Downloaded " + archive + " (", "Verified checksum (sha256)", "Installed zlib v0.0.8 in ", " - zlib v0.0.7", " + zlib v0.0.8", "Restart zlib"} {
		if !strings.Contains(text, want) {
			t.Errorf("output lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, ".previous") {
		t.Errorf("output names the backup file:\n%s", text)
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Errorf("plain output carries escape sequences: %q", out.String())
	}
	if data, _ := os.ReadFile(executable); string(data) != "new executable" {
		t.Errorf("executable = %q", data)
	}

	out.Reset()
	if err := runReleaseUpdate(t.Context(), nil, &out, false, updater, executable, "v0.0.8"); err != nil {
		t.Fatal(err)
	}
	if text := ansi.Strip(out.String()); !strings.Contains(text, "zlib v0.0.8 is up to date") || strings.Contains(text, "Downloaded") {
		t.Errorf("up-to-date output:\n%s", text)
	}
	out.Reset()
	if err := runReleaseUpdate(t.Context(), nil, &out, false, updater, executable, "v0.1.0"); err != nil {
		t.Fatal(err)
	}
	if text := ansi.Strip(out.String()); !strings.Contains(text, "This build (v0.1.0) is newer than the latest release (v0.0.8)") {
		t.Errorf("newer-build output:\n%s", text)
	}
}

func TestPlainReleaseUpdateCancelledIsErrCancelled(t *testing.T) {
	updater, executable := localReleaseFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := runReleaseUpdate(ctx, nil, &bytes.Buffer{}, false, updater, executable, "v0.0.7")
	if !errors.Is(err, errCancelled) {
		t.Fatalf("err = %v, want errCancelled", err)
	}
	if data, _ := os.ReadFile(executable); string(data) != "old executable" {
		t.Errorf("cancelled update replaced the executable: %q", data)
	}
}

func progressStep(t *testing.T, m updateProgressModel, msg tea.Msg) (updateProgressModel, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	return next.(updateProgressModel), cmd
}

func TestUpdateProgressModelLiveLine(t *testing.T) {
	m := newUpdateProgressModel(func() {})
	m, _ = progressStep(t, m, tea.WindowSizeMsg{Width: 90, Height: 20})
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "Checking for updates…") {
		t.Fatalf("initial view = %q", view)
	}
	m, cmd := progressStep(t, m, updateEventMsg{kind: updateChecked, version: "v0.0.13", elapsed: 180 * time.Millisecond})
	if cmd == nil {
		t.Fatal("finished step printed no line")
	}
	archive := "zlib_0.0.13_linux_x86_64.tar.gz"
	m, _ = progressStep(t, m, updateEventMsg{kind: updateDownloadStarted, archive: archive, total: 10 << 20})
	m, cmd = progressStep(t, m, updateEventMsg{kind: updateDownloadProgress, archive: archive, written: 4 << 20, total: 10 << 20})
	if cmd == nil {
		t.Fatal("progress did not animate the bar")
	}
	view := ansi.Strip(m.View().Content)
	for _, want := range []string{"zlib v0.0.13", "─", "4.0 MiB/10.0 MiB"} {
		if !strings.Contains(view, want) {
			t.Errorf("download line lacks %q: %q", want, view)
		}
	}
	if ansi.StringWidth(m.View().Content) > 90 {
		t.Errorf("download line is wider than the terminal: %q", view)
	}
	m, _ = progressStep(t, m, updateEventMsg{kind: updateDownloaded, archive: archive, written: 10 << 20, total: 10 << 20, elapsed: 2 * time.Second})
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "Verifying checksum…") {
		t.Errorf("after download = %q", view)
	}
	m, _ = progressStep(t, m, updateFinishedMsg{result: updateResult{version: "v0.0.13", updated: true}})
	if !m.done || m.View().Content != "" {
		t.Errorf("finished model still draws %q", m.View().Content)
	}
}

func TestUpdateProgressModelUnknownSizeShowsBytesOnly(t *testing.T) {
	m := newUpdateProgressModel(func() {})
	archive := "zlib_0.0.13_linux_x86_64.tar.gz"
	m, _ = progressStep(t, m, updateEventMsg{kind: updateDownloadStarted, archive: archive, total: -1})
	m, cmd := progressStep(t, m, updateEventMsg{kind: updateDownloadProgress, archive: archive, written: 3 << 20, total: -1})
	view := ansi.Strip(m.View().Content)
	if cmd != nil || strings.ContainsAny(view, "━─") || !strings.Contains(view, "3.0 MiB") || strings.Contains(view, "MiB/") {
		t.Errorf("unknown-size line = %q", view)
	}
}

func TestUpdateProgressBarAnimatesTowardTheTarget(t *testing.T) {
	m := newUpdateProgressModel(func() {})
	m, _ = progressStep(t, m, tea.WindowSizeMsg{Width: 90, Height: 20})
	archive := "zlib_0.0.13_linux_x86_64.tar.gz"
	m, _ = progressStep(t, m, updateEventMsg{kind: updateDownloadStarted, archive: archive, total: 100})
	m, cmd := progressStep(t, m, updateEventMsg{kind: updateDownloadProgress, archive: archive, written: 50, total: 100})
	// Run the bar's animation frames, as the program would.
	for i := 0; cmd != nil && i < 120; i++ {
		frame, ok := cmd().(progress.FrameMsg)
		if !ok {
			break
		}
		m, cmd = progressStep(t, m, frame)
	}
	view := ansi.Strip(m.View().Content)
	filled, empty := strings.Count(view, "━"), strings.Count(view, "─")
	if filled == 0 || empty == 0 || filled < empty-2 || filled > empty+2 {
		t.Errorf("half-way bar has %d filled and %d empty cells: %q", filled, empty, view)
	}
}

func TestUpdateProgressModelWaitsForStepLinesBeforeQuitting(t *testing.T) {
	m := newUpdateProgressModel(func() {})
	m, _ = progressStep(t, m, updateEventMsg{kind: updateInstalled, version: "v0.0.13"})
	m, cmd := progressStep(t, m, updateFinishedMsg{result: updateResult{version: "v0.0.13", updated: true}})
	if cmd != nil {
		t.Fatal("quit before the Installed line was printed")
	}
	if _, cmd = progressStep(t, m, updateLinePrintedMsg{}); cmd == nil {
		t.Fatal("did not quit once every line was printed")
	}
}

func TestUpdateProgressModelCtrlCCancels(t *testing.T) {
	cancelled := false
	m := newUpdateProgressModel(func() { cancelled = true })
	m, _ = progressStep(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if !cancelled || !m.cancelled || !strings.Contains(m.View().Content, "Cancelling") {
		t.Fatalf("ctrl+c: cancel called %v, model %+v", cancelled, m.cancelled)
	}
	if _, cmd := progressStep(t, m, updateEventMsg{kind: updateChecked}); cmd != nil {
		t.Error("events after cancelling still print lines")
	}
}

// syncBuffer lets the test read what the program wrote while it runs.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestInteractiveReleaseUpdateRunsInlineProgram(t *testing.T) {
	updater, executable := localReleaseFixture(t)
	var out syncBuffer
	in, writer := io.Pipe()
	defer writer.Close()
	if err := runReleaseUpdate(t.Context(), in, &out, true, updater, executable, "v0.0.7"); err != nil {
		t.Fatal(err)
	}
	text := ansi.Strip(out.String())
	for _, want := range []string{"Checked for updates in ", "Verified checksum (sha256)", "Installed zlib v0.0.8", " - zlib v0.0.7", " + zlib v0.0.8"} {
		if !strings.Contains(text, want) {
			t.Errorf("output lacks %q:\n%s", want, text)
		}
	}
	if data, _ := os.ReadFile(executable); string(data) != "new executable" {
		t.Errorf("executable = %q", data)
	}
}
