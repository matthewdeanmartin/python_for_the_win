# Known limitations

- **No signed release yet.** Windows SmartScreen warns about the download; see [signing](../signing.md).
- **Windows only for installing Python.** Linux and macOS builds of pftw exist for development checks only.
- **The graphical wizard** opens automatically only for clear double-click launches from Explorer, and a
  console window may appear alongside it. `pftw gui` always opens it.
- **Screen readers** are not verified; see [accessibility](../accessibility.md). The command line and
  `--json` output are the most predictable alternatives.
- **No `py` launcher and no registry entries for Python**, so tools that only look in the registry won't list
  non-default pftw installs. See [`py` does not see pftw's Pythons](../troubleshooting/path.md#py-does-not-see-pftws-pythons).
- **Venvs aren't managed.** Upgrades keep install paths stable but don't repair or upgrade existing venvs.
- **Proxy auto-configuration (PAC/WPAD) isn't supported.** Give the proxy address explicitly.
- **Very old Python releases** that python.org lists only as NuGet packages without checksums can't be installed.

Planned work is listed in the [TODO](../TODO.md).
