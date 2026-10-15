# Sprint 5 validation — 2026-10-15

Local build: Go 1.27.2, CGO disabled. No commit, tag, push, release, SignPath application or signing request
was made. Existing staged files were preserved; new edits were not staged on the owner's behalf.

Delivered executable offline bundles, mirrors, static Windows proxy fallback and a local signing handoff.
Signing itself is not activated; see [signing.md](../../docs/signing.md).

## Validation performed

- Local vet, unit tests, Staticcheck and both Windows builds passed (`dist/sprint5-checks.txt`).
- The full Windows integration suite passed (`dist/sprint5-integration.txt`). This includes real runtime
  lifecycle and patch-upgrade checks. Targeted tests also passed after the integrity exit-code fix.
- Snapshot packaging passed for six manager targets and two Windows companion targets
  (`dist/sprint5-packaging.txt`). Hosted CI was not triggered.
- Unit tests cover overlay bounds/integrity, unsafe and duplicate ZIP names, member-path escapes,
  executable bundle creation, zero-write embedded-source previews, and integrity exit code 6.
- Mirror tests cover original hashes, deduplication, overwrite refusal, no installation-root writes, and
  serving the generated index/payload through a local HTTP server with explicit insecure-test opt-in.
- Proxy tests cover direct success and HTTP errors without registry reads, fallback on connection failure,
  explicit/environment precedence, NO_PROXY disabling fallback, bypass patterns and per-protocol settings.
  Settings are injected; the account's proxy settings were not changed.
- Real smoke: created a Python 3.15.0 mirror and executable bundle, then installed through the executable
  with an unusable HTTPS proxy and no companion beside it. Verified Python/pip and its embedded windowed
  companion. The installed manager matched the standalone build byte-for-byte, without the overlay.
  The isolated runtime was uninstalled; user PATH was unchanged (`dist/sprint5-smoke.txt`).
- Executable-bundle `list --available --dry-mode` and `install --dry-run` left the missing root absent and
  created no extracted launcher files. Corrupting the attached archive failed with exit 6 and no network
  fallback. The first smoke run exposed an error-wrapper classification bug, fixed with regression coverage.

## Remaining acceptance

- SignPath onboarding, signing and signature verification require the owner. The workflow example is
  inactive and all artifacts remain unsigned. Signed-overlay format validation awaits a signing rehearsal.
- Enterprise proxy authentication and OS-specific certificate policies were not tested against a live
  enterprise network. PAC/WPAD scripts are intentionally unsupported.
- Automatic GUI launch selection, screen readers and multi-monitor testing remain Sprint 6 work.
- Mirrors require a new output directory; incremental in-place updates are not implemented.
- Builders keep temporary files beside their output and do not create installation-root logs. The mirror
  manifest records versions/hashes. Archive checksums detect corruption, not publisher identity.
