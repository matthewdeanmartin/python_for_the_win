package tui

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
)

func key(m Model, s string) Model {
	k := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	if s == "enter" {
		k = tea.KeyMsg{Type: tea.KeyEnter}
	}
	n, _ := m.Update(k)
	return n.(Model)
}
func prepared(m Model, p Prepared) Model { n, _ := m.Update(p); return n.(Model) }
func TestConsent(t *testing.T) {
	for _, decision := range []string{"MustChoose", "OnlyNone", "Unknown"} {
		m := New(Choices{Path: "auto"}, Services{}, "log")
		m.Screen = Path
		m = prepared(m, Prepared{Decision: decision})
		m = key(m, "enter")
		if m.Screen == Summary || m.Screen == Installing {
			t.Fatal("Enter consented", decision)
		}
		m.Screen = Path
		if decision == "MustChoose" {
			m = key(m, "1")
		} else {
			m = key(m, "y")
			m = key(m, "enter")
		}
		if m.Screen != Summary {
			t.Fatal(decision, m.Screen)
		}
	}
	m := New(Choices{Path: "none"}, Services{}, "log")
	m.Screen = Path
	m = prepared(m, Prepared{Decision: "OnlyNone"})
	if m.Screen != Summary {
		t.Fatal("No must bypass conflict")
	}
	m.Choices.Path = "auto"
	m.Screen = Path
	m = prepared(m, Prepared{Decision: "AddNoClash"})
	if m.Screen != Summary || m.Choices.Path != "first" {
		t.Fatal(m)
	}
}
func TestFlowAndFailure(t *testing.T) {
	m := New(Choices{}, Services{}, "example.log")
	m = key(m, "enter")
	m = prepared(m, Prepared{Err: errors.New("offline")})
	if m.Screen != Blocked {
		t.Fatal(m)
	}
	m = key(m, "r")
	m = prepared(m, Prepared{Version: "3.16.0a1", Pre: true})
	if m.Screen != Version || !strings.Contains(m.View(), "development release") {
		t.Fatal(m)
	}
	m = key(m, "enter")
	if m.Screen != Location {
		t.Fatal(m)
	}
	m = key(m, "enter")
	if m.Screen != Path {
		t.Fatal(m)
	}
	m = key(m, "n")
	m = key(m, "enter")
	m = prepared(m, Prepared{})
	if m.Screen != Summary {
		t.Fatal(m)
	}
	m = key(m, "enter")
	if m.Screen != Installing {
		t.Fatal(m)
	}
	n, _ := m.Update(Finished{Err: errors.New("broken")})
	m = n.(Model)
	if m.Screen != Failed || !strings.Contains(m.View(), "example.log") {
		t.Fatal(m)
	}
	// Failures link to troubleshooting instead of explaining everything on screen.
	if !strings.Contains(strings.ReplaceAll(m.View(), "\n", ""), TroubleshootingURL) || m.Page().HelpURL != TroubleshootingURL {
		t.Fatal("failure screen does not lead to troubleshooting")
	}
	n, _ = m.Update(Finished{Message: "done"})
	if n.(Model).Screen != Done {
		t.Fatal(n)
	}
}
func TestPlainSmallTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := New(Choices{Root: strings.Repeat("long folder ", 40)}, Services{}, "log")
	for _, screen := range []Screen{Welcome, Version, Location, Path, Summary, Installing, Blocked, Done, Failed} {
		m.Screen = screen
		m.Result.Err = errors.New(strings.Repeat("error ", 200))
		text := m.View()
		if strings.Contains(text, "\x1b") {
			t.Fatal("color")
		}
		lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
		if len(lines) > 25 {
			t.Fatal("height", len(lines))
		}
		for _, line := range lines {
			if len([]rune(line)) > 80 {
				t.Fatal("width", line)
			}
		}
	}
}

func TestInfoHintOnlyWhenActionWorks(t *testing.T) {
	m := New(Choices{}, Services{}, "log")
	m.Prepared.Info = "Details"
	m.Screen = Version
	for _, busy := range []bool{true, false} {
		m.busy = busy
		visible := strings.Contains(m.View(), "I: Advanced user info")
		next := key(m, "i")
		if visible == busy || (next.info != m.info) != visible {
			t.Fatal("hint and input disagree", busy, visible)
		}
	}
	m.Screen = Installing
	m.busy = true
	if strings.Contains(m.View(), "I: Advanced user info") || len(m.Page().Actions) > 0 {
		t.Fatal("advertised unavailable input")
	}
	m.Screen = Version
	m.busy = false
	m.Prepared.Info = ""
	if strings.Contains(m.View(), "I: Advanced user info") {
		t.Fatal("empty info hint")
	}
}
func TestGUIConsentActions(t *testing.T) {
	m := New(Choices{Path: "auto"}, Services{}, "log")
	m.Screen = Path
	m = prepared(m, Prepared{Decision: "MustChoose"})
	for _, a := range m.Page().Actions {
		if a.Key == "enter" || a.Primary {
			t.Fatal("GUI supplies implicit takeover choice", a)
		}
	}
	m = key(m, "enter")
	if m.Screen != Path {
		t.Fatal("Enter consented")
	}
	m = key(m, "2")
	if m.Screen != Summary || m.Choices.Path != "none" {
		t.Fatal(m)
	}
}

func TestExitLeavesNoDeadKeyboardHints(t *testing.T) {
	m := New(Choices{}, Services{}, "log")
	m.Prepared.Info = "Details"
	m.Screen = Done
	m.Result.Message = "Run python --version"
	m = key(m, "enter")
	if strings.Contains(m.View(), "I:") || strings.Contains(m.View(), "Enter:") || !strings.Contains(m.View(), "Run python --version") {
		t.Fatal(m.View())
	}
}

func TestPreviousRevokesPathConsent(t *testing.T) {
	m := New(Choices{Path: "first"}, Services{}, "")
	m.Screen = Summary
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	got := next.(Model)
	if got.Screen != Path || got.Choices.Path != "auto" || got.conflict != "" {
		t.Fatal(got)
	}
	page := got.Page()
	groups := map[string]string{}
	for _, a := range page.Actions {
		groups[a.Key] = a.Group
	}
	if groups["b"] != "navigation" || groups["enter"] != "navigation" || groups["help"] != "utility" || groups["n"] != "form" {
		t.Fatal(groups)
	}
	got.Screen = Installing
	got.busy = true
	if got.PreviousAvailable() {
		t.Fatal("previous offered during mutation")
	}
}
