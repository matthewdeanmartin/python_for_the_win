# Removing pftw

```bash
pftw uninstall 3.14      # one version
pftw uninstall --all     # every version, pftw's PATH entries, commands, settings and logs
```

`uninstall` removes only what pftw created: the versions it installed, the PATH entries it added (matched
exactly, everything else in PATH is left as is), its commands in `shims`, and its own files. Pythons that pftw
didn't install are never touched. Preview first with `--dry-run`.

## "The root contains foreign files or files still in use; they were left in place"

- **Foreign files:** anything pftw didn't create stays, for example files you saved into its folder. Delete them
  yourself if you don't need them.
- **Files in use:** close programs using those Pythons and run the uninstall again, or delete what's left after
  closing them.
- **pftw itself:** a running program can't delete its own file on Windows. If you ran the copy inside pftw's
  `shims` folder, delete `%LOCALAPPDATA%\pftw\shims\pftw.exe` (and the empty folder) after it exits.

## Uninstalling the default version

If other versions remain, pftw asks which becomes the new `python`. Without a terminal it picks the newest
final release for your machine's architecture.

## Removing it by hand

Everything lives in one folder (`%LOCALAPPDATA%\pftw` unless you used `--root`), plus pftw's entries in your
user PATH. Deleting the folder and removing those PATH entries removes pftw completely. See
[What pftw changes on your computer](../reference/what-pftw-changes.md).

## Copies installed in other ways

- **pip:** `python -m pip uninstall pftw` removes the package's `pftw.exe`; the Pythons it installed stay until
  you run `pftw uninstall --all` first.
- **Scoop:** `scoop uninstall pftw`, again after `pftw uninstall --all` if you want the Pythons gone.
- **`--target` folders:** plain folders, not tracked by pftw. Delete them yourself.
