# Changelog

All notable changes are documented here, following Keep a Changelog.

## [Unreleased]

### Added
- Executable offline bundles with read-only embedded sources and standalone installed managers.
- `pftw mirror SPEC... --out DIR` for self-hostable indexes and verified runtime ZIPs.
- Last-resort read-only Windows static-proxy fallback; PAC/WPAD remain unsupported.
- Inactive SignPath workflow draft and owner onboarding instructions.
- Global `--dry-run` / `--dry-mode`, including previews without cache/log/lock writes.
- Runner with explicit, shebang and inline-metadata selection; verified offline ZIP bundles.
- Conservative `doctor --fix`, Restart Manager process diagnostics and optional windowed Python shims.
- Wizard Previous navigation, separated button rows and browser-based Help.
- `pftw gui` explicitly opens a Windows Gio window using the shared wizard state machine.
- Verified official CPython ZIP installation, safe extraction, feed paging, explicit prerelease selection,
  portable roots, isolated Python/pip/venv checks, logs and transaction recovery (Sprint 1).
- PATH consent and recovery, versioned Python commands, default selection, doctor and managed uninstall;
  Windows resources, native architecture detection and Git Bash terminal support (Sprint 2).
- Terminal wizard with safe defaults, explicit PATH conflict decisions and collapsed advanced information.
- Recoverable same-minor upgrades, copied packages/Scripts, `--fresh`, and interruption tests.
- Checksum-verified self-update, Scoop detection, release packaging and generated CLI documentation (Sprint 3).

### Fixed
- Terminal advanced-info hints appear only when usable and are removed when the wizard exits.

### Changed
- Available versions select the most recent usable binary; unsupported entries explain exact reasons.
- `py` and `pyw` remain entirely outside pftw's namespace. The launcher is `pftw run`.

### Known limitations
- No automatic desktop GUI selection, verified screen-reader support, signing or Linux/macOS runtime installation yet.
- First release and hosted CI/manual console acceptance are human follow-up work; no release is published here.
