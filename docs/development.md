# Development

Build with Go 1.27.2 from Git Bash:

```bash
make build test vet
./dist/pftw.exe
./dist/pftw.exe list --available
./dist/pftw.exe doctor
```

Installing runtimes requires Windows. Linux/macOS builds compile and run unit tests, for checking only.

- `make lint` runs pinned Staticcheck and Errcheck.
- `make integration` installs official runtimes into temporary roots and checks pip/venv, upgrades,
  versioned commands, defaults and uninstall.
- `make resources` regenerates Windows resources with pinned pure-Go go-winres.
- `make changelog-check` validates `CHANGELOG.md` with keepachangelog-manager. Add entries with
  `uvx --from keepachangelog-manager-fork==6.10.1 kaclm add -t fixed -m "..."`.
- `make workflows-check` runs actionlint and zizmor. To upgrade actions, run
  `GITHUB_TOKEN=$(gh auth token) gha-update`, then check that the new major versions still accept the inputs we pass.
- `make wheels VERSION=...` / `make wheels-check VERSION=...` build and test the [PyPI wheels](pypi.md).
- `make docs-check` builds the documentation site strictly (broken links fail); `make docs-serve` previews it.
  Pages live in `docs/`, the table of contents in `mkdocs.yml`. Read the Docs builds from `.readthedocs.yaml`.
- CI covers Windows, Linux and macOS.

Tests never change your real PATH: they use an in-memory PATH store and scratch registry keys under
`HKCU\Software\pftw-test\Environment`.

See also: [dependency maintenance](maintenance.md), [releasing](releasing.md),
[signing handoff](signing.md), and validation records in [`spec/sprint_validation/`](https://github.com/matthewdeanmartin/python_for_the_win/tree/main/spec/sprint_validation).
