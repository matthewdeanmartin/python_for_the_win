# Offline, mirrors, sources and proxies

## Offline bundle (ZIP)

```bash
pftw bundle 3.15 --out python-offline.zip
```

Unzip it on the other machine and run `pftw.exe install 3.15 --path none` from it. It finds the `index.json`
beside it and checks the bundled runtime against its original SHA-256. `bundle` won't overwrite an existing file.

## Offline installer (EXE)

```bash
pftw bundle 3.15 --out python-offline.exe --dry-run
pftw bundle 3.15 --out python-offline.exe
./python-offline.exe install 3.15 --path none
```

A single executable carrying its own index and runtime. Run it with no arguments for the wizard, with `gui`
for the graphical wizard, or with normal commands. It reads the attached data in place (nothing is unpacked
to a temp folder) and installs a plain copy of pftw into the managed root. `--source` and `PFTW_SOURCE` can
override the built-in source. Bundles are unsigned.

## Mirror (host your own copy)

```bash
pftw mirror 3.14 3.15 --out python-mirror
pftw list --available --source ./python-mirror
```

Creates a new folder containing `index.json`, the verified ZIPs and `mirror-manifest.json` (the versions and
hashes exported). Only the requested versions are included and identical payloads are stored once. Use the
folder directly, or serve its contents over HTTPS yourself; pftw doesn't run a server. Existing output is
refused. Temporary work files sit beside the output and are cleaned up afterwards.

## Choosing the source

The first of these that is set wins:

1. `--source URL|FILE|DIR`
2. the `PFTW_SOURCE` environment variable
3. an `index.json` next to `pftw.exe` (offline bundle)
4. `pftw config set source ...`
5. python.org

Only `https://` URLs, files and folders are accepted; `http://` needs `--allow-insecure-source` and still
requires SHA-256 hashes. **Whoever controls the source controls what gets installed.** Use
`pftw config list|get|unset` to inspect or clear saved settings.

## Proxies

`--proxy URL` wins, then `HTTPS_PROXY`/`NO_PROXY`, then `pftw config set proxy URL`.

If none is set and a direct connection fails, Windows builds retry once using your static proxy from
Internet Settings. pftw reads that setting only after a failure and never changes it. Bypass rules and
per-protocol settings are respected; PAC/WPAD auto-config scripts are not run (failures say so).
Setting `NO_PROXY` turns this fallback off. Other operating systems never read Windows proxy settings.
