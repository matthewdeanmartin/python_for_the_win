# Sprint 2 — System integration: preflight, PATH, shims, default, doctor

**Status:** implemented; local validation recorded below. Hosted CI and fresh-account manual UX pending.

**Goal:** a CLI-only user can run `pftw install` and then type `python`, `pip`, `python3.15` and `python3.16t` in a
new terminal, and it all works. PATH clashes are handled per spec D10, blockers refuse cleanly, and `pftw doctor`
explains the machine's state.

Prerequisite: sprint 1 merged. Read `spec/spec.md` sections 7, 9, 11.

## Tasks

### S2-1 Platform probes (`internal/platform`)
Windows implementations plus non-Windows stubs, all behind an `Info` interface with a fake for tests:
- `IsElevated()`: `TokenElevation` on the process token.
- `FreeBytes(path)`: `GetDiskFreeSpaceExW`.
- `OwnsConsole()`: `GetConsoleProcessList` count == 1 (used in sprint 3).
- `IsNetworkPath(path)`: UNC prefix or `GetDriveTypeW == DRIVE_REMOTE`.
- `SystemPath()` / `UserPath()`: raw **read** of `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment\Path`
  and `HKCU\Environment\Path`. This is a justified registry read (spec principle 9): the process `PATH` env var is
  the two merged and expanded, and nothing else records which part is user vs system or the unexpanded form. Comment
  it as such. Writing HKLM: never.
- No other registry reads in this sprint. If a task seems to need one, find a file/API/env alternative or raise it
  in the PR.
- Every probe returns "unknown/absent" rather than an error when a Windows feature is missing or behaves
  differently (spec principle 10). E.g. no WindowsApps dir, or an unrecognised reparse payload, means "unknown alias status" (info only), not verified absence. Unknown probe results remain
  unknown and appear in Advanced user info; do not claim successful verification.
- `IsAppExecLink(path)`: open with `FILE_FLAG_OPEN_REPARSE_POINT`, `FSCTL_GET_REPARSE_POINT`, check tag
  `IO_REPARSE_TAG_APPEXECLINK` (0x8000001B). If possible, parse the target package family name to tell the Store
  Python stub (`PythonSoftwareFoundation.Python.3.x_...`), the Install Manager (`PythonSoftwareFoundation.PythonManager`)
  and the "open the Store" stub (`Microsoft.DesktopAppInstaller`) apart.

**Accept:** unit tests via fakes. A Windows-only test asserts that `IsElevated()` is false in CI's normal runner
context (skip if CI runs elevated, and log it).

### S2-2 Embedded manifest
- `go-winres` config under `cmd/pftw/winres/` producing `rsrc_windows_*.syso` via `go generate`, with
  `asInvoker`, `longPathAware`, UTF-8 active code page, and version info. Commit the generated `.syso` files (they're
  object files from pure-Go tooling, not C; this is the standard pattern, and `go install` needs them).
  Document regeneration in the Makefile.

**Accept:** an exe named `pftw-setup.exe` launches without a UAC prompt on Windows (manual check, recorded in the PR).

### S2-3 PATH store (`internal/envpath`)
Per spec 11.2:
- `Store` interface. `RegistryStore{Key, ValueName}` for Windows. `MemStore` fake.
- `Plan` functions that are pure and unit-testable on every OS:
  - `Split(value) []string` / `Join([]string) string`: preserve empty segments and original spelling.
  - `AddFront(value, entries) string`, `Replace(value, old, new []string) string` (in place, keep the position),
    `Remove(value, entries) string`. Case-insensitive, trailing-backslash-insensitive matching. Never touch
    non-pftw segments.
- `Broadcast()`: `SendMessageTimeoutW(HWND_BROADCAST, WM_SETTINGCHANGE, 0, "Environment", SMTO_ABORTIFHUNG, 5000)`.
- Backup on first modification (state + `logs\path-backup-<ts>.txt`).
- Preserve `REG_EXPAND_SZ` vs `REG_SZ`. If the value doesn't exist, create it as `REG_EXPAND_SZ`.

