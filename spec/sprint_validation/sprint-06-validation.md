# Sprint 6 validation — 2026-10-15

Built with Go 1.27.2, CGO disabled. Artifact: `dist/pftw.exe` (13913600 bytes).

Passed locally:

- `make vet test lint build`; focused tests for target extraction/rollback, lookup, project selection and entry precedence.
- `make security`: no vulnerabilities found after updating x/net to v0.61.0, x/image to v0.47.0 and x/text
  to v0.43.0. The initial scan reported six reachable findings across those three modules.
- Real Python 3.15 target-only extraction, smoke execution, absent manager root/state/shims, temporary-file cleanup.
- Zero-write target preview, managed installation with PATH disabled, `which` executable lookup and ancestor
  `.python-version` selection for both preview and actual execution. User registry PATH unchanged.
- Forced `--gui` with redirected handles created a visible welcome window; screenshot checked and window closed
  cleanly with exit 0. No graphics needed for ordinary commands. Redirected no-command/TUI/conflicting flags
  returned usage errors; version/help retained exit 0.
- GoReleaser snapshot: Windows/Linux/macOS amd64/arm64 managers, Windows companions, archives and checksums.
  Local unpublished Scoop metadata refreshed. Nothing published or signed.

Full Windows lifecycle integration passed (`PFTW_INTEGRATION=1 go test -tags integration -timeout 10m ./...`; CLI package 225.707 seconds).

Evidence (ignored local build output): `dist/sprint6-checks-final.txt`, `sprint6-security-before.txt`,
`sprint6-security-after.txt`, `sprint6-targeted-final.txt`, `sprint6-smoke.txt`, `sprint6-gui.png`,
`sprint6-integration.txt`, and `sprint6-packaging.txt`.

Acceptance still pending:

- Human dogfooding and hosted CI; no third-party signing until the owner is satisfied.
- Explorer double-click, conhost/Windows Terminal/Git Bash interactive matrix; the auto-launch test here
  covers selection rules but does not substitute for those desktop hosts.
- Console versus GUI subsystem/AttachConsole comparison. This build deliberately retains the console
  subsystem and does not detach: a console may accompany an Explorer GUI launch.
- Narrator/NVDA/UI Automation, high contrast, complete focus audit and multi-monitor DPI. See
  [accessibility status](../../docs/accessibility.md). No new accessibility certification is claimed.
- Startup/shim performance targets have not been remeasured in this sprint.

Plain target installs are intentionally unmanaged: no pip bootstrapping, upgrade journal, package migration,
relocation guarantee, or automatic cleanup after process termination. Existing output folders are refused.
