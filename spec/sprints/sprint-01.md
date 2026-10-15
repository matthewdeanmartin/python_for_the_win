# Sprint 1 — Foundations: download, verify, extract, run

**Status:** implemented; local verification passed. Hosted CI is configured but not run here.

**Goal:** `pftw install 3.14 --path none --yes` puts a working, sha256-verified CPython in
`<root>\versions\3.14-64\` with pip. `pftw list` shows it and `pftw uninstall 3.14` removes it. No PATH, no shims,
no TUI yet.

Read `spec/spec.md` first. Section numbers below refer to it.

## Ground rules

- Git Bash for all shell work. `CGO_ENABLED=0` everywhere. Go 1.27.2.
- Don't push tags, publish releases, or post anywhere external (see `C:\github\AGENTS.md`).
- Prefer stable 3.14/3.15 for real-network tests. 3.16 is only at alpha (3.16.0a1 as of 2026-10-15).

## Tasks

### S1-1 Repo skeleton
- `go.mod`: module `github.com/matthewdeanmartin/python_for_the_win`, `go 1.27.2`.
- Layout:
  ```
  cmd/pftw/main.go
  internal/cli/        cobra commands (thin; logic lives in packages below)
  internal/feed/       index parsing, paging, caching
  internal/pyver/      PEP 440-ish version parse/compare (only what the feed needs: X.Y.Z[aN|bN|rcN][t])
  internal/resolve/    spec normalisation + entry selection (spec 4.1)
  internal/download/   HTTP fetch, retry, sha256
  internal/archive/    hardened unzip
  internal/layout/     root/dir path helpers
  internal/state/      pftw.json + lock
  internal/platform/   native arch, (stubs for elevation/disk/console: sprint 2)
  internal/proc/       Runner interface for child processes (smoke test, ensurepip)
  internal/install/    the pipeline (spec 10, install path only)
  internal/logx/       file logger under <root>\logs
  testdata/
  ```
- `LICENSE` (MIT, "Copyright (c) 2026 Matthew Dean Martin").
- `Makefile` (Git Bash compatible): `build`, `test`, `vet`, `lint`, `integration`.
- `.gitignore` already covers `*.exe`. Add `/dist/`.

**Accept:** `make build test vet` passes on Windows. `CGO_ENABLED=0 go build ./...` passes with `GOOS=linux` and
`GOOS=darwin` too.

### S1-2 CI
- `.github/workflows/ci.yml`: matrix windows-latest, ubuntu-latest, macos-latest. Steps: setup-go (from go.mod),
  `go vet`, `go test ./...`, `CGO_ENABLED=0 go build ./...`, staticcheck (or golangci-lint with a small config).
- Separate job `integration` on windows-latest only: `PFTW_INTEGRATION=1 go test -tags integration ./...`.
- `no-shell` test (`internal/lint_test.go` or similar): scan non-test `.go` files for the literals `cmd.exe`,
  `cmd /c`, `powershell`, `pwsh` (case-insensitive), and fail if found.

**Accept:** a PR shows all jobs green.

### S1-3 Version parsing (`internal/pyver`)
- Parse `3`, `3.16`, `3.16.0`, `3.16.0a1`, `3.16.0rc2`, `3.16t`, `3.16.0a1t`. Compare with a < b < rc < final.
- Table tests including the ordering 3.15.0b3 < 3.15.0b4 < 3.15.0rc1 < 3.15.0rc3 < 3.15.0 < 3.15.1.

### S1-4 Feed client (`internal/feed`)
- Types for the entry schema (spec 4). Unknown fields ignored. `schema > 1` entries skipped.
- `Load(ctx, source)`, where source is the default URL, an http(s) URL, a local file, or a dir containing
  `index.json`. Lazy `next` paging: an iterator that fetches the next page only when asked.
- First-page cache keyed by source (`<root>\cache\index-<sha256(source)[:12]>.json`) with a 1 h TTL. `Refresh`
  option bypasses it.
- Source selection per spec 4.0: `ResolveSource(flag, env PFTW_SOURCE, exeDir, state.config.source)` returns the
  effective source and where it came from (for display). Reject `http://` unless `--allow-insecure-source`.
  Table-test the full precedence order.
