// Package tui collects choices; services own resolution, preflight and mutations.
package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"time"
)

type Screen string

const (
	Welcome    Screen = "welcome"
	Version    Screen = "version"
	Location   Screen = "location"
	Path       Screen = "path"
	Summary    Screen = "summary"
	Installing Screen = "installing"
	Done       Screen = "done"
	Blocked    Screen = "blocked"
	Failed     Screen = "failed"
)

type Choices struct {
	Spec, Arch, Root, Path string
	Prerelease             bool
}
type Prepared struct {
	Version, Decision, Info, LogPath string
	Pre                              bool
	Err                              error
}
type Finished struct {
	Message string
	Err     error
}
type Step string
type Progress struct{ Done, Total int64 }
type tick time.Time
type Services struct {
	Prepare func(Choices) Prepared
	Install func(Choices) Finished
}
type Model struct {
	Screen               Screen
	Choices              Choices
	Services             Services
	Prepared             Prepared
	Result               Finished
	LogPath              string
	busy, advanced, info bool
	exiting              bool
	field                int
	input                string
	choice               int
	conflict             string
	steps                []string
	progress             Progress
	frame                int
	width, height        int
}

func New(c Choices, s Services, logPath string) Model {
	return Model{Screen: Welcome, Choices: c, Services: s, LogPath: logPath, width: 80, height: 25}
}
func (m Model) Init() tea.Cmd { return nil }
func (m Model) prepare() tea.Cmd {
	c := m.Choices
	return func() tea.Msg { return m.Services.Prepare(c) }
}
func pulse() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tick(t) })
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case EditText:
		if m.advanced && !m.busy {
			m.input = string(v)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
		return m, nil
	case tick:
		m.frame++
		if m.Screen == Installing {
			return m, pulse()
		}
		return m, nil
	case Step:
		m.steps = append(m.steps, string(v))
		m.progress = Progress{}
		return m, nil
	case Progress:
		m.progress = v
		return m, nil
	case Finished:
		m.busy = false
		m.Result = v
		m.Screen = Done
		if v.Err != nil {
			m.Screen = Failed
		}
		return m, nil
	case Prepared:
		m.busy = false
		m.Prepared = v
		if v.LogPath != "" {
			m.LogPath = v.LogPath
		}
		if v.Err != nil {
			m.Screen = Blocked
			return m, nil
		}
		if m.Screen == Welcome || m.Screen == Version {
			m.Screen = Version
			return m, nil
		}
		if m.Choices.Path == "none" {
			m.Screen = Summary
			return m, nil
		}
		m.conflict = v.Decision
		m.choice = 0
		switch v.Decision {
		case "AddNoClash":
			m.Choices.Path = "first"
			m.Screen = Summary
		case "AliasBlock":
			m.Screen = Blocked
		default:
			m.Screen = Path
		}
		return m, nil
	case tea.KeyMsg:
		key := v.String()
		if m.Screen == Installing {
			return m, nil
		} // Finish or recover before leaving; process interruption is journaled.
		if key == "ctrl+c" || (!m.advanced && (key == "q" || key == "esc")) {
			m.exiting = true
			return m, tea.Quit
		}
		if m.busy {
			return m, nil
		}
		if m.advanced {
			if key == "esc" {
				m.advanced = false
				return m, nil
			}
			if key == "enter" || key == "tab" {
				if m.Screen == Location {
					m.Choices.Root = m.input
					m.advanced = false
				} else {
					switch m.field {
					case 0:
						m.Choices.Spec = m.input
						m.field = 1
						m.input = m.Choices.Arch
					case 1:
						m.Choices.Arch = m.input
						m.field = 2
						m.input = "no"
					case 2:
						m.Choices.Prerelease = strings.EqualFold(m.input, "yes")
						m.advanced = false
						m.busy = true
						return m, m.prepare()
					}
				}
				return m, nil
			}
			if key == "backspace" {
				r := []rune(m.input)
				if len(r) > 0 {
					m.input = string(r[:len(r)-1])
				}
			} else if v.Type == tea.KeyRunes {
				m.input += string(v.Runes)
			} else if key == "space" {
				m.input += " "
			}
			return m, nil
		}
		if key == "b" && m.PreviousAvailable() {
			switch m.Screen {
			case Version:
				m.Screen = Welcome
			case Location:
				m.Screen = Version
			case Path:
				m.Screen = Location
			case Summary:
				m.Screen = Path
			case Blocked:
				m.Screen = Location
			}
			// Going back revokes any earlier PATH takeover consent.
			m.Choices.Path = "auto"
			m.conflict = ""
			m.choice = 0
			m.info = false
			return m, nil
		}
		if key == "i" && m.InfoAvailable() {
			m.info = !m.info
			return m, nil
		}
		if key == "a" && (m.Screen == Version || m.Screen == Location) {
			m.advanced = true
			m.field = 0
			m.input = m.Choices.Spec
			if m.Screen == Location {
				m.input = m.Choices.Root
			}
			return m, nil
		}
		switch m.Screen {
		case Welcome:
			if key == "enter" {
				m.busy = true
				return m, m.prepare()
			}
		case Version:
			if key == "enter" {
				m.Screen = Location
			}
		case Location:
			if key == "enter" {
				m.Screen = Path
				m.conflict = ""
				m.choice = 0
			}
		case Path:
			switch m.conflict {
			case "MustChoose":
				// There is deliberately no preselection. Enter cannot consent.
				if key == "1" {
					m.Choices.Path = "first"
					m.Screen = Summary
				}
				if key == "2" {
					m.Choices.Path = "none"
					m.Screen = Summary
				}
			case "OnlyNone", "Unknown":
				if key == "y" {
					m.choice = 1
				}
				if key == "n" {
					m.choice = 0
				}
				if key == "enter" {
					if m.choice == 1 {
						m.Choices.Path = "none"
						m.Screen = Summary
					} else {
						m.Screen = Blocked
					}
				}
			default:
				if key == "n" || key == "right" || key == "down" {
					m.choice = 1
				}
				if key == "y" || key == "left" || key == "up" {
					m.choice = 0
				}
				if key == "enter" {
					m.Choices.Path = "auto"
					if m.choice == 1 {
						m.Choices.Path = "none"
					}
					m.busy = true
					return m, m.prepare()
				}
			}
		case Summary:
			if key == "enter" {
				m.Screen = Installing
				m.busy = true
				c := m.Choices
				return m, tea.Batch(pulse(), func() tea.Msg { return m.Services.Install(c) })
			}
		case Blocked:
			if key == "r" || key == "enter" {
				m.Screen = Welcome
				m.conflict = ""
				m.busy = true
				return m, m.prepare()
			}
		case Done, Failed:
			if key == "enter" {
				m.exiting = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}
func (m Model) View() string {
	if m.exiting {
		switch m.Screen {
		case Done:
			return "Python is ready." + string(rune(10)) + m.Result.Message + string(rune(10))
		case Failed:
			return m.Result.Err.Error() + string(rune(10)) + "Log: " + m.LogPath + string(rune(10))
		default:
			return "Wizard closed." + string(rune(10))
		}
	}
	lines := []string{"Python for the Win", ""}
	add := func(s string) { lines = append(lines, strings.Split(s, "\n")...) }
	if m.busy && m.Screen != Installing {
		add("Checking available Python and your installation...")
	} else if m.advanced {
		label := "Installation folder"
		if m.Screen == Version {
			label = []string{"Version (latest or explicit minor, e.g. 3.16)", "Architecture (amd64, arm64, x86)", "Allow development releases? (yes/no)"}[m.field]
		}
		add("Advanced options\n" + label + "\n> " + m.input + "_\nEnter: accept   Esc: cancel")
	} else {
		switch m.Screen {
		case Welcome:
			add("Install official Python for your user account.\n\nEnter: begin   Q: quit")
		case Version:
			badge := ""
			if m.Prepared.Pre {
				badge = " [development release]"
			}
			add("Python " + m.Prepared.Version + badge + "\n\nEnter: next   A: Advanced options")
		case Location:
			add("Installation folder:\n" + m.Choices.Root + "\n\nEnter: next   A: Advanced options")
		case Path:
			switch m.conflict {
			case "MustChoose":
				add("Another Python is on your user PATH. Choose explicitly:\n\n1: Make pftw's Python take precedence\n2: Install without changing PATH")
			case "OnlyNone", "Unknown":
				add("Another Python may still run from PATH.\nContinue without changing PATH?")
				if m.choice == 0 {
					add("[No]  Yes")
				} else {
					add("No  [Yes]")
				}
				add("Y/N: choose   Enter: confirm")
			default:
				add("Add Python to PATH?")
				if m.choice == 0 {
					add("[Yes]  No")
				} else {
					add("Yes  [No]")
				}
				add("Y/N or arrows: choose   Enter: next")
			}
		case Summary:
			add("Ready to install Python " + m.Prepared.Version + "\nFolder: " + m.Choices.Root + "\nPATH: " + m.Choices.Path + "\n\nEnter: install   Q: quit")
		case Installing:
			for n, s := range m.steps {
				mark := "[ok]"
				if n == len(m.steps)-1 {
					mark = string("|/-\\"[m.frame%4])
				}
				add(mark + " " + s)
			}
			if m.progress.Total > 0 {
				p := int(m.progress.Done * 100 / m.progress.Total)
				if p < 0 {
					p = 0
				}
				if p > 100 {
					p = 100
				}
				add(fmt.Sprintf("[%s%s] %d%%", strings.Repeat("#", p/5), strings.Repeat("-", 20-p/5), p))
			}
		case Done:
			add("Python is ready.\n" + m.Result.Message + "\n\nEnter: finish")
		case Failed:
			add("[failed] Installation could not finish.\n" + m.Result.Err.Error() + "\nLog: " + m.LogPath + "\n\nEnter: quit")
		case Blocked:
			message := "Installation paused; nothing new was installed."
			if m.Prepared.Err != nil {
				message = m.Prepared.Err.Error()
			} else if m.conflict == "AliasBlock" {
				message = "A Python app execution alias takes precedence.\nTurn off that Python alias, or choose No for PATH."
			}
			add(message + "\nLog: " + m.LogPath + "\n\nR / Enter: recheck   Q: quit")
		}
	}
	if m.PreviousAvailable() {
		add("B: Previous")
	}
	if m.InfoAvailable() {
		add("\nI: Advanced user info")
		if m.info {
			add(m.Prepared.Info)
		}
	}
	width := m.width - 1
	if width < 20 {
		width = 20
	}
	out := []string{}
	for _, line := range lines {
		out = append(out, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
	}
	if m.height > 0 && len(out) >= m.height {
		out = append(out[:m.height-2], "More detail: pftw doctor / log file")
	}
	return strings.Join(out, "\n") + "\n"
}

// InfoAvailable mirrors the input gate, so a visible hint always works.
func (m Model) InfoAvailable() bool {
	return !m.exiting && !m.advanced && !m.busy && m.Screen != Installing && m.Prepared.Info != ""
}

type EditText string

func (m Model) PreviousAvailable() bool {
	if m.busy || m.advanced || m.exiting {
		return false
	}
	switch m.Screen {
	case Version, Location, Path, Summary, Blocked:
		return true
	}
	return false
}
