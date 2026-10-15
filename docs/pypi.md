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
make wheels VERSION=0.3.0        # dist/wheels/pftw-0.3.0-py3-none-win_{amd64,arm64}.whl
make wheels-check VERSION=0.3.0  # pip install into a temp venv (wheels only), run, uninstall
uvx twine check --strict dist/wheels/*.whl
```

`VERSION` is the release tag without `v`. The tag is the only version source: goreleaser stamps it into
the release exe, and `make wheels` stamps the same text into the wheel's exe. Wheel metadata needs PEP 440,
so semver prereleases are translated (`0.4.0-rc.1` → `0.4.0rc1`, `-beta.2` → `b2`, `-alpha.1` → `a1`,
`-dev.3` → `.dev3`); anything else is rejected. Wheels are reproducible:
the same inputs give byte-identical files. Add `WHEEL_ARCHES="amd64 arm64 386"` for a 32-bit wheel.

## Publishing (human operator)

No upload has been done, and none of this is active.

1. Confirm the `pftw` name is still free on PyPI (it was on 2026-10-15), or reserve it with a first upload.
2. On PyPI, add a *trusted publisher* for repository `matthewdeanmartin/python_for_the_win`, workflow
   `pypi.yml`, environment `pypi`. No API token is needed.
3. Create a protected `pypi` environment in the GitHub repository settings.
4. Copy [`pypi-workflow.yml.example`](pypi-workflow.yml.example) to `.github/workflows/pypi.yml`.
   On the next `v*` tag it builds the wheels, test-installs them on Windows, and uploads them.

Alternatively, upload one release by hand: `make wheels VERSION=...` then `uvx twine upload dist/wheels/*`.

Wheels are unsigned like the exe. PyPI shows trusted-publishing provenance for uploads from the workflow.