**Accept:** extensive table tests on the pure functions (including the existing `%USERPROFILE%\...` tokens kept
unexpanded, a 4000-char PATH, duplicates, and trailing `;`). An integration test (tag `integration`) uses
`RegistryStore{Key: HKCU\Software\pftw-test\Environment}`, creates and deletes that key, and must never touch
the real `HKCU\Environment` in tests.

### S2-4 Clash detection (`internal/preflight`)
- `FindPythons(systemPath, userPath, lookup)`: for each expanded dir, look for `python.exe` and `python3.exe`
  (and `py.exe`, separately). Skip pftw-owned dirs (under root). Classify each hit as `System`, `User` or
  `StoreAlias` (via `IsAppExecLink`) and record the dir and its position.
- `PathDecision(hits, mode)` → one of `AddNoClash`, `MustChoose`, `OnlyNone`, plus explanation strings. Implements D10:
  any non-alias `System` hit → `OnlyNone`; else any `User` hit → `MustChoose`; else `AddNoClash`.
- Store alias blocking (spec 9, owner-confirmed): block **only on an actual alias conflict**, meaning pftw is
  adding itself to PATH and an alias stub would still resolve before pftw's `python` after the planned change (e.g.
  WindowsApps on system PATH). With `--path none` it's always info. Otherwise it's info. The fix text says to turn off the Python
  entries under "App execution aliases" and must not deep-link or launch Settings.
- Legacy `py.exe` on system PATH → warning.

### S2-5 Preflight runner
- A `Check` struct `{ID, Status: OK|Info|Warn|Block, Title, Detail, Fix}` and `Run(ctx, plan) []Check` covering every
  row of spec 9: elevation (`--allow-admin` bypass), disk, foreign target dir, feed reachability (the actual feed fetch; do not require separate HEAD support), store aliases, clashes, legacy py, network root.
- CLI prints checks compactly. Any Block → exit 3 with all blockers listed (not just the first).

### S2-6 Shims (`internal/shims` + shim mode in `cmd/pftw`)
Per spec 11.1:
- At the very top of `main()`: if the base name matches the shim regex (excluding `py`/`pyw` this sprint), call
  `shims.Run()` and exit. No cobra init, no logging setup beyond stderr.
- `shims.Run`: locate `pftw.json` in the parent of the exe's dir (read-only, no lock), resolve the alias via state
  (`python3.exe` → default; `python3.16.exe` → native-arch `3.16-*` install, preferring non-`t`; explicit
  `-64/-32/-arm64/t` forms → exact key), then CreateProcess on the target `python.exe` with the raw command-line
  tail from `GetCommandLineW` (parse off argv[0] with the same rules as `CommandLineToArgvW` for the first token
  only), inherited handles, a Job object with KILL_ON_JOB_CLOSE, ignored Ctrl+C/Break in the shim, and the child's
  exit code propagated.
- Non-Windows shim mode: `syscall.Exec` (used in tests and v2).
- `shims.Sync(root, state)`: ensure `shims\pftw.exe` is the current binary (copy self if different), then make
  the set of hard links match exactly what the state implies. Remove stale links, create missing ones, and fall back
  to copying if `os.Link` fails. Skip windowed aliases (sprint 4).
- Measure shim overhead (`python3.x -c pass` vs `python -c pass`, 20 runs) and record it in the PR. Target < 15 ms.

**Accept:** unit tests for alias resolution. A Windows integration test checks that `shims\python3.14.exe -c
"import sys; print(sys.argv)" "a b" "c\"d"` prints the exact argv, exit codes propagate (`-c "raise
SystemExit(7)"` → 7), and stdin piping works.

### S2-7 Wire PATH + shims into install/uninstall
- `--path first|none|auto` fully implemented (spec 7). `auto` + `MustChoose` + non-interactive → exit 4. In
  interactive CLI (stdin is a TTY, no `--yes`), ask with a numbered prompt and **no default** (empty input
  re-asks).
