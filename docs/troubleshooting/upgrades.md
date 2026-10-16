# Upgrades, packages and venvs

## How upgrades work

`pftw upgrade 3.14` replaces 3.14.x with the newest 3.14 patch release in the same folder, so paths that point
at it keep working. Installed packages and their commands (`Scripts`) are copied into the new version unless you
pass `--fresh`. The old version is kept until the new one passes its checks; on any failure it's put back.
Upgrades never move between minor versions: 3.14 stays 3.14. Install 3.15 alongside it instead.

## "Python files are in use"

Windows can't replace files that a running program has open. Close everything using that Python, then retry:

- Python scripts, REPLs and servers started from it
- Jupyter kernels, language servers and debuggers in editors (VS Code's Python extension keeps one running)
- venvs created from it (they run the same `python3XX.dll`)
- background tools such as pre-commit hooks or file watchers

Where Windows can tell, the message names the programs holding the files. Nothing is changed when this
happens; the old version stays in place. Exit code 7.

## A venv or package broke after an upgrade

A venv points at the Python it was created from. After a patch upgrade (3.14.7 → 3.14.8) existing venvs
normally keep working, but pftw doesn't promise it, and it never searches for, upgrades or repairs your venvs.
If one breaks, recreate it:

```bash
python -m venv --clear .venv
.venv/Scripts/python -m pip install -r requirements.txt
```

Packages with compiled parts can break after upgrading a **pre-release** (alpha, beta, rc): those versions may
change the binary interface between builds. Reinstall the affected packages, or upgrade with `--fresh` to start
without the old packages.

## Don't install packages during an upgrade

pftw copies packages while upgrading. A `pip install` running at the same time can be half-copied. Wait for the
upgrade to finish.

## "No final upgrade is available"

You have a final release (say 3.14.8), and the newest build of that minor version is a pre-release. pftw never
moves a final install to a pre-release. Nothing to do.

## Moving a portable folder

`--portable` keeps everything in one folder, but Python, pip's launchers (`pip.exe` and other `Scripts\*.exe`)
and venvs record absolute paths. Moving or renaming the folder can break them. pftw doesn't rewrite them. Move it
back, or reinstall into the new location.
