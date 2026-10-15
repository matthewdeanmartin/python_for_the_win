# Dependency maintenance

Use Go 1.27.2 and Git Bash for the Makefile workflow.

- `make security`: run pinned govulncheck without changing dependencies.
- `make deps-update`: upgrade dependencies within their current major versions, tidy the module graph,
  then run security scanning, vet, tests, Staticcheck, Errcheck, and both Windows builds. Review `go.mod` and `go.sum` before committing.
- Override the Go executable when needed: `make GO=/c/Users/matth/sdk/go1.27.2/bin/go.exe security`.

A failed check stops the job and leaves dependency edits available for review; it does not silently roll back
or commit them. CI also scans vulnerabilities. Scanning is evidence for known issues, not a security guarantee.
Sprint acceptance and experiment records live in `spec/sprint_validation/`, not in this user/contributor folder.

## Quality checks

- `make quality`: security scan, vet, Staticcheck, Errcheck, unit tests and builds.
- `make fuzz`: two bounded 10-second fuzz runs (Python version parsing and archive path validation).
- `make coverage`: build, run the unit suite with `dist/coverage.out`, then print statement coverage.
- `make integration`: opt-in Windows real-runtime lifecycle tests. These take longer and need runtime sources.

Errcheck analyzes production code; test fixtures are excluded. Explicitly ignored errors still require review:
checked write/close errors, command results and consent must propagate failures. Releasing read-only handles,
OS notifications and progress rendering can be best effort, with a comment explaining the choice. Do not use
blank assignments to silence newly discovered mutation failures. A completed mutation can be followed by an
output or cleanup error; inspect the reported path before retrying.

The coverage report is a diagnostic, not a percentage gate. GUI/OS behavior, race detection and manual
accessibility acceptance require separate evidence. Fuzz duration is deliberately bounded, not exhaustive.
