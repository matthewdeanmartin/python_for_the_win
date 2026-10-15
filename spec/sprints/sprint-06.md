# Sprint 6 — Desktop entry and tool integration

Code delivered for dogfooding:

- Conservative Explorer GUI / terminal TUI entry, explicit `--gui` and `--tui`; CLI and shim dispatch first.
- `which [SPEC]` prints an installed interpreter path, with JSON support and no writes.
- `install --target DIR` produces a plain verified runtime, no manager state/shims/PATH or pip bootstrap.
- Working-directory ancestor `.python-version` lookup after explicit/script selection and before default.
- Dependency vulnerability upgrades, repeatable Makefile scan/update targets, CI security scanning.
- Validation documents moved into `spec/sprint_validation/`.

Third-party signing is deferred by the owner until dogfooding is complete. Hosted CI, desktop-launch matrix,
console-subsystem alternatives, Narrator/NVDA, focus/high-contrast and multi-DPI acceptance remain pending.
See [validation](../sprint_validation/sprint-06-validation.md) for observed checks and limitations.
