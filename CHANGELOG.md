# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Install wizard: double-click `pftw.exe` and press Enter until done to get the newest official Python with pip, for your user only. No administrator rights, Microsoft Store, MSI, registry entries (other than your PATH) or Program Files. Works in the terminal or as a graphical window (`pftw gui`).
- Official, verified Python: downloads python.org's own Windows builds and checks each download's SHA-256 before use. Each install is checked to run Python, pip and venv before it's reported as finished.
- Several Pythons side by side: `pftw install 3.14`, exact versions, pre-releases on request, free-threaded builds, and 64-bit, ARM64 or 32-bit. `pftw list --available` shows what can be installed.
- PATH that asks first: Python goes on PATH by default, but pftw asks before taking over from another Python and never changes the system PATH. Every install also gets versioned commands such as `python3.14`. `pftw default` chooses which version `python` means.
- `pftw run`: run a script with the right Python, chosen from `--python`, the script's shebang, its inline `requires-python` metadata, a `.python-version` file, or the default. `pftw which` prints an interpreter's path.
- Upgrades and removal: `pftw upgrade` moves to the newest patch release, keeping installed packages, and rolls back if anything fails. `pftw uninstall` removes only what pftw created.
- `pftw doctor` explains which `python`, `pip` and `py` a new terminal will run, and `pftw doctor --fix` restores missing pftw commands and PATH entries.
- Portable and plain installs: keep everything in one folder with `--portable --root DIR`, or extract a plain Python folder with `--target DIR`.
- Offline and company networks: offline installers as a ZIP or a single exe (`pftw bundle`), self-hosted mirrors (`pftw mirror`), a configurable download source, and proxy support, including your Windows proxy setting.
- Preview anything: every command accepts `--dry-run` to show what it would change without changing it.
- Scripting and automation: `--json` output (including errors), documented exit codes, and no prompts when there's no terminal.
- Ways to get pftw: a single exe, Scoop, `pip install pftw` (then `python -m pftw`), one-line install scripts for PowerShell and Git Bash, or `go install`. `pftw self-update` updates a downloaded copy.
- Documentation site with troubleshooting for the messages pftw shows and a list of exactly what pftw changes on your computer. Help in the installer opens it, and failure screens link straight to troubleshooting.
