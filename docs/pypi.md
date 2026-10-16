# PyPI package

`pip install pftw` gives people who already have Python a way to get pftw without downloading an exe.

## What the package is

- Windows-only wheels (`win_amd64`, `win_arm64`) that carry the same Go `pftw.exe` and `pftww.exe` as a
  release. pip copies them into the environment's `Scripts` folder. This is the same approach ruff and uv
  use. There is no sdist, so pip never tries to compile anything, and other operating systems get
  "no matching distribution".
- A tiny `pftw` Python package so `python -m pftw ...` works even when `Scripts` isn't on PATH, which is
  common after `pip install --user`. It only finds and starts `pftw.exe`; pftw itself is not Python.
- A pip-installed pftw refuses `self-update` and says to use `pip install --upgrade pftw` instead.
  Managed copies (`shims\pftw.exe`) are refreshed from the running copy on the next install, upgrade or default change.

## Build and check locally (Git Bash)

```bash
make wheels VERSION=0.1.0        # dist/wheels/pftw-0.1.0-py3-none-win_{amd64,arm64}.whl
make wheels-check VERSION=0.1.0  # pip install into a temp venv (wheels only), run, uninstall
uvx twine check --strict dist/wheels/*.whl
```

`VERSION` is the release tag without `v`. The tag is the only version source: goreleaser stamps it into
the release exe, and `make wheels` stamps the same text into the wheel's exe. Wheel metadata needs PEP 440,
so semver prereleases are translated (`0.2.0-rc.1` → `0.2.0rc1`, `-beta.2` → `b2`, `-alpha.1` → `a1`,
`-dev.3` → `.dev3`); anything else is rejected. Wheels are reproducible:
the same inputs give byte-identical files. Add `WHEEL_ARCHES="amd64 arm64 386"` for a 32-bit wheel.

## Publishing

Publishing happens in the `pypi` job of `.github/workflows/release.yml` when a draft release is published on
GitHub; see [releasing.md](releasing.md). The wheels are built at draft time from the exact exes attached to the
release (`make release-assets`) and attached to it as files. After publishing, the release's own wheels are
test-installed with pip on Windows and uploaded with PyPI trusted publishing (no API token), using the trusted
publisher for workflow `release.yml`, environment `pypi`. To publish an older release later:
`gh workflow run release.yml -f tag=v0.1.2`.

The `pftw` name was free on PyPI on 2026-10-15. Wheels are unsigned like the exe; PyPI shows trusted-publishing
provenance, and the release also carries GitHub build-provenance attestations for the wheels.
