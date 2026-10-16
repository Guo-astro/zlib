package cli

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestLoadingSpinnersStopAfterCompletion(t *testing.T) {
	p := newProfilePageModel(nil)
	if _, cmd := p.Update(p.spinner.Tick()); cmd == nil {
		t.Fatal("profile loading spinner did not schedule a frame")
	}
	next, _ := p.Update(profileLoadedMsg{})
	p = next.(profilePageModel)
	if _, cmd := p.Update(p.spinner.Tick()); cmd != nil {
		t.Fatal("profile spinner continued after completion")
	}
	s, err := newSettingsModel(t.Context(), testSettingsStore(t), "domain")
	if err != nil {
		t.Fatal(err)
	}
	batch := s.probe("http://127.0.0.1:1", "domain")().(tea.BatchMsg)
	if len(batch) != 2 {
		t.Fatal("domain probe must start both animation and request")
	}
	if _, cmd := s.Update(s.spinner.Tick()); cmd == nil {
		t.Fatal("domain loading spinner did not schedule a frame")
	}
	s.back()
	if _, cmd := s.Update(s.spinner.Tick()); cmd != nil {
		t.Fatal("domain spinner continued after cancellation")
	}
	m := newShellModel()
	m.kind, m.spinner = "login-wait", newLoadingSpinner()
	if _, cmd := m.Update(m.spinner.Tick()); cmd == nil {
		t.Fatal("login loading spinner did not schedule a frame")
	}
}
