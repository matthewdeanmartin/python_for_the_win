# Installing: choices and options

The defaults (double-click, press Enter until done) install the newest final Python for your machine,
add it to PATH, and set up pip. This page covers everything else.

## SmartScreen warning

pftw is not code-signed yet. If Windows SmartScreen blocks it, first make sure you downloaded it from this
project, then choose **More info → Run anyway**. See the [signing plan](signing.md).

## Already have Python?

```bash
python -m pip install pftw    # or: pipx install pftw / uv tool install pftw
python -m pftw install        # newest final Python; or e.g. python -m pftw install 3.15
```

`python -m pftw` works even when pip's `Scripts` folder isn't on PATH; if it is, plain `pftw` works too.
Your existing Python is not changed or removed. Because it's already on PATH, pftw asks what `python` should mean:

- **Put pftw first:** `python` and `pip` become the new version. The `pftw` and `python3.X` commands also
  move onto PATH. Run your old Python by its full path.
- **Leave PATH alone:** `python` stays your old version. Run the new one with
  `python -m pftw run --python 3.15 script.py`, or find its path with `python -m pftw which 3.15`.

Update pftw later with `python -m pip install --upgrade pftw`. See [the PyPI package notes](pypi.md).

## Wizard, GUI or command line

- `pftw` with no arguments opens the terminal wizard when run from a terminal. Launched from Explorer, it
  opens the graphical wizard when it can tell; in this build a console window may appear alongside it.
- `pftw gui` (or `pftw --gui`) forces the graphical wizard; `pftw --tui` forces the terminal wizard.
- In the terminal wizard, **A** opens advanced options and **I** shows advanced details. The GUI has
  Previous/Next buttons and a Help button that opens these docs.
- If input or output is redirected, `pftw` prints help instead of opening a wizard. Use explicit commands.

## Choosing a version

```bash
pftw install                   # newest final release (same as "latest")
pftw install 3.14              # newest 3.14.x
pftw install 3.16              # newest 3.16 build; may be a development release if no final exists yet
pftw install 3.16.0a1          # exactly this build
pftw install 3.15 --prerelease # allow development releases
pftw install 3.15 --freethreaded
pftw install 3.15 --arch arm64 # amd64, arm64 or x86
pftw list --available          # what you can install
pftw list --available --show-unavailable   # including entries pftw can't use, with reasons
```

`latest` never quietly falls back to an alpha. A few old releases are listed only as NuGet packages
without SHA-256 hashes. pftw never skips verification, so it can't install those builds. If an older
usable ZIP exists, pftw picks that one instead.

Installing a version that's already installed upgrades it if a newer patch exists; otherwise it does nothing.

## PATH

PATH is the list of folders Windows searches when you type a command such as `python`.

- No other Python on PATH: pftw adds itself automatically.
- Another Python on your *user* PATH: you must choose. `--path first` puts pftw first; `--path none`
  leaves PATH alone. `--yes` never makes this choice for you.
- Another Python on the *system* PATH: pftw can't outrank it by editing your user PATH. Use `--path none`.
- `python` is the real interpreter. pftw never supplies `py`/`pyw` and never takes over `.py` files.
  Versioned commands such as `python3.14` are always available once installed.

Open a new terminal after any PATH change.

## Plain runtime folder (`--target`)

```bash
pftw install 3.15 --target ./python --dry-run
pftw install 3.15 --target ./python
```

Extracts the verified official runtime into a new directory (its parent must exist) and checks it.
No pftw state, versioned commands, PATH changes or pip are added. The folder is yours: `upgrade` and
`uninstall` don't track it, and pftw makes no promise it still works after you move it.

## Portable install (`--portable`)

```bash
pftw install 3.15 --portable --root C:/pftw-demo
C:/pftw-demo/shims/python3.15.exe --version
C:/pftw-demo/shims/pftw.exe doctor
pftw default 3.15 --root C:/pftw-demo
pftw uninstall --all --root C:/pftw-demo
```

Everything stays inside `--root` and PATH is never touched. Choose a writable folder. "Portable" means
contained, not relocatable: moving the folder may break Python, package launchers or venvs. A pftw started
from a root's `shims` folder finds that root automatically.

## Windowed commands

Keep `pftww.exe` next to `pftw.exe` when installing if you want `pythonw3.X.exe` commands. `make build` and
the Windows release ZIPs include it.

## Administrator

pftw installs per user and refuses to run elevated unless you pass `--allow-admin`.