- Mode `first`: `path.added` = [default dir, default\Scripts, shims] prepended as a block, then broadcast.
- `install` runs `shims.Sync` after a successful install. `uninstall` removes its aliases. `--all` also removes the
  PATH entries, the shims dir, and the root (only pftw-known children; if foreign files remain, leave them and
  say so).
- Final message depends on the outcome: suggest bare `python` only when PATH selects pftw; otherwise
  show the installed interpreter path and usage. A system clash requires explicit consent, default No,
  to continue without PATH changes. `--path none` is explicit consent; `--yes` alone is not.

### S2-8 `pftw default SPEC`
- Must be installed. Update the state, `envpath.Replace` the old default dir + Scripts with the new ones (same
  position), re-sync shims (`python3.exe` target changes), broadcast.
- If path mode is `none`, update the state + shims only and print that PATH isn't managed.

### S2-9 `--portable`
- Containment only: no promise of relocation; do not rewrite Python, package launchers, or external venvs.
- Requires `--root`, forces `--path none`, sets `"portable": true`, keeps cache and logs inside the root, and never
  reads or writes `%LOCALAPPDATA%\pftw`. Later commands detect portable mode when run with `--root` pointing at it,
  or when the pftw.exe in use lives in `<root>\shims\`.

### S2-10 `pftw doctor` (read-only)
Implements spec 11.3 minus `--fix` and minus the "newer pftw" check. Uses the same `Check` struct and renders the
same way as preflight. `--json` output.
- Effective resolution: walk system PATH then user PATH (expanded) with `PATHEXT` rules to report what `python`,
  `python3`, `pip`, `py` resolve to, and whether each is pftw's.

**Accept:** golden-output tests over fake platform/state for each finding type.

## Out of scope
TUI wizard, upgrade, self-update, `py` launcher, windowed shims, bundle, `doctor --fix`.

## Definition of done
- CI green. Integration tests use only the scratch registry key.
- Manual walkthrough on a real Windows account, recorded in the PR as terminal output: clean machine → `pftw install`
  → new terminal `python --version` / `pip --version` / `python3.15 --version` (after `pftw install 3.15`) →
  `pftw default 3.15` → new terminal `python --version` shows 3.15 → `pftw uninstall --all` → the user PATH
  value is byte-identical to the backup (except where the user changed it meanwhile).

### Sprint-end build and owner handoff

- Use Go 1.27.2. Run `go vet ./...`, `go test ./...`, the sprint integration checks, and
  `CGO_ENABLED=0 go build -o dist/pftw.exe ./cmd/pftw`. Retain cross-platform CI checks.
- Report actual results and unrun/failed checks, and link the compiled Windows executable.
- Provide "What you can try now" with exact commands/clicks and expected results: install with PATH integration, open a new terminal, switch defaults, inspect `doctor`, and try a contained portable root.
- State what remains unavailable. Follow the handoff requirement in spec section 15.

### Implementation handoff

- Artifact: `dist/pftw.exe`, built with Go 1.27.2 and no cgo.
- Build, unit tests, vet, Staticcheck and Linux/macOS cross-builds pass locally.
- Real Windows lifecycle coverage: install 3.14 and 3.15, pip/venv, versioned shim quoting (spaces, quotes,
  trailing backslashes, empty arguments, Unicode), exit code 7, piped stdin, default switching, exact PATH
  restoration in the fake store, manager cleanup and portable-root discovery. Scratch registry storage tested.
- `list --available --show-unavailable` now explains legacy NuGet/checksum limitations exactly. The normal
  list selects usable binaries and supports fallback to an older usable patch.
- Full output is retained locally at `dist/sprint2-integration.txt`. No releases, pushes, tags or PRs were made.
- `doctor` was run read-only on the real account. Live account PATH writes and a fresh-account terminal
  walkthrough were deliberately not substituted for isolated integration tests; those manual checks remain.

- Shim benchmark: first 20-run mean overhead 17.66 ms; repeat 15.34 ms on this host. Functional checks pass,
  but the <15 ms performance target is not yet met. Keep this as a measured tuning item, not a passed check.
