# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- README reduced to installation and usage; advanced material moved to `docs/`, including a scripting guide.
- Executable offline bundles with read-only embedded sources and standalone installed managers.
- `pftw mirror SPEC... --out DIR` for self-hostable indexes and verified runtime ZIPs.
- Last-resort read-only Windows static-proxy fallback; PAC/WPAD remain unsupported.
- Inactive SignPath workflow draft and owner onboarding instructions.
- Global `--dry-run` / `--dry-mode`, including previews without cache/log/lock writes.
- Runner with explicit, shebang and inline-metadata selection; verified offline ZIP bundles.
- Conservative `doctor --fix`, Restart Manager process diagnostics and optional windowed Python shims.
- Wizard Previous navigation, separated button rows and browser-based Help.
- `pftw gui` explicitly opens a Windows Gio window using the shared wizard state machine.
- Verified official CPython ZIP installation, safe extraction, feed paging, explicit prerelease selection, portable roots, isolated Python/pip/venv checks, logs and transaction recovery (Sprint 1).
- PATH consent and recovery, versioned Python commands, default selection, doctor and managed uninstall; Windows resources, native architecture detection and Git Bash terminal support (Sprint 2).
- Terminal wizard with safe defaults, explicit PATH conflict decisions and collapsed advanced information.
- Recoverable same-minor upgrades, copied packages/Scripts, `--fresh`, and interruption tests.
- Checksum-verified self-update, Scoop detection, release packaging and generated CLI documentation (Sprint 3).
- Install pftw with pip: `python -m pip install pftw`, then `python -m pftw install`. Windows wheels carry the same executables; `python -m pftw` works even when Scripts is not on PATH.
- `make wheels VERSION=...` and `make wheels-check` build and test-install the PyPI wheels; release tags such as `v0.4.0-rc.1` become PEP 440 versions automatically.
- `make changelog-check` validates this changelog with keepachangelog-manager, also in CI.
- One-line install scripts: `irm .../install.ps1 | iex` (PowerShell) and `curl .../install.sh | bash` (Git Bash) fetch pftw from PyPI, verify its SHA-256 and start it; they never edit PATH themselves.

### Fixed
- Processes started by Python under `pftw run` or `python3.X` commands are no longer killed when Python exits.
- With `--json`, failures write `{"error":{"code","kind","message"}}` to stderr instead of prose.
- `install --json` (and `--target`) report `exe`, `version` and `key`.
- `doctor` detail lines no longer print mis-encoded characters.
- Unknown commands exit 2 (usage error) with a suggestion, as documented.
- Terminal advanced-info hints appear only when usable and are removed when the wizard exits.
- Installation roots below a symlinked folder (such as macOS /var or a redirected Windows profile folder) are accepted; links inside the root are still refused.
- `make lint` checks Windows, Linux and macOS code, so OS-specific files are linted locally as in CI.

### Changed
- Available versions select the most recent usable binary; unsupported entries explain exact reasons.
- `py` and `pyw` remain entirely outside pftw's namespace. The launcher is `pftw run`.
- `self-update` refuses for pip-installed copies (exit 3) and names the pip upgrade command.
- Known limitations moved from the changelog to the documentation index.

### Security
- GitHub Actions are upgraded and pinned to commit SHAs; checkouts drop persisted credentials, release builds skip the Go cache, and release permissions are scoped to the job. `make workflows-check` runs actionlint and zizmor.
