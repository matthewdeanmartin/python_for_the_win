# Python for the Win

Python installer for Windows that doesn't require an enterprise IT department or attempt to integrate with one.

An installer that isn't arrogant and elitist and require you to understand the internals of the installer and operating
system to install or to be a machine who doesn't need to understand.

## Installation: Don't have Python at all yet?

1. Download `pftw.exe`.
2. Double-click it.
3. Press Enter until it says it's done.

That's it. You get the newest Python, with pip, set up for your user only. No Microsoft Store, no `.msi` or `.msix`.

If Windows says it "protected your PC", click **More info → Run anyway**.

### Installation: Already have Python?

Get the newest Python without downloading anything else:

```bash
python -m pip install pftw
python -m pftw install
```

Your old Python stays installed. See [more on this](docs/installing.md#already-have-python).

### Don't have python at all, but you have PowerShell or bash?

PowerShell:

```powershell
irm https://raw.githubusercontent.com/matthewdeanmartin/python_for_the_win/main/install.ps1 | iex
```

Git Bash:

```bash
curl -LsSf https://raw.githubusercontent.com/matthewdeanmartin/python_for_the_win/main/install.sh | bash
```

These fetch pftw from PyPI, check it, and start the same installer.
[Details](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/install-scripts.md).


### Installation: You have Scoop?

```bash
scoop bucket add pftw https://github.com/matthewdeanmartin/python_for_the_win
scoop install pftw/pftw
```

## Usage

Open a **new** terminal window, then:

```bash
python --version
pip install requests
python my_script.py
```

Need another version too?

```bash
pftw install 3.14
python3.14 --version
```

Something not working? Run `pftw doctor`, then see [Troubleshooting](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/troubleshooting/index.md).

## Everything else

The [documentation](https://python-for-the-win.readthedocs.io/) ([source](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/index.md)) covers the rest, including:

- [Choosing versions, PATH, portable installs](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/installing.md)
- [Switching, running, upgrading and removing Pythons](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/everyday-commands.md)
- [Offline installers, mirrors and proxies](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/offline-and-mirrors.md)
- [Scripting, CI and AI agents](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/scripting.md)
- [Every command and flag](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/cli.md)
- [Troubleshooting](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/troubleshooting/index.md) and [what pftw changes on your computer](https://github.com/matthewdeanmartin/python_for_the_win/blob/docs/reference/what-pftw-changes.md)

Why this exists: [motivation](https://github.com/matthewdeanmartin/python_for_the_win/blob/spec/motivation.md).

## License

MIT. CPython keeps its own bundled licenses.
