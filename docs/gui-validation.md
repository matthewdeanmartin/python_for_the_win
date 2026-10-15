# Explicit GUI and hint fix — 2026-10-15

`pftw gui` now opens a native Windows Gio window, including when launched from a terminal or with
redirected standard streams. No arguments still launch the terminal wizard. The same executable provides
CLI, TUI, GUI and versioned shim entry points; no `py` or `pyw` commands are added.

## Changes

- Advanced-info hints are displayed only when the key works and details exist. Busy/installing screens
  hide the hint. Exiting the terminal wizard leaves a plain result instead of dead keyboard instructions.
- The graphical frontend uses the existing wizard state machine and CLI service callbacks. It has buttons,
  advanced text fields, a scrollable layout, download progress and collapsed advanced information.
- Yes/No selections have explicit selected text. User-PATH conflicts have no preselected takeover button.
  Window closure cancels service context and waits for workers to finish cleanup before returning.
- Gio v0.10.3 is behind Windows build constraints. Graphics are entered only for the explicit command;
  terminal/CLI/shim launches keep the console subsystem. The regular Go font keeps the stripped Windows
  binary at 13,257,728 bytes (about 13.3 MB), below the 15 MB budget.

## Validation performed

- Go 1.27.2, CGO disabled: build, all unit tests, vet and pinned Staticcheck passed.
- Regression tests check hint/input agreement during busy states, no hints after exit, and GUI user-PATH
  actions having no default takeover/Enter action. Existing shared-model consent tests still pass.
- Opened the actual window and inspected screenshots of welcome, version, advanced version/architecture
  fields, location, PATH, summary, progress and completion. Verified the advanced field value changes when
  moving from version to architecture. Verified default-button Enter navigation on the final build.
- Completed an actual graphical Python 3.15.0 installation into an isolated root with PATH No. Verified
  final-path pip and the versioned Python shim, closed the window and uninstalled the test runtime.
  The test did not modify the user's real PATH.
- `pftw gui --help` prints help without a window; `pftw gui --json` rejects the incompatible mode with exit 2.
- GoReleaser snapshot packaging passed for Windows amd64/arm64 and Linux/macOS amd64/arm64. Non-Windows
  builds do not compile the graphical backend. No release or tag was created.

Evidence is local under `dist/`: `gui-checks.txt`, `gui-packaging.txt`, `gui-welcome.png`,
`gui-advanced.png`, `gui-architecture.png`, `gui-path.png`, `gui-summary.png`, `gui-installing.png`,
`gui-done.png`, and `gui-keyboard.png`. The screenshots are development artifacts, not release assets.

## Remaining acceptance work

Automatic Explorer GUI selection, explicit `--gui`/`--tui` mode flags, console detachment decisions,
multiple-DPI/monitor testing, screen-reader/accessibility coverage and the full Sprint 6 acceptance matrix
remain pending. Active-install window-close cancellation has code-level cancellation/worker cleanup but
has not been manually interruption-tested through the graphical window; runtime recovery has its existing
fault-injection tests. No timing improvement for the existing shim-overhead target is claimed.
