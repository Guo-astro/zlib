package cli

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zlib "github.com/heartleo/zlib"
)

func doctorAvailable(domain string) zlib.DomainCheck {
	return zlib.DomainCheck{
		Domain: domain, Status: zlib.DomainStatusHealthy,
		FinalURL: domain + "/eapi/info/domains",
	}
}

func TestDoctorPageListsOnlyAvailableDomains(t *testing.T) {
	m := newDoctorPageModel([]string{"one", "two", "three"}, "", "https://z-lib.gd", nil)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(doctorPageModel)
	next, _ = m.Update(doctorResultMsg{sequence: m.sequence, checks: []zlib.DomainCheck{
		doctorAvailable("https://z-lib.gd"),
		{Domain: "https://z-lib.sk", Status: zlib.DomainStatusChallenged},
		doctorAvailable("https://z-lib.gl"),
	}})
	m = next.(doctorPageModel)
	if len(m.list.Items()) != 2 || m.loading {
		t.Fatal("doctor did not keep only available domains")
	}
	view := m.View()
	text := ansi.Strip(view.Content)
	if !view.AltScreen ||
		!strings.Contains(text, "https://z-lib.gd  · Current domain") ||
		!strings.Contains(text, "│ https://z-lib.gd") ||
		!strings.Contains(text, "z-lib.gd") || !strings.Contains(text, "z-lib.gl") || strings.Contains(text, "z-lib.sk") {
		t.Fatalf("doctor page displayed incorrect domains: %q", text)
	}
	if strings.Contains(text, "Available ·") || strings.Contains(text, " available") || strings.Contains(text, " selected") {
		t.Fatalf("doctor repeats availability on each row: %q", text)
	}
	if len(strings.Split(view.Content, "\n")) != 24 {
		t.Fatal("doctor page does not fill the terminal")
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = next.(doctorPageModel)
	if m.list.Index() != 1 {
		t.Fatal("doctor list did not move its selection")
	}
	text = ansi.Strip(m.View().Content)
	if !strings.Contains(text, "│ https://z-lib.gl") || !strings.Contains(text, "https://z-lib.gd  · Current domain") {
		t.Fatalf("doctor cursor changed the current domain marker: %q", text)
	}
	next, _ = m.Update(tea.WindowSizeMsg{Width: 35, Height: 14})
	m = next.(doctorPageModel)
	for _, line := range strings.Split(m.View().Content, "\n") {
		if ansi.StringWidth(line) > 35 {
			t.Fatalf("doctor exceeds compact terminal width: %q", line)
		}
	}
}

func TestDoctorPageRetryRejectsOldResultsAndCancelsOnExit(t *testing.T) {
	m := newDoctorPageModel([]string{"https://z-lib.gd"}, "", "", nil)
	oldContext, oldSequence := m.ctx, m.sequence
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m = next.(doctorPageModel)
	if cmd == nil || oldContext.Err() == nil || m.sequence == oldSequence || !m.loading {
		t.Fatal("retry did not cancel the old probe and start a new one")
	}
	next, _ = m.Update(doctorResultMsg{sequence: oldSequence, checks: []zlib.DomainCheck{doctorAvailable("https://z-lib.gd")}})
	m = next.(doctorPageModel)
	if len(m.list.Items()) != 0 || !m.loading {
		t.Fatal("stale probe result changed the active page")
	}
	next, _ = m.Update(doctorResultMsg{sequence: m.sequence})
	m = next.(doctorPageModel)
	if !strings.Contains(ansi.Strip(m.View().Content), "No available EAPI domains") {
		t.Fatal("empty result did not explain how to retry")
	}
	currentContext := m.ctx
	next, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = next.(doctorPageModel)
	if cmd == nil || currentContext.Err() == nil {
		t.Fatal("leaving doctor did not cancel its probe context")
	}
}

func TestDoctorPageInitiallyFocusesCurrentDomain(t *testing.T) {
	m := newDoctorPageModel(nil, "", "https://z-lib.gl", nil)
	defer m.stop()
	next, _ := m.Update(doctorResultMsg{sequence: m.sequence, checks: []zlib.DomainCheck{
		doctorAvailable("https://z-lib.gd"), doctorAvailable("https://z-lib.gl"),
	}})
	m = next.(doctorPageModel)
	if m.list.Index() != 1 {
		t.Fatal("current domain did not receive initial focus")
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = next.(doctorPageModel)
	if m.list.Index() != 0 || m.current != "https://z-lib.gl" {
		t.Fatal("moving focus changed the configured domain")
	}
}

func TestDoctorPageScrollsLongDomainList(t *testing.T) {
	m := newDoctorPageModel(nil, "", "", nil)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 14})
	m = next.(doctorPageModel)
	checks := make([]zlib.DomainCheck, 12)
	for i := range checks {
		checks[i] = doctorAvailable(fmt.Sprintf("https://%d.z-lib.gd", i))
	}
	next, _ = m.Update(doctorResultMsg{sequence: m.sequence, checks: checks})
	m = next.(doctorPageModel)
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	m = next.(doctorPageModel)
	if m.list.Index() == 0 || !strings.Contains(ansi.Strip(m.View().Content), checks[m.list.Index()].Domain) {
		t.Fatal("doctor list did not page through available domains")
	}
}

func TestShellDoctorOpensDedicatedPage(t *testing.T) {
	t.Setenv(zlib.EnvDomain, "https://z-lib.gd/")
	m := newShellModel()
	if cmd := m.dispatch("/doctor"); cmd == nil {
		t.Fatal("doctor did not start a probe")
	}
	page, ok := m.page.(doctorPageModel)
	if !ok || m.kind != "doctor" {
		t.Fatalf("doctor opened %T (%q)", m.page, m.kind)
	}
	if page.current != "https://z-lib.gd" {
		t.Fatalf("doctor current domain = %q", page.current)
	}
	ctx := page.ctx
	m, cmd := shellStep(m, key("esc"))
	if cmd == nil {
		t.Fatal("doctor did not return to shell")
	}
	m, _ = shellStep(m, cmd())
	if m.kind != "" || m.page != nil || ctx.Err() == nil {
		t.Fatal("doctor exit did not restore home and cancel the probe")
	}
}

func TestShellDoctorCancelsWhenAnotherCommandOpens(t *testing.T) {
	m := newShellModel()
	m.dispatch("/doctor")
	ctx := m.page.(doctorPageModel).ctx
	m.dispatch("/theme")
	if ctx.Err() == nil || m.kind != "theme" {
		t.Fatal("switching away from doctor kept its probe running")
	}
}
