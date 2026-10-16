# What pftw changes on your computer

## Files

Everything goes in one folder: `%LOCALAPPDATA%\pftw` (usually `C:\Users\you\AppData\Local\pftw`), or the
folder given with `--root`.

| Path | What it is |
|---|---|
| `pftw.json` | pftw's records: installed versions, the default, settings, and PATH entries it added |
| `pftw.lock` | held while a command is changing things, so two pftws can't collide |
| `versions\3.14-64\` | one Python per minor version, variant and architecture (`3.14t-arm64`, `3.13-32`, ...) |
| `shims\` | on PATH when pftw manages PATH: a copy of `pftw.exe` (and `pftww.exe`), plus `python3.14.exe`-style commands that start the matching Python |
| `cache\` | the downloaded python.org list (kept an hour) and verified Python ZIPs; safe to delete |
| `staging\`, `trash\` | temporary work during installs and upgrades; emptied afterwards (don't delete while an operation is pending) |
| `transaction.json`, `path-transaction.json` | present only during an operation, or after an interruption until recovery |
| `logs\pftw-YYYYMMDD.log` | what each command changed, including PATH before and after; one file per day of use, the newest 10 kept |
| `logs\path-backup-*.txt` | your user PATH exactly as it was before pftw first changed it |

If pftw itself crashes, details go to `%TEMP%\pftw-crash-*.log`.

## Registry and settings

- **Writes one value:** your user PATH (`HKEY_CURRENT_USER\Environment\Path`), keeping its type so `%VAR%`
  entries still expand, and changing nothing but pftw's own entries. Afterwards it tells running programs that
  the environment changed.
- **Reads** the system PATH (to detect conflicts) and, only after a failed direct connection, your Windows
  proxy setting.
- **Never writes:** Python registry entries (PEP 514), Add/Remove Programs entries, file associations,
  App Paths, Start Menu or desktop shortcuts, the system PATH, or anything under Program Files or `%APPDATA%`.
- **Never changes** Microsoft Store app execution aliases, other Pythons, or the `py` launcher.

## Programs it runs

Only the Python it's installing (to check it works), with a full path, never through `cmd.exe` or PowerShell.
`pftw run` and the versioned commands start Python the same way.

## Portable mode

With `--portable --root DIR`, everything stays in `DIR` and PATH is never touched.

## Network

HTTPS requests to python.org (or your configured source) for the build list and Python ZIPs, and to GitHub only
when you run `self-update`. No telemetry, no update checks in the background.
