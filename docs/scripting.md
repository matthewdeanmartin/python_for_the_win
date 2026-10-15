# Scripting and automation

pftw is a plain executable: no shell, PowerShell or interactive session is needed. Call it from any
language with ordinary process APIs (`subprocess.run`, `execFile`, `Command::new`, `os/exec`, ...).

## Rules of thumb

- Add `--json` to get one JSON object on stdout. Progress messages go to stderr.
- Use the exit code to tell success from failure; use JSON fields for details.
- Never parse the human text; it can change.
- Without a terminal, pftw never waits for input. Questions that need a human fail with exit code 4
  instead (for example a PATH clash: pass `--path first` or `--path none`).
- Use `--root DIR` for an isolated install that won't disturb the user's own setup.
- Preview first with `--dry-run --json`.

## Useful calls

```bash
pftw install 3.15 --path none --json
# {"exe":"C:\\...\\versions\\3.15-64\\python.exe","key":"3.15-64","message":"...","root":"...","version":"3.15.x"}

pftw which 3.15 --json
# {"exe":"C:\\...\\versions\\3.15-64\\python.exe"}

pftw list --json
pftw list --available --json
pftw run --python 3.15 --dry-run --json -c "print(1)"
# {"argv":[...],"dry_run":true,"interpreter":"...","reason":"flag"}
```

The best way to run Python from a script is usually to ask `which` once, then start that interpreter directly.

## Errors

With `--json`, a failure writes a single object to **stderr** and leaves stdout empty:

```json
{"error":{"code":9,"kind":"not_installed","message":"Python 3.12 isn't installed: use pftw install with the desired version"}}
```

## Exit codes

| Code | `kind` | Meaning |
|---|---|---|
| 0 | | Success, or nothing to do |
| 1 | `unexpected` | Unexpected error |
| 2 | `usage` | Bad command, flag or argument |
| 3 | `preflight` | Blocked by a check (for example running as administrator) |
| 4 | `decision_required` | A human decision is needed; pass an explicit flag |
| 5 | `network` | Network or download failure |
| 6 | `integrity` | Hash mismatch, corrupt or unsafe archive |
| 7 | `files_in_use` | A Python process is using the target |
| 8 | `unsupported_platform` | Not supported on this OS |
| 9 | `not_installed` | The selected Python isn't installed |

`pftw run` and the `python3.X` commands return **Python's** exit code once Python has started. To tell
pftw failures apart from your script's, run `pftw run --dry-run` first.

## Processes started by Python

Python launched through `pftw run` or a `python3.X` command is stopped if pftw itself is killed. Processes
that Python starts are independent, so detached servers and daemons keep running after Python exits.
