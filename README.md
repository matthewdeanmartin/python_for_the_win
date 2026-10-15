# Python for the Win

1. Get `pftw.exe` from a local build or a published release when available (none is published by this sprint).
2. Double-click `pftw.exe`.
3. Press Enter through the defaults; explicitly choose what to do if another Python is on PATH.

If Windows SmartScreen blocks this unsigned development build, verify its origin, then choose
**More info → Run anyway**. Signing is planned; this project does not yet have a signing certificate.

For a graphical window, run `./dist/pftw.exe gui`. No arguments still open the terminal wizard.

The wizard chooses the newest usable final Python. Advanced options are behind **A**;
**Advanced user info** is collapsed behind **I** in the terminal wizard. The GUI has Previous/Next
navigation and a Help button opening the documentation. PATH defaults to Yes. Choose No to leave PATH alone.

## Try it

Requires Windows to install runtimes. Build with Go 1.27.2 and Git Bash:

```bash
make build test vet
./dist/pftw.exe
./dist/pftw.exe list --available
./dist/pftw.exe doctor
```

For a contained demo that leaves your command resolution alone:

```bash
./dist/pftw.exe install 3.15 --portable --root C:/pftw-demo
./dist/pftw.exe list --json --root C:/pftw-demo
C:/pftw-demo/shims/python3.15.exe --version
C:/pftw-demo/shims/pftw.exe doctor
```

Choose a writable directory. Portable means containment, not permission to relocate Python and packages.
A manager launched from its `shims` directory automatically finds its own root.

```bash
./dist/pftw.exe default 3.15 --root C:/pftw-demo
./dist/pftw.exe upgrade --all --root C:/pftw-demo
./dist/pftw.exe uninstall --all --root C:/pftw-demo
```

Uninstall removes managed runtimes, their PATH entries and commands, and known manager files. Foreign files
and files still in use are retained with a message. Existing Pythons outside that root are untouched.

## Preview, run and use offline

All CLI commands accept `--dry-run` (alias `--dry-mode`). Mutation commands describe proposed filesystem
and settings changes without creating caches, locks, logs or temporary files. Read-only commands run
normally, without writing their index cache. Remote metadata queries are allowed. Pending recovery can
block a preview: the command reports that uncertainty rather than recovering during a dry run.
Preview does not guarantee a later operation will succeed; available disk space, permissions, process locks
and remote contents can change. `gui --dry-run` directs you to `install --dry-run` instead.

```bash
./dist/pftw.exe install 3.15 --path none --dry-run
./dist/pftw.exe upgrade --all --dry-mode
./dist/pftw.exe uninstall --all --dry-run
./dist/pftw.exe doctor --fix --dry-run
./dist/pftw.exe run --python 3.15 --dry-run -c 'print("hello")'
./dist/pftw.exe run --python 3.15 -c 'print("hello")'
./dist/pftw.exe bundle 3.15 --out python-offline.zip
```

Runner flags go before the script, `-m`, or `-c`; everything afterwards belongs to Python/the script.
Dry-run displays the interpreter, selection reason and argv without executing Python. Missing Python
requires explicit `--install` or an interactive Yes (default No); installation through the runner leaves
PATH alone. Inline metadata selects Python but never installs dependencies. The supported constraints
cover numeric Python versions, alpha/beta/RC versions, comparisons, comma intersections, compatible
releases and equality wildcards; unsupported syntax is rejected rather than guessed.

Extract an offline bundle and run its `pftw.exe install 3.15 --path none`. It finds the adjacent `index.json`
and verifies the bundled runtime ZIP against its original SHA-256. Bundle creation refuses to overwrite
an existing output. The optional `pftww.exe` companion must stay beside `pftw.exe` when installing if you
want versioned `pythonw3.X.exe` commands; `make build` and Windows release ZIPs include it.

`doctor --fix` lists repairs before applying them. It restores missing aliases recorded in pftw's ownership
manifest and missing, previously owned user PATH entries. It preserves existing files and unrelated PATH
entries. Missing runtimes, ambiguous/corrupt state and unowned files still require inspection.

See [accessibility status](docs/accessibility.md) and [Sprint 4 validation](docs/sprint-04-validation.md).

## Executable bundles and mirrors

```bash
./dist/pftw.exe bundle 3.15 --out python-offline.exe --dry-run
./dist/pftw.exe bundle 3.15 --out python-offline.exe
./python-offline.exe install 3.15 --path none
./dist/pftw.exe mirror 3.14 3.15 --out python-mirror
./dist/pftw.exe list --available --source ./python-mirror
```

An executable bundle carries its own index and runtime ZIP. Launch it without arguments for the terminal
wizard, use `gui` for the graphical wizard, or pass ordinary CLI commands. It reads the attached archive
without unpacking a temporary launcher, including during dry-run. Installation copies the standalone
manager and optional windowed companion into the managed root, without retaining the attached download.
Explicit `--source` and `PFTW_SOURCE` can override the embedded source. Executable bundles are unsigned.

A mirror is a new directory containing `index.json`, verified ZIPs and `mirror-manifest.json`. It includes
only requested versions and deduplicates payloads by hash. Use it locally or serve its contents over HTTPS;
pftw does not start or publish a server. Existing output files/directories are refused. Bundle/mirror work
files stay beside the output and are cleaned after completion or failure; installation-root state is untouched.
The mirror manifest records the exported versions and hashes.

