//go:build !windows

package gui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/tui"
)

func Main(run func()) { run() }
func Run(_ context.Context, _ context.CancelFunc, m tui.Model, _ func(func(tea.Msg))) (tui.Model, error) {
	return m, errs.New(8, "The graphical wizard is currently supported only on Windows")
}
