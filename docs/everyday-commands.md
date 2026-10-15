# Everyday commands

## See and switch versions

```bash
pftw list                # installed versions; * marks the default
pftw default 3.14        # make 3.14 answer to "python"
pftw which 3.15          # full path to an installed interpreter
pftw which               # path to the default
```

`which` never installs anything and ignores project files.

## Run a specific Python (`pftw run`)

```bash
pftw run script.py arg1 arg2
pftw run --python 3.15 -c 'print("hello")'
pftw run -m http.server 8000
pftw run --python .venv/Scripts/python.exe script.py
pftw run --python 3.15 --dry-run -c 'print("hello")'   # show the choice without running
```

pftw's own flags go **before** the script, `-m` or `-c`. Everything after that is passed to Python untouched.

The Python is chosen by the first of these that applies:

1. `--python` (a version or a path to an interpreter).
2. The script's shebang, for example `#!/usr/bin/env python3.15`.
3. PEP 723 inline metadata (`requires-python`). Listed dependencies are **not** installed; use a venv,
   uv or pipx for those.
4. The nearest `.python-version` file in the current folder or above. Put one version such as `3.15` in it;
   comments and blank lines are fine.
5. The default install.

If the chosen version isn't installed, `run` stops (exit code 9). Add `--install` to install it
(without touching PATH), or answer **y** when asked in an interactive terminal (the default answer is No).
Supported version constraints: plain versions, alpha/beta/RC, comparisons, comma-separated ranges, `~=`
and `==3.15.*`. Anything else is rejected rather than guessed.

## Upgrade

```bash
pftw upgrade 3.14
pftw upgrade --all
pftw upgrade 3.16 --fresh    # start without the old installed packages
pftw self-update             # update pftw itself (Scoop users: scoop update pftw)
```

Upgrades stay within the same minor version, variant and architecture. Installed packages and Scripts are
copied over unless you use `--fresh`. The old install is kept until the new one passes its checks, and an
interrupted upgrade is recovered the next time you run a command that makes changes. Don't install packages
while an upgrade runs. Existing venvs are not touched or repaired, and compiled packages may need
reinstalling after a pre-release upgrade. `self-update` needs a published release; development builds need
`--force`.

## Remove

```bash
pftw uninstall 3.14
pftw uninstall --all
```

Removes managed runtimes, their PATH entries and commands, and pftw's own files. Files pftw didn't create,
and files still in use, are left in place with a message. Pythons that pftw didn't install are never touched.

## Check your setup

```bash
pftw doctor                    # which python/pip/py a new terminal will run, and any problems
pftw doctor --fix --dry-run    # preview repairs
pftw doctor --fix
```

`--fix` lists repairs before making them. It restores missing commands and PATH entries that pftw owned,
and never touches unrelated files or PATH entries. Missing runtimes or damaged state need a closer look.

## Preview anything

Every command accepts `--dry-run` (alias `--dry-mode`). Commands that would change things describe the
changes instead, and write nothing: no caches, locks, logs or temp files. Read-only commands run normally.
Network lookups still happen. A preview isn't a guarantee: disk space, permissions, locked files or the
download source can change before the real run. `gui --dry-run` points you to `install --dry-run`.
