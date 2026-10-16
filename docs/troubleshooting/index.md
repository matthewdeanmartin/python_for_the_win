# Troubleshooting

The installer keeps its screens short on purpose. This section holds the details: what each message means,
why it happens, and what to do.

## First steps

1. **Open a new terminal.** PATH changes only reach terminals opened afterwards. In editors such as VS Code,
   restart the editor itself, not just its terminal panel.
2. **Run `pftw doctor`.** It shows which `python`, `pip` and `py` a new terminal will run, and lists problems
   with a one-line fix for each. `pftw doctor --fix --dry-run` previews safe repairs.
3. **Read the log.** Every change is logged in `%LOCALAPPDATA%\pftw\logs\pftw-YYYYMMDD.log`. The installer's
   failure screen shows the exact path.

## Find your problem

| What you see | Go to |
|---|---|
| "Windows protected your PC" | [Installing: SmartScreen](installing.md#windows-protected-your-pc) |
| "Run pftw as yourself, not as administrator" | [Installing: administrator](installing.md#run-pftw-as-yourself-not-as-administrator) |
| "Not enough free disk space" | [Installing: disk space](installing.md#not-enough-free-disk-space) |
| "Installation target is not owned by pftw" | [Installing: target folder](installing.md#installation-target-is-not-owned-by-pftw) |
| "another pftw is running" | [Installing: another pftw](installing.md#another-pftw-is-running) |
| The install stopped part-way, or the computer restarted | [Installing: interrupted](installing.md#an-install-or-upgrade-was-interrupted) |
| `python` runs a different Python, or isn't found | [Which Python runs](path.md) |
| "Another Python is on user PATH" / "system PATH" | [Which Python runs: another Python](path.md#another-python-is-on-path) |
| "A Python app execution alias would still take precedence" | [Which Python runs: Store aliases](path.md#app-execution-aliases) |
| `py` doesn't see pftw's Pythons | [Which Python runs: py](path.md#py-does-not-see-pftws-pythons) |
| "Python files are in use or access is denied" | [Upgrades: files in use](upgrades.md#python-files-are-in-use) |
| A venv or package broke after an upgrade | [Upgrades: venvs and packages](upgrades.md#a-venv-or-package-broke-after-an-upgrade) |
| "download failed", "cannot read Python index", proxy errors | [Networks and downloads](network.md) |
| "runtime sha256 mismatch", "cannot safely extract runtime" | [Networks and downloads: integrity](network.md#integrity-failures) |
| Files left behind after uninstalling | [Removing pftw](removing.md) |

## Exit codes

Every pftw failure has an exit code, which scripts can rely on: 2 usage, 3 blocked by a check, 4 needs your
decision, 5 network, 6 integrity, 7 files in use, 8 not supported here, 9 Python not installed.
See [Scripting and automation](../scripting.md#exit-codes).

## Still stuck?

Open an issue at <https://github.com/matthewdeanmartin/python_for_the_win/issues> with the output of
`pftw doctor` and the relevant part of the log. Remove anything private first; logs include folder paths
and your PATH.
