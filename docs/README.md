# pftw documentation

The [main README](../README.md) covers the default install. Everything here is optional.

| Guide | Read it when you want to… |
|---|---|
| [Installing: choices and options](installing.md) | pick a specific version, architecture or location, handle another Python on PATH, or make a portable install |
| [Everyday commands](everyday-commands.md) | list, switch, run, upgrade or remove Pythons, or check your setup with `doctor` |
| [Offline, mirrors, sources and proxies](offline-and-mirrors.md) | install without internet, build your own installer, host your own copy, or work behind a proxy |
| [Scripting and automation](scripting.md) | drive pftw from scripts, CI or AI agents: JSON output, exit codes and previews |
| [Command reference](cli.md) | see every command and flag (generated) |
| [Accessibility status](accessibility.md) | know what the GUI and terminal wizards support today |
| [PyPI package](pypi.md) | install pftw with pip, or build and publish the wheels |
| [One-line install scripts](install-scripts.md) | install with one pasted line in PowerShell or Git Bash, pin a version, or use a mirror |
| [Development](development.md) | build, test or release pftw itself |

## Known limitations

- No release is published yet, and builds are not code-signed.
- Python runtimes install on Windows only; Linux/macOS builds are for development checks.
- Choosing the graphical wizard automatically on desktop launch is conservative and not fully verified.
- Screen-reader support is not verified; see [accessibility status](accessibility.md).

Background: [why this project exists](../spec/motivation.md) and the [full specification](../spec/spec.md).
