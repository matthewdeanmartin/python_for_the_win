package tui

import "fmt"

// Page is the graphical presentation of the shared wizard state machine.
// Actions use the same inputs as the terminal frontend; neither frontend grants consent itself.
type Action struct {
	Key, Label string
	Primary    bool
	Group      string
}
type Page struct {
	Title, Body, Info, EditorLabel, EditorValue string
	Actions                                     []Action
	Editing, Working                            bool
	Fraction                                    float32
}

func (m Model) Page() Page {
	p := Page{Working: m.busy, Fraction: -1}
	add := func(key, label string, primary bool) {
		p.Actions = append(p.Actions, Action{Key: key, Label: label, Primary: primary, Group: actionGroup(key)})
	}
	choice := func(key, label string, selected bool) {
		if selected {
			label += " (selected)"
		}
		add(key, label, selected)
	}
	if m.busy && m.Screen != Installing {
		p.Title = "Checking your installation"
		p.Body = "Looking for available Python versions and checking this folder."
		return p
	}
	if m.advanced {
		p.Title = "Advanced options"
		p.Editing = true
		p.EditorValue = m.input
		p.EditorLabel = "Installation folder"
		if m.Screen == Version {
			p.EditorLabel = []string{"Version (latest or an explicit version, such as 3.16)", "Architecture (amd64, arm64, x86)", "Allow development releases? (yes/no)"}[m.field]
		}
		add("enter", "Continue", true)
		add("esc", "Cancel editing", false)
		return p
	}
	switch m.Screen {
	case Welcome:
		p.Title = "Install Python"
		p.Body = "Official Python, installed for your user account.\nThe defaults select the newest available final release."
		add("enter", "Get started", true)
	case Version:
		p.Title = "Choose Python"
		p.Body = "Python " + m.Prepared.Version
		if m.Prepared.Pre {
			p.Body += " (development release)"
		}
		add("enter", "Next", true)
		add("a", "Advanced options", false)
	case Location:
		p.Title = "Choose a folder"
		p.Body = m.Choices.Root
		add("enter", "Next", true)
		add("a", "Change folder", false)
	case Path:
		switch m.conflict {
		case "MustChoose":
			p.Title = "Another Python is on your PATH"
			p.Body = "Choose whether pftw's Python should take precedence."
			add("1", "Make pftw take precedence", false)
			add("2", "Leave PATH unchanged", false)
		case "OnlyNone", "Unknown":
			p.Title = "Continue without changing PATH?"
			p.Body = "Another Python may still run from PATH."
			choice("n", "No", m.choice == 0)
			choice("y", "Yes", m.choice == 1)
			add("enter", "Confirm choice", false)
		default:
			p.Title = "Add Python to PATH?"
			p.Body = "This lets you run Python by typing python in a new terminal."
			choice("y", "Yes", m.choice == 0)
			choice("n", "No", m.choice == 1)
			add("enter", "Next", true)
		}
	case Summary:
		p.Title = "Ready to install"
		p.Body = fmt.Sprintf("Python %s\n\nFolder: %s\n\nPATH: %s", m.Prepared.Version, m.Choices.Root, m.Choices.Path)
		add("enter", "Install", true)
	case Installing:
		p.Title = "Installing Python"
		p.Body = "Preparing installation..."
		if len(m.steps) > 0 {
			p.Body = ""
			for i, s := range m.steps {
				mark := "Done: "
				if i == len(m.steps)-1 {
					mark = "Working: "
				}
				p.Body += mark + s + "\n"
			}
		}
		if m.progress.Total > 0 {
			p.Fraction = float32(m.progress.Done) / float32(m.progress.Total)
			if p.Fraction > 1 {
				p.Fraction = 1
			}
			p.Body += fmt.Sprintf("\nDownload: %.0f%%", p.Fraction*100)
		}
		return p
	case Done:
		p.Title = "Python is ready"
		p.Body = m.Result.Message
		add("enter", "Finish", true)
	case Failed:
		p.Title = "Installation could not finish"
		p.Body = m.Result.Err.Error() + "\n\nLog: " + m.LogPath
		add("enter", "Close", true)
	case Blocked:
		p.Title = "Installation paused"
		p.Body = "Nothing new was installed."
		if m.Prepared.Err != nil {
			p.Body = m.Prepared.Err.Error()
		} else if m.conflict == "AliasBlock" {
			p.Body = "A Python app execution alias takes precedence. Turn off that alias, or choose No for PATH."
		}
		p.Body += "\n\nLog: " + m.LogPath
		add("r", "Recheck", true)
	}
	if m.PreviousAvailable() {
		add("b", "Previous", false)
	}
	add("help", "Help", false)
	if m.Screen != Done && m.Screen != Failed {
		add("q", "Quit", false)
	}
	return p
}

func actionGroup(key string) string {
	switch key {
	case "enter", "b", "r", "esc":
		return "navigation"
	case "help", "q":
		return "utility"
	default:
		return "form"
	}
}
