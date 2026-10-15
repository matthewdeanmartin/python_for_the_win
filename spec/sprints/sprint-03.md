# Sprint 3 — Wizard and lifecycle: next-next-finish, upgrade, self-update, releases

**Goal:** a non-technical user downloads `pftw.exe`, double-clicks it, presses Enter a few times, and gets a working
latest final Python. Upgrades are recoverable across interruption. pftw can update itself. Release automation exists (ready for a human to tag).

Prerequisite: sprint 2 merged. Read `spec/spec.md` sections 8, 10 (upgrade), 12.

## Tasks

### S3-1 Wizard (`internal/tui`)
- Bubble Tea app implementing spec 8, steps 1–5. All logic calls the same packages the CLI uses (preflight,
  resolve, install, envpath). The TUI only renders and collects choices.
- Model the flow as an explicit state machine (`welcome → version → location → path → summary → installing →
  done | blocked | failed`) so it's unit-testable with `teatest` (or by driving `Update` directly with key msgs).
- PATH starts with Yes/No, default Yes. No bypasses conflict decisions. With Yes, a user clash requires
  explicit choice; a system clash asks to continue without PATH changes, default No. Test that Enter cannot
  authorize takeover or bypass this consent. `--yes` does not supply conflict consent.
- Default/latest selects a final release. Put version, architecture, prerelease and location overrides in
  Advanced options. Add collapsed Advanced user info for diagnostics and integration findings. Essential
  choices and actionable errors stay visible; do not show routine enterprise-integration warnings.
- Installing screen: one line per pipeline step with a spinner/✓/✗ and a download progress bar from the sprint 1
  progress callback.
- Failure screen: the error in plain words, the log file path, and *Quit*.
- Styling: plain, high-contrast, works in conhost (legacy console) and Windows Terminal. Test that `NO_COLOR` is
  respected and that it renders on a 80×25 window.
- Entry: `pftw` with no args, stdin and stdout are terminals → wizard. Otherwise → help.

**Accept:** state-machine unit tests cover every branch (blocked → recheck → ok, the three PATH variants, the
pre-release badge, install failure). A manual run on conhost and on Windows Terminal is recorded in the PR (an
asciinema cast or screenshots).

### S3-2 Double-click UX
- If `platform.OwnsConsole()`, wait for Enter on the done/failed/blocked screens before exiting. Also wrap
  panics: show "Something went wrong — details in <log>" and wait for Enter instead of a vanishing window.
- Set the console title to "Python for the Win".

### S3-3 In-place upgrade (`pftw upgrade`)
Implements the recoverable upgrade transaction in spec 10:
- `upgrade SPEC` and `upgrade --all` stay within each minor/variant/arch. Final installs remain on finals;
  explicitly selected development installs can track development builds through their final release.
- Stage and validate the replacement before modifying the live tree. Copy package data instead of moving it
  out of the active installation. `--fresh` omits both old site-packages and package Scripts.
- Journal every publication/commit phase durably. Retain a complete old tree until final-path Python, both pip
  command forms, and temporary venv checks pass and state commits. On failure restore old tree and state.
- On next startup recover unfinished transactions under the lock before any cleanup. Never delete a backup
  or staging tree referenced by an unfinished transaction. Detect sharing violations and report exit 7.
- Describe this as recoverable replacement, not an atomic multi-rename operation. No support for concurrent
  package mutation during upgrade.
- `install` on an existing key with a newer build calls upgrade.
- Keep the interpreter path stable. Do not discover, upgrade or repair external venvs; no ABI compatibility
  guarantee. Put prerelease compatibility details in Advanced user info.

**Accept:** fault injection at each filesystem/state step plus process-interruption/restart tests demonstrate
recovery to a complete old or new installation, with package data retained. Test final-path pip launchers.
A real Windows stable patch-upgrade test checks a pre-existing venv as a regression check for stable paths,
not as a universal promise. Local fixtures cover prerelease tracking and `--fresh` behavior.

### S3-4 Self-update (`pftw self-update`)
Per spec 12:
- GitHub Releases API (`/repos/matthewdeanmartin/python_for_the_win/releases/latest`, or the list with `--prerelease`).
  Unauthenticated. Handle rate-limit 403 with a clear message.
