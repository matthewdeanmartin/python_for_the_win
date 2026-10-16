# Python for the Win

Python installer for windows that doesn't require an enterprise IT department or attempt to integrate with one.

An installer that isn't arrogant and elitist and require you to understand the internals of the installer and operating
system to install or to be a machine who doesn't need to understand.

## Installation

1. Download `pftw.exe`. *(No release is published yet. For now, [build it yourself](docs/development.md).)*
2. Double-click it.
3. Press Enter until it says it's done.

That's it. You get the newest Python, with pip, set up for your user only. No administrator, no Store,
no registry clutter.

If Windows says it "protected your PC", click **More info → Run anyway**.
If you already have another Python, pftw asks which one should win instead of guessing.

### Already have Python?

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
[Details](docs/install-scripts.md).

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

Something not working? Run `pftw doctor`, then see [Troubleshooting](docs/troubleshooting/index.md).

## Everything else

The [documentation](https://python-for-the-win.readthedocs.io/) ([source](docs/index.md)) covers the rest, including:

- [Choosing versions, PATH, portable installs](docs/installing.md)
- [Switching, running, upgrading and removing Pythons](docs/everyday-commands.md)
- [Offline installers, mirrors and proxies](docs/offline-and-mirrors.md)
- [Scripting, CI and AI agents](docs/scripting.md)
- [Every command and flag](docs/cli.md)
- [Troubleshooting](docs/troubleshooting/index.md) and [what pftw changes on your computer](docs/reference/what-pftw-changes.md)

Why this exists: [motivation](spec/motivation.md).

## License

MIT. CPython keeps its own bundled licenses.
