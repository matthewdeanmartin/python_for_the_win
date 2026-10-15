# Sprint 3 local validation — 2026-10-15

Implementation is available for owner testing. Hosted CI and the separate manual acceptance runs below
remain pending; this is not a claim that every release acceptance criterion has been met.

## Automated checks

- Go 1.27.2, CGO disabled: `go vet ./...`, `go test ./...`, pinned Staticcheck and Windows build.
- Upgrade fault tests cover backup/publication renames, state-save failure, every Python/pip/venv check,
  and process exit after prepared, backed-up, published, validated and committed journal checkpoints.
  Recovery is idempotent; old package data survives rollback; `--fresh` omits old packages and Scripts.
- Wizard model tests cover PATH No bypass, clean PATH, explicit user choice, default-No system/unknown
  decisions, blocked/recheck, prerelease display, failure, and plain 80-column/25-row output with NO_COLOR.
- Self-update fake HTTP tests cover newer/same/older releases, prereleases, missing assets, checksum failure,
  rate limits and development-build refusal. A Windows subprocess test replaces its own running manager
  and resynchronizes the versioned command hardlinks. An empty-runtime manager can also be updated.
- Scoop manifest validates against the vendored upstream JSON schema; checkver/autoupdate enrichment is tested.
- GoReleaser v2.18.0 `check` and `release --snapshot --clean` pass locally, producing Windows amd64/arm64
  ZIPs and raw EXEs plus Linux/macOS amd64/arm64 ZIPs and checksums. No publishing was performed.

The final development executable is approximately 8.6 MiB, below the 15 MB budget.
The full integration suite initially encountered an access-denied directory rename during the final
portable uninstall. Runtime renames now retry only sharing/access-denied errors for at most 1.55 seconds;
persistent failures retain recovery data and report exit 7. Unit tests cover successful delayed release,
persistent locks and non-retryable failures. The initial log is retained as
`dist/sprint3-integration-first-run.txt`.

The subsequent full Windows integration run passed (CLI integration package: 196.272 seconds).
Its output arrived late; an extra verbose diagnostic rerun was stopped after the successful result arrived,
and that duplicate run's isolated files were removed. It is not counted as an additional successful run.

## Real Windows checks

- Installed official Python 3.14.7, created an external venv and a site-packages marker, upgraded to 3.14.8,
  imported the retained marker, ran final-path `pip.exe`, and executed the pre-existing venv on 3.14.8.
  This checks stable base paths, not universal external-venv or ABI compatibility.
- Ran the Bubble Tea wizard through a real interactive Windows PTY using Git Bash. Selected the default
  Python 3.15.0, kept the demo folder, selected PATH No, confirmed the summary and installed successfully.
  Observed the progress bar, step status and final Enter-to-finish screen. Checked pip afterward and removed
  the isolated demo using `uninstall --all`. The actual user PATH was never modified by these checks.
- Full integration suite retains existing install/shim quoting/stdin/exit/default/uninstall/portable checks.

Local logs: `dist/sprint3-checks.txt`, `dist/sprint3-integration.txt`, `dist/sprint3-upgrade.txt`,
and `dist/sprint3-packaging.txt`. They are development artifacts, excluded from source control.

## Not yet exercised

- Separate Explorer double-click, legacy conhost and Windows Terminal screenshot walkthroughs.
- Fresh Windows user account: default PATH integration, a new terminal, published-wheel requests install,
  and venv creation; second account with a user-PATH Python requiring an explicit choice.
- Hosted CI runners, actual GitHub release/self-update round trip, Scoop installation and signing.
- Exhaustive low-level disk/power-loss failures; journal tests simulate process termination and injected IO
  failures, not physical power failure. Concurrent pip writes during upgrade remain unsupported.
- Shim timing target remains open from Sprint 2; this sprint does not claim a performance improvement.

## Try this build

Double-click `dist/pftw.exe`, or run `./dist/pftw.exe` from a terminal. Enter accepts safe defaults;
A opens advanced options on version/location screens, I expands Advanced user info. PATH defaults Yes;
No bypasses conflict decisions. Another user-PATH Python requires 1 or 2; Enter alone cannot take over.
A system-PATH clash offers continuation without PATH changes and defaults No.

For a contained same-minor upgrade demo (assuming those official historical ZIPs remain available):

```bash
./dist/pftw.exe install 3.14.7 --portable --root C:/pftw-upgrade-demo
./dist/pftw.exe upgrade 3.14 --root C:/pftw-upgrade-demo
C:/pftw-upgrade-demo/versions/3.14-64/python.exe --version
./dist/pftw.exe uninstall --all --root C:/pftw-upgrade-demo
```

The upgrade selects the newest usable 3.14 final, retaining the same base path. On an already-current
installation it reports Already installed. `upgrade --all` updates installed minors without crossing minors.
`--fresh` deliberately omits old packages and Scripts when a newer runtime is installed.

`self-update` on this dev build explains its refusal; `--force` permits release lookup, but no release is
published by this sprint. Gio GUI, `pftw run`, windowed shims, bundle creation and doctor repair remain future work.