- Pick the `pftw_<ver>_windows_<nativearch>.zip` asset, verify it against `checksums.txt`, extract `pftw.exe`.
- Replace `shims\pftw.exe` with rename-to-`.old` then write. Re-sync the shim hard links. If the running exe is
  outside the root and writable, update it too. Delete `*.old` on the next start.
- Scoop detection (path contains `\scoop\apps\pftw\`) → refuse with "use `scoop update pftw`".
- `go install` builds (version == `dev`) → refuse with an explanation unless `--force`.
- Semver comparison. "Already up to date" → exit 0.

**Accept:** an httptest fake GitHub API covers newer/same/older/missing-asset/bad-checksum/rate-limited.

### S3-5 Release automation (prepared, not executed)
- `.goreleaser.yaml`: windows amd64 + arm64 (plus linux/darwin archives so CI artifacts prove cross-compiles,
  marked "unsupported for installing" in the release notes template), `CGO_ENABLED=0`, `-trimpath`, ldflags
  version/commit/date, zip + raw exe, `checksums.txt`.
- `.github/workflows/release.yml`: on tag `v*`, run goreleaser and `actions/attest-build-provenance` on the
  artifacts.
- `bucket/pftw.json`: a Scoop manifest with `bin`, `checkver: github`, and `autoupdate` URLs + hash from
  checksums.txt. This repo is the bucket (spec D15). Verify locally with Scoop's `checkver.ps1`/`checkurls.ps1`
  **only if** the human runs them. Don't add PowerShell to CI or the Makefile. A JSON-schema test of the manifest
  in Go is enough for CI. Have goreleaser's `scoops:` section update `bucket/pftw.json` in this repo (not a separate
  repo) when the human cuts a release.
- `goreleaser check` and `goreleaser release --snapshot --clean` run in CI (no publish).
- **Do not create tags or releases.** The PR description lists the exact human steps for the first release.

### S3-6 Docs
- Rewrite `README.md` user-first: a 3-line "Download pftw.exe → double-click → Enter, Enter, Enter" section on top,
  then the SmartScreen "More info → Run anyway" note (unsigned, with the reason), then the CLI reference
  (generated from cobra: `pftw docs` hidden command, or `go run ./internal/cli/gendocs` into `docs/cli.md`). Keep
  the owner's wish list and philosophy sections, moved lower down.
- `CHANGELOG.md` in Keep a Changelog format with an `[Unreleased]` section covering sprints 1–3.

### S3-7 Spec upkeep
- Update `spec/spec.md` with any deviations. Move finished roadmap rows to "done". Draft `spec/sprints/sprint-04.md`
  from the backlog row (`pftw run` per spec 7.2, bundle zip, doctor --fix, Restart Manager, windowed shims) for
  the owner to review.
- Downgrade the legacy-`py.exe` preflight/doctor finding from warning to info, with the new wording in spec 9 (pftw
  no longer ships a `py`, so there's no conflict).

## Out of scope
`pftw run`, windowed shims, `bundle`, `doctor --fix`, signing, Linux installs.

## Definition of done
- CI green including the goreleaser snapshot job.
- Manual end-to-end on a fresh Windows user account (no prior Python), recorded in the PR: download a snapshot
  `pftw.exe` → double-click → Enter ×N → finish → new terminal `python --version` = the newest eligible final release, `pip install
  requests` works, `python -m venv .venv` works.
- A second manual run on an account that has another Python on user PATH shows the forced choice with no default.

### Sprint-end build and owner handoff

- Use Go 1.27.2. Run `go vet ./...`, `go test ./...`, the sprint integration checks, and
  `CGO_ENABLED=0 go build -o dist/pftw.exe ./cmd/pftw`. Retain cross-platform CI checks.
- Report actual results and unrun/failed checks, and link the compiled Windows executable.
- Provide "What you can try now" with exact commands/clicks and expected results: double-click the TUI wizard, try safe defaults and conflict decisions, then exercise a same-minor upgrade.
- State what remains unavailable. Follow the handoff requirement in spec section 15.