If no proxy or bypass configuration is supplied and a direct connection fails, Windows builds can retry
using the user's static Internet Settings proxy. Settings are read only on failure and are never changed.
Configured command-line/environment/config proxies retain precedence. Environment `NO_PROXY` also disables
this last-resort fallback. Static bypass rules and per-protocol proxy settings are respected. PAC/WPAD scripts
are not evaluated; failed fallback errors say so. Other operating systems do not read Windows proxy settings.

See [Sprint 5 validation](docs/sprint-05-validation.md) and the [unsigned-build signing handoff](docs/signing.md).

## Selection and PATH

The default is the newest eligible final release. Explicit `install 3.16` permits its development release.
`latest` never silently falls back to an alpha. The available list chooses the newest usable binary for each
minor/variant/architecture. `list --available --show-unavailable` includes exact rejection reasons. Older
legacy entries point to NuGet packages without SHA-256 metadata; binaries exist, but that package layout
is unsupported and verification is never waived. If an older usable ZIP exists, pftw selects it.

PATH integration is automatic when there is no conflict. A user-PATH clash requires an explicit choice:
`--path first` puts pftw first, and `--path none` leaves PATH alone. A system-PATH Python cannot be outranked
by editing user PATH. `--yes` never supplies conflict consent. Open a new terminal after PATH changes.

`python` points directly at Python. pftw never supplies `py` or `pyw` and does not take over `.py` associations.
Use `pftw run` to select an installed interpreter by flag, script shebang, inline metadata, or default.

## Upgrade and self-update

```bash
./dist/pftw.exe upgrade 3.14
./dist/pftw.exe upgrade --all
./dist/pftw.exe upgrade 3.16 --fresh
./dist/pftw.exe self-update
```

Upgrades stay within the installed minor/variant/architecture. Packages and Scripts are copied unless
`--fresh` is selected; the complete old tree remains until checks and state commit succeed. Do not modify
packages concurrently with an upgrade. Existing venvs are not searched, upgraded or repaired, and ABI
compatibility is not guaranteed. Interrupted upgrades recover before the next mutating command.

Self-update requires a published release with checksums; development builds require `--force`.
Scoop-managed copies use `scoop update pftw`. No release was published by this sprint.
The explicit Gio GUI, runner, windowed shims, offline ZIP bundles and conservative doctor repairs are available.
Automatic desktop GUI selection remains planned. Linux/macOS builds are for inspection.

`--source` accepts a compatible HTTPS index or local bundle directory. `config set source ...` and
`config set proxy ...` persist settings. Whoever controls a source controls the installed payloads.

See the generated [CLI reference](docs/cli.md), [release checklist](docs/releasing.md), and
[Sprint 3 validation record](docs/sprint-03-validation.md).

## Development

`make lint` runs pinned Staticcheck; `make integration` installs official runtimes in temporary roots and
checks pip/venv, upgrades, shims, defaults and uninstall. Tests do not change your real PATH: they use an
in-memory PATH store and scratch registry keys under `HKCU\Software\pftw-test\Environment`.
`make resources` regenerates Windows resources with pinned pure-Go go-winres. CI covers Windows/Linux/macOS.
The source is MIT licensed; CPython keeps its bundled licenses.

## Original idea and philosophy

Python installer for windows that doesn't require an enterprise IT department or attempt to integrate with one.

An installer that isn't arrogant and elitist and require you to understand the internals of the installer and operating
system to install or to be a machine who doesn't need to understand.

## Feature wish list

### Usability

- "Next, next, next, finish" should be a correct option
- All default installation works or the installer refuses to run. For example, if other applications need to be
  uninstalled first, then the installer will refuse to run until the other applications are uninstalled.

### Removing Marketing Concerns of one Company

- No money grabs with commercial stores.
- No requirement for a virtual machine like WSL or a Macintosh. Some people are stuck with Windows for one reason or
  another and do not want to participate in the further enrichment of the elites at Microsoft.
- No enterprise concerns, such as Active Directory integration
- No "Add Remove/Programs", "Settings", "Control Panel"
- No registry usage, no registry corruption
- No usage of "Program Files"
- No cross promoting of Microsoft products and services
- No integration with Edge browser
- Default to platform dirs (AppData, etc) but provide options to not use it.
- No cmd.exe or Powershell dependencies, if possible
- No Trademarks or restricted intellectual property of Microsoft's.

### Installer Shape

- Single .exe binary
- Single .zip
- Option to create your own installer from source
- Some sort of upgrade path
- Possible Scoop or NSIS support
- PATH support baked in with a reasonable default
- Is not itself Python
- No C or C++
- Crossplatform as a check on if it has too many Windows-only concerns, but not expectation that this would be used
  anywhere else.

## AI Use

- Will be written by LLMs. Anti-LLM scolds are elitists that weren't going to participate anyhow & on average aren't
  Windows users and can afford a Macintosh.
- Before final, AI text should be updated and more importantly, the default experience should not require reading
  anything.

## License
The implementation is MIT licensed. CPython retains its own bundled licenses.
