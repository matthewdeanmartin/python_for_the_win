# Dependency maintenance

Use Go 1.27.2 and Git Bash for the Makefile workflow.

- `make security`: run pinned govulncheck without changing dependencies.
- `make deps-update`: upgrade dependencies within their current major versions, tidy the module graph,
  then run security scanning, vet, tests, Staticcheck, and both Windows builds. Review `go.mod` and `go.sum` before committing.
- Override the Go executable when needed: `make GO=/c/Users/matth/sdk/go1.27.2/bin/go.exe security`.

A failed check stops the job and leaves dependency edits available for review; it does not silently roll back
or commit them. CI also scans vulnerabilities. Scanning is evidence for known issues, not a security guarantee.
Sprint acceptance and experiment records live in `spec/sprint_validation/`, not in this user/contributor folder.