- Fixture: save a **trimmed real snapshot** (all 3.16, 3.15.0 + one rc, 3.14 latest, for all three arches, the `t`
  variants, plus one PythonEmbed and one PythonTest entry to prove they're filtered) as
  `testdata/index-windows.sample.json`. Add a second fixture page to test `next`.
- Filter to `company == "PythonCore"`.

**Accept:** unit tests with `httptest.Server` cover paging, caching, TTL expiry, malformed JSON (clear error), and
local-dir sources with relative URLs.

### S1-5 Resolution (`internal/resolve`)
- Implement spec 4.1 exactly: native-arch suffix, `install-for` matching, stable-only default/`latest`, explicit-minor prerelease fallback with
  a `Prerelease bool` flag on the result, `--prerelease` override, `--arch` and `--freethreaded` flags mapped to
  suffix/`t`.
- `platform.NativeArch()`: Windows uses `IsWow64Process2` (fall back to `GetNativeSystemInfo`). Other OSes map
  from `runtime.GOARCH` (used only for tests and later v2).
- Install key derivation: tag with `-dev` removed (`3.16-dev-64` → `3.16-64`).

**Accept:** table tests against the fixture: `3.16` → 3.16.0a1 with prerelease=true; `3.15` → 3.15.0;
`3.15 --prerelease` → still 3.15.0 (stable is newer than its rcs); `3.16t --arch arm64` → `pythoncore-3.16t-arm64`;
`3`, `latest`, omitted spec → newest eligible final (error if none, never implicit alpha); `3.16.0a1` → exact pin; `3.9` → "not found" error listing the nearest available minors.
`list --available` default view includes 3.16.0a1 marked `pre` (spec 4.1 rule 5).
- The feed parser must not fail on a missing optional field (`alias`, `hash`, `display-name`, `next`). A missing
  `hash` makes the entry *uninstallable* (integrity is mandatory), but it's still listed with a note. Spec
  principle 10.

### S1-6 Download (`internal/download`)
- Stream to `cache\<filename>.part` while hashing, then rename. If the cached file already exists and its hash
  matches, skip the download.
- Retry ×3 with exponential backoff on network errors and 5xx. No retry on 4xx. `http.ProxyFromEnvironment`.
- Progress callback `func(done, total int64)` for future TUI use. CLI prints a simple percentage line when stderr is
  a terminal.
- On hash mismatch: delete the file and return a typed `ErrIntegrity`.

**Accept:** httptest tests cover success, resume-from-cache, mismatch, a 500-then-OK retry, and a 404.

### S1-7 Hardened unzip (`internal/archive`)
- Extract to a target dir. Reject (`ErrIntegrity`) absolute paths, `..`, drive letters, `:` (ADS), symlink entries,
  and reserved Windows names in any segment (CON, PRN, AUX, NUL, COM1–9, LPT1–9, case-insensitive, with or
  without an extension).
- Preserve file modes where meaningful. Create dirs as needed.

**Accept:** unit tests with crafted malicious zips (generated in-test with `archive/zip`) cover each rejection,
plus a normal zip round-trip.

### S1-8 Layout + state (`internal/layout`, `internal/state`)
- Default root: `%LOCALAPPDATA%\pftw` on Windows. On other OSes use `os.UserCacheDir()/pftw` (tests only for now).
  `--root` overrides it.
- `pftw.json` schema per spec 6. Atomic write (tmp + rename). Unknown fields preserved (round-trip via
  `map[string]json.RawMessage` for unknown keys).
- `pftw.lock`: an exclusive lock (Windows: `LockFileEx` via x/sys; Unix: `flock`). It's taken by mutating commands.
  A second process gets a clear "another pftw is running" error.
- Startup recovery runs under the mutation lock before housekeeping. Never delete transaction-referenced
  staging or backup data. Journal publication and state changes, including new installs.

**Accept:** tests cover round-trip with unknown fields, concurrent-lock refusal (two goroutines/processes), and
crash-safety (a tmp file left behind is ignored and replaced).

### S1-9 Process runner + install pipeline (`internal/proc`, `internal/install`)
- `proc.Runner` interface: `Run(ctx, exe string, args []string, opts{Dir, Timeout, Env}) (stdout, stderr, code, err)`
  with a real implementation (`os/exec`, absolute exe path only, never via a shell) and a fake for tests.
- Pipeline (spec 10 install path, steps 1–7 minus shims/PATH): resolve → download → verify → extract to
  `staging\<key>-<rand>` → smoke test (cwd = fresh empty temp dir, `-I`, 30 s timeout, version check) →
  ensurepip if needed → journaled publication at the final path → repair pip launchers → verify
  both pip command forms and a temporary venv → commit state. Pip failure prevents success unless `--no-pip`.
- First successful install sets `default` in state (PATH effects come in sprint 2).
- Existing key with the same version → "already installed", exit 0. A newer patch → print "upgrade available — run
  `pftw upgrade` (coming in a later version)" and exit 0 (upgrade is sprint 3).
- A non-pftw dir present at the target → refuse, exit 3.

### S1-10 CLI commands (`internal/cli`)
- `pftw install [SPEC] --path none --yes [--arch] [--freethreaded] [--prerelease] [--source] [--root] [--no-pip]`.
  In this sprint `--path` accepts only `none`. Anything else → exit 2 with "PATH support arrives in a later
  version".
- `pftw list` (installed; default marked `*`) and `pftw list --available [--prerelease] [--all-arch]` (feed,
  grouped by minor). `--json` for both.
- `pftw uninstall SPEC|--all`: removes dirs + state entries (rename to `trash\`, then delete). If the default is
  removed, pick the newest stable native-arch install as the new default (or none).
- `pftw config list|get|set|unset` (spec 7) for the keys `source` and `proxy`. `set source` fetches and parses
  the index before saving, and prints the "whoever controls the source controls what gets installed" warning for
  non-python.org sources. `proxy` is stored now and used by the downloader (`--proxy` flag > `HTTPS_PROXY` env >
  config).
- Each install records its `source` in state. `list` shows the effective source when it isn't the default.
- `pftw version`. Version info is injected via `-ldflags -X`, with defaults `dev`/`unknown`.
- Exit codes per spec 7.1 (the ones reachable this sprint: 0, 1, 2, 3, 5, 6).
- `pftw` with no args prints help (the wizard is sprint 3).

### S1-11 Integration test (Windows, real network)
- Build tag `integration`, gated on `PFTW_INTEGRATION=1`. Temp root.
  `install 3.14 --path none --yes` → assert `versions\3.14-64\python.exe -c "import sys"` works, both `python -m pip` and `Scripts/pip.exe` work at the final path, a temporary venv runs,
  `list --json` shows it, `uninstall 3.14` removes it, and the root has no leftover `staging\`.

## Out of scope (later sprints)
PATH, shims, preflight beyond "foreign dir at target", elevation checks, TUI, upgrade, self-update, bundle, py launcher.

## Definition of done
- All S1 tasks' acceptance criteria met. CI green on all three OSes plus the Windows integration job.
- Manual check by the implementer on a real Windows box: `go run ./cmd/pftw install 3.16 --path none --yes` installs
  3.16.0a1 with a visible pre-release warning.
- `spec/spec.md` updated if anything deviated.

### Sprint-end build and owner handoff

- Use Go 1.27.2. Run `go vet ./...`, `go test ./...`, the sprint integration checks, and
  `CGO_ENABLED=0 go build -o dist/pftw.exe ./cmd/pftw`. Retain cross-platform CI checks.
- Report actual results and unrun/failed checks, and link the compiled Windows executable.
- Provide "What you can try now" with exact commands/clicks and expected results: install a final release with `--path none`, inspect `list`, run the installed Python and pip by full path, then uninstall.
- State what remains unavailable. Follow the handoff requirement in spec section 15.

### Implementation handoff

- Artifact: `dist/pftw.exe` (Go 1.27.2, Windows amd64, pure Go).
- Local checks: unit tests, vet, Staticcheck, and Windows/Linux/macOS compilation.
- Real Windows integration: Python 3.14.8 with pip module, final-path pip launcher, temporary venv, and uninstall.
- Explicit development-version CLI smoke test: Python 3.16.0a1, with a visible development-release message.
- Deviations are recorded in spec section 17: all-page resolution for stable correctness, content-addressed
  payload cache, final-location pip setup, and runtime-only uninstall in this phase.
- No PR, tag, push, release, or hosted CI run was performed. Hosted CI acceptance remains pending.
