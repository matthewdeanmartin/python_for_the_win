# Sprint 4 validation — 2026-10-15

Built with Go 1.27.2, CGO disabled. No release, tag, commit or upstream message was published.

## Delivered

- GUI Previous/Next row, whitespace between form/navigation/Help-Quit groups, and Help opening the repository documentation. Previous revokes earlier PATH takeover consent. The terminal wizard also accepts B for Previous.
- Global `--dry-run`, with `--dry-mode` as an alias. Installation, upgrade, uninstall, default, settings and self-update have plans; bundle and doctor repairs have their own previews. Read-only commands run with cache writes suppressed. Runner previews do not execute Python. Wizard previews use `install --dry-run`.
- `pftw run`: explicit interpreter/version, Python shebang, inline TOML metadata or default selection; explicit missing-version installation without PATH changes; preserved arguments, standard handles and child exit code.
- Verified offline bundles with relative payload references and original SHA-256 checksums. Optional pftww.exe is included when available beside the manager.
- Conservative `doctor --fix`: restores missing aliases with ownership records and missing previously owned PATH entries. Existing files, unknown runtimes and ambiguous state are preserved for inspection.
- Restart Manager process naming on sharing/access failures; no automatic process termination.
- Windows GUI-subsystem companion and versioned pythonw aliases. No py/pyw ownership or associations.

## Checks performed

- `make vet test lint build` passed with Go 1.27.2 and pinned Staticcheck.
- Full `PFTW_INTEGRATION=1 go test -tags integration ./...` passed; CLI lifecycle/upgrade package completed in 143.994 seconds.
- Snapshot packaging passed for six manager targets (Windows/Linux/macOS amd64/arm64), plus two Windows windowed companions. Windows ZIPs contain both executables.
- Filesystem snapshots verify dry-mode mutation previews and read-only commands do not alter an existing root. Another test verifies a preview does not create a missing root.
- Regression tests cover script-argument boundaries, runner dry-run suppression, selection precedence, malformed metadata, supported version constraints, bundle checksum rejection and preservation of foreign files during repair.
- A Windows Restart Manager test identifies the running test executable by PID.
- Built a real Python 3.15.0 bundle; extracted it and installed through the adjacent index with HTTPS_PROXY pointing to an unreachable local port. Verified Python/pip/venv checks and windowed companion installation.
- Ran Python through the runner with whitespace, quotes, trailing backslashes, empty strings, Unicode and a script-owned --dry-run argument. Verified redirected stdin and exit status 23. Verified metadata selection and dry-run without executing the supplied code.
- Explicit runner `--install` installed a missing Python 3.15 from the offline source and executed the requested code; the isolated root was then uninstalled.
- Launched pythonw3.15.exe without a console and checked GetConsoleWindow returned zero, with argv and working-directory propagation. Repaired a deliberately removed owned shim, then uninstalled the isolated runtime. The real user PATH was unchanged.
- Inspected actual GUI screenshots of the revised welcome, version, location and PATH pages. Verified the requested spacing and same-row navigation, then clicked Previous to return from PATH to location. Closed the test window.

Local evidence: `dist/sprint4-checks.txt`, `dist/sprint4-integration.txt`, `dist/sprint4-packaging.txt`,
`dist/sprint4-smoke.txt`, `dist/sprint4-run-install.txt`, and `dist/gui-path.png` / `dist/gui-previous.png`.
The initial smoke run exposed a local Windows path/URL classification bug and a runner flag parsing bug;
both were fixed and received regression coverage before the successful smoke run.

## Limits and remaining acceptance

- Narrator/NVDA, Windows UI Automation, OS high-contrast themes and multi-monitor DPI acceptance are not verified. See [accessibility status](../../docs/accessibility.md).
- Ctrl+C uses the existing direct-process/job handling, but an interactive Ctrl+C test was not added in this sprint. The GUI close-during-install manual test remains pending.
- Doctor repair deliberately does not reconstruct corrupt state, overwrite existing aliases, or delete missing-runtime records. Those cases require inspection.
- Python requirement parsing rejects unsupported syntax rather than implementing every PEP 440 extension (for example development/post/local/epoch versions).
- Automatic Explorer GUI selection, signing and timing improvements for the earlier shim-overhead target remain pending.
- `pftww.exe` is optional: keep it beside pftw.exe when installing to enable windowed aliases. A standalone manager/go-install build without the companion still installs console aliases.

Technical references used: [Restart Manager API](https://learn.microsoft.com/en-us/windows/win32/api/restartmanager/nf-restartmanager-rmgetlist),
[TOML decoder](https://pkg.go.dev/github.com/pelletier/go-toml/v2), and
[GoReleaser archive binary counts](https://goreleaser.com/resources/errors/multiple-binaries-archive/).
