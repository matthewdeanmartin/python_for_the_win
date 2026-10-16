# Which Python runs (PATH)

When you type `python`, Windows searches the folders listed in PATH, in order, and runs the first `python.exe`
it finds. PATH is built from the **system** list (shared by all users) followed by your **user** list. pftw
only ever edits your user list, adding its folders at the front.

`pftw doctor` shows what a new terminal will run for `python`, `python3`, `pip` and `py`, and whether that's
pftw's.

## `python` still runs the old Python

- **Open a new terminal.** Existing terminals keep the PATH they started with. Editors (VS Code, PyCharm) and
  Windows Terminal tabs started from a running editor inherit the editor's PATH: restart the editor. In rare
  cases, signing out and back in is needed for programs started from Explorer.
- **You chose not to change PATH**, or answered No to "Make Python available as a command?". Run pftw's
  Python with `pftw run`, by its versioned name if `pftw`'s commands are on PATH, or by full path
  (`pftw which`).
- **The other Python is on the system PATH.** System entries always come before user entries, and pftw never
  edits the system PATH. Either uninstall that Python, ask an administrator to remove it from the system PATH,
  or keep using pftw's Python through `pftw run` / full paths.
- **Git Bash remembers command locations.** Run `hash -r`, or open a new Git Bash window.

## Another Python is on PATH

pftw never silently takes over from another Python, because that changes what existing scripts and projects run.

- **On your user PATH:** the wizard asks you to choose; on the command line, pass `--path first` (pftw's Python
  becomes `python`) or `--path none` (leave PATH alone). `--yes` can't make this choice for you.
- **On the system PATH:** pftw can't come first without editing the system PATH, which it never does. Choose
  "continue without changing PATH" in the wizard, or pass `--path none`.

The other Python is not changed or removed either way.

## App execution aliases

Windows ships small `python.exe` and `python3.exe` stubs ("app execution aliases") that open the Microsoft
Store. pftw only stops for them when one would still run instead of pftw's Python after installing, which
happens when their folder (`%LOCALAPPDATA%\Microsoft\WindowsApps`) is on the system PATH.

To turn them off: **Settings → Apps → Advanced app settings → App execution aliases**, then switch off
"App Installer python.exe" and "python3.exe". pftw never changes this setting for you. Or install with
`--path none`.

## `py` does not see pftw's Pythons

`py` is a different launcher (from the old python.org installer or the Python Install Manager). It finds Pythons
through registry entries that pftw deliberately doesn't create. pftw doesn't provide or replace `py`.

Instead:

| Instead of | Use |
|---|---|
| `py script.py` | `pftw run script.py` |
| `py -3.14 script.py` | `pftw run --python 3.14 script.py`, or `python3.14 script.py` |
| `py -0p` | `pftw list`, `pftw which 3.14` |

Tools that discover Pythons through the registry won't list non-default pftw installs; point them at the path
from `pftw which`.

## `pip` installs into a different Python

`pip` is just another command on PATH and can belong to a different Python than `python`. Use
`python -m pip ...` (or `python3.14 -m pip ...`) so pip always matches the interpreter you mean.

## Versioned commands are missing

`python3.14` and similar commands live in pftw's `shims` folder, which is on PATH only when pftw manages PATH.
`pftw doctor` reports missing ones, and `pftw doctor --fix` restores those pftw created. Windowed
`pythonw3.X` commands also need `pftww.exe` next to `pftw.exe` at install time; the release ZIP includes it.

## pftw's PATH entries disappeared

Another program or a manual edit removed them. `pftw doctor` reports "Managed PATH entry is missing";
`pftw doctor --fix` adds back only entries pftw owns, without touching anything else. Your PATH as it was
before pftw first changed it is saved in `%LOCALAPPDATA%\pftw\logs\path-backup-*.txt`.
