# Sprint 5 — Offline distribution and network polish

Implemented scope:

- `bundle SPEC --out FILE.exe`: a Windows manager plus a checksummed ZIP overlay. Ordinary wizard/GUI/CLI
  entry points work directly from the embedded source. Dry-run must not extract a launcher or write cache.
  Retain the existing ZIP form. Installed managers omit the overlay; optional windowed aliases still work.
- `mirror SPEC... --out DIRECTORY`: export only requested runtimes with original SHA-256 values and relative
  URLs. Verify every payload, deduplicate identical payloads, refuse existing output, publish index last.
  Do not run or publish a server. Support the global dry-run aliases.
- Last-resort Windows static Internet Settings proxy read after direct transport failure and only with no
  configured proxy/bypass. Respect static exclusions, never evaluate PAC/WPAD, never write proxy settings.
- Prepare SignPath onboarding and an inactive workflow example. Per spec section 15, the human applies and
  supplies credentials; implementation does not submit signing requests or publish artifacts.

Validation: Go 1.27.2, vet, unit tests, pinned Staticcheck, local build, Windows lifecycle integration,
real mirror-to-executable offline install, corruption rejection, zero-write previews, and release snapshots.
See `docs/sprint-05-validation.md` for observed results and remaining acceptance work.
