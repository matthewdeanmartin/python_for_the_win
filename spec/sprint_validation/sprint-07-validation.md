# Sprint 7 validation — 2026-10-15

Go 1.27.2, CGO disabled. Rebuilt `dist/pftw.exe` and `dist/pftww.exe`.

## Findings repaired

| Finding | Regression evidence |
|---|---|
| Ignored `Atoi` overflow accepted enormous minor/patch/prerelease values | Three overflow cases now return errors; version parser fuzz target |
| Invalid known state fields silently became zero-value settings | Malformed PATH mode/structure, portable flag, source/proxy values and unsafe default key fail read-only loading |
| Updating a JSON-null install panicked | Null update now returns an error |
| Plain text output reported success when the writer failed | Injected writer failures cover version, list, doctor and generated docs; JSON remains checked |
| A directory named python.exe passed interpreter lookup | Directory interpreter is rejected |
| Archive device matching rejected COM12/LPT123 | Exact device suffix checks accept ordinary names and reject reserved names; invalid UTF-8 names also rejected |
| Local downloads ignored cancellation | Cancellation before download makes no cache; progress-triggered cancellation leaves no payload/partial file |
| Truncated yes reply could count as install consent | EOF/partial yes rejected; complete yes/no replies tested |
| Target rollback errors were hidden | A real exclusively opened Windows file forces removal failure; both smoke and cleanup errors survive |

Existing-file extraction protection also has an explicit adversarial test. Builder/target temporary cleanup
and partial-download removal now propagate failures rather than claim silent success. Read-only handle releases,
progress rendering and selected OS notifications have explicit best-effort annotations; durable writes remain checked.

## Local checks

- `make quality`: security scan, vet, Staticcheck, pinned Errcheck, unit tests and both builds passed.
- Final vet/lint/unit/build rerun passed after the final consent/UTF-8 changes.
- Errcheck also passed while analyzing Linux and Darwin build constraints from Windows.
- Two 10-second fuzz runs passed: 9,439 version-parser executions and 31,843 archive-name executions.
- Coverage snapshot: 56.1% of statements in the unit suite; not a release gate or an integration/GUI coverage claim.
- Full Windows lifecycle integration passed (`PFTW_INTEGRATION=1 go test -tags integration -timeout 10m ./...`).
- Final GoReleaser snapshot packaging passed for Windows/Linux/macOS amd64/arm64; local unpublished Scoop
  metadata refreshed. Nothing published or signed. Final manager size: 13,928,448 bytes.

Evidence lives in ignored `dist/sprint7-*.txt` logs, with `dist/coverage.out` for coverage. Reproduced failures
are in `sprint7-regressions-before.txt`; the early Errcheck v1.9.0 attempt could not analyze this Go toolchain,
so v1.30.0 is pinned instead. No upstream report was filed.

## Limits

Hosted CI was not triggered. Runtime race detection was not run in this CGO-disabled workflow. Fuzzing was
bounded and does not establish absence of parser bugs. The existing desktop, accessibility and performance
acceptance matrix remains pending. Signing remains deferred until owner dogfooding is complete.

Errcheck does not audit explicit blank assignments automatically, nor are all existing test-fixture errors
covered by it. This sprint manually reviewed relevant ignored parsing, output, consent and cleanup paths;
it does not claim every possible I/O failure in the application is now covered.
