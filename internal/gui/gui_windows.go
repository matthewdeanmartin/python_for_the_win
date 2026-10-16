// Package gui renders the shared wizard in a native Gio window on Windows.
package gui

import (
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"image/color"
	"sync"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/matthewdeanmartin/python_for_the_win/internal/crash"
	"github.com/matthewdeanmartin/python_for_the_win/internal/tui"
)

// Main gives Gio the process main goroutine only for an explicitly graphical launch.
func Main(run func()) { go run(); app.Main() }

// Run owns the UI model on one goroutine. Worker messages wake the window; close
// cancels services and waits for their rollback/cleanup before the command returns.
func Run(ctx context.Context, cancel context.CancelFunc, model tui.Model, bind func(func(tea.Msg))) (tui.Model, error) {
	window := new(app.Window)
	window.Option(app.Title("Python for the Win"), app.Size(unit.Dp(720), unit.Dp(620)), app.MinSize(unit.Dp(420), unit.Dp(360)))
	messages := make(chan tea.Msg, 64)
	var workers sync.WaitGroup
	send := func(msg tea.Msg) {
		select {
		case messages <- msg:
			window.Invalidate()
		case <-ctx.Done():
		}
	}
	bind(send)
	execute := func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		workers.Add(1)
		go func() {
			defer workers.Done()
			var msg tea.Msg
			var err error
			func() { defer crash.Recover(&err); msg = cmd() }()
			if err != nil {
				msg = tui.Finished{Err: err}
			}
			if msg != nil {
				send(msg)
			}
		}()
	}
	defer func() { cancel(); workers.Wait() }()
	go func() { <-ctx.Done(); window.Perform(system.ActionClose) }()
	update := func(msg tea.Msg) {
		next, cmd := model.Update(msg)
		model = next.(tui.Model)
		execute(cmd)
	}
	press := func(action string) {
		k := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(action)}
		if action == "enter" {
			k = tea.KeyMsg{Type: tea.KeyEnter}
		}
		if action == "esc" {
			k = tea.KeyMsg{Type: tea.KeyEsc}
		}
		update(k)
	}
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Regular()))
	theme.Palette.Bg = color.NRGBA{R: 248, G: 250, B: 252, A: 255}
	theme.Palette.Fg = color.NRGBA{R: 24, G: 35, B: 50, A: 255}
	theme.Palette.ContrastBg = color.NRGBA{R: 30, G: 75, B: 140, A: 255}
	var ops op.Ops
	var list widget.List
	list.Axis = layout.Vertical
	buttons := map[string]*widget.Clickable{}
	var editor widget.Editor
	editor.SingleLine = true
	editor.Submit = true
	editingLabel := ""
	lastTitle := ""
	helpError := ""
	for {
		switch event := window.Event().(type) {
		case app.DestroyEvent:
			if event.Err != nil {
				return model, fmt.Errorf("could not open the graphical wizard: %w; run pftw without a command in a terminal for the text wizard", event.Err)
			}
			return model, nil
		case app.FrameEvent:
		drain:
			for {
				select {
				case msg := <-messages:
					switch msg := msg.(type) {
					case tea.QuitMsg:
						window.Perform(system.ActionClose)
					case tea.BatchMsg:
						for _, cmd := range msg {
							execute(cmd)
						}
					default:
						update(msg)
					}
				default:
					break drain
				}
			}
			gtx := app.NewContext(&ops, event)
			page := model.Page()
			if page.Editing {
				if editingLabel != page.EditorLabel {
					editor.SetText(page.EditorValue)
					editingLabel = page.EditorLabel
					gtx.Execute(key.FocusCmd{Tag: &editor})
				}
				for {
					ev, ok := editor.Update(gtx)
					if !ok {
						break
					}
					switch ev.(type) {
					case widget.ChangeEvent:
						update(tui.EditText(editor.Text()))
					case widget.SubmitEvent:
						update(tui.EditText(editor.Text()))
						press("enter")
					}
				}
			} else {
				editingLabel = ""
			}

			// Process only controls present on this screen; stale clicks cannot authorize a new choice.
			for _, action := range page.Actions {
				if button := buttons[action.Key]; button != nil && button.Clicked(gtx) {
					if page.Editing {
						update(tui.EditText(editor.Text()))
					}
					if action.Key == "help" {
						verb, _ := windows.UTF16PtrFromString("open")
						destination := page.HelpURL
						if destination == "" {
							destination = DocumentationURL
						}
						url, _ := windows.UTF16PtrFromString(destination)
						if err := windows.ShellExecute(0, verb, url, nil, nil, windows.SW_SHOWNORMAL); err != nil {
							helpError = "Could not open Help. Visit " + destination
						}
					} else {
						press(action.Key)
					}
					break
				}
			}
			page = model.Page()
			if page.Editing && editingLabel != page.EditorLabel {
				editor.SetText(page.EditorValue)
				editingLabel = page.EditorLabel
				gtx.Execute(key.FocusCmd{Tag: &editor})
			}
			if page.Title != lastTitle {
				list.Position = layout.Position{}
				lastTitle = page.Title
				for _, action := range page.Actions {
					if buttons[action.Key] == nil {
						buttons[action.Key] = new(widget.Clickable)
					}
					if action.Key == "enter" && action.Primary && !page.Editing {
						gtx.Execute(key.FocusCmd{Tag: buttons[action.Key]})
					}
				}
			}
			paint.Fill(gtx.Ops, theme.Palette.Bg)
			layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				items := []layout.Widget{
					material.Caption(theme, "PYTHON FOR THE WIN").Layout,
					func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 12, Bottom: 18}.Layout(gtx, material.H4(theme, page.Title).Layout)
					},
				}
				if page.Body != "" {
					items = append(items, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: 20}.Layout(gtx, material.Body1(theme, page.Body).Layout)
					})
				}
				if page.Fraction >= 0 {
					items = append(items, material.ProgressBar(theme, page.Fraction).Layout)
				}
				if page.Editing {
					items = append(items, material.Body1(theme, page.EditorLabel).Layout, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 12, Bottom: 20}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return widget.Border{Color: color.NRGBA{R: 100, G: 116, B: 139, A: 255}, Width: 1, CornerRadius: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(12).Layout(gtx, material.Editor(theme, &editor, "Enter a value").Layout)
							})
						})
					})
				}
				renderButton := func(action tui.Action) layout.Widget {
					if buttons[action.Key] == nil {
						buttons[action.Key] = new(widget.Clickable)
					}
					return func(gtx layout.Context) layout.Dimensions {
						button := material.Button(theme, buttons[action.Key], action.Label)
						if !action.Primary {
							button.Background = color.NRGBA{R: 226, G: 232, B: 240, A: 255}
							button.Color = theme.Palette.Fg
						}
						return button.Layout(gtx)
					}
				}
				for _, group := range []string{"form", "navigation", "utility"} {
					actions := []tui.Action{}
					for _, action := range page.Actions {
						if action.Group == group {
							actions = append(actions, action)
						}
					}
					if len(actions) == 0 {
						continue
					}
					if group == "navigation" {
						for i, a := range actions {
							if a.Key == "b" {
								actions = append([]tui.Action{a}, append(actions[:i], actions[i+1:]...)...)
								break
							}
						}
					}
					if group == "form" {
						for _, a := range actions {
							render := renderButton(a)
							items = append(items, func(gtx layout.Context) layout.Dimensions { return layout.Inset{Bottom: 8}.Layout(gtx, render) })
						}
					} else {
						children := []layout.FlexChild{}
						for i, a := range actions {
							if i > 0 {
								children = append(children, layout.Rigid(layout.Spacer{Width: 12}.Layout))
							}
							children = append(children, layout.Flexed(1, renderButton(a)))
						}
						items = append(items, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: 24}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
							})
						})
					}
				}
				if helpError != "" {
					items = append(items, material.Body2(theme, helpError).Layout)
				}

				if page.Info != "" {
					items = append(items, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: 16}.Layout(gtx, material.Body2(theme, page.Info).Layout)
					})
				}
				return list.Layout(gtx, len(items), func(gtx layout.Context, index int) layout.Dimensions { return items[index](gtx) })
			})
			event.Frame(gtx.Ops)
		}
	}
}
