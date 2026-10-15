# One-line install scripts

For people who'd rather paste one line than download an exe. Both scripts fetch pftw from PyPI and start it.
**They need a published PyPI release**; none exists yet.

PowerShell (Windows PowerShell 5.1 or PowerShell 7):

```powershell
irm https://raw.githubusercontent.com/matthewdeanmartin/python_for_the_win/main/install.ps1 | iex
```

Git Bash:

```bash
curl -LsSf https://raw.githubusercontent.com/matthewdeanmartin/python_for_the_win/main/install.sh | bash
```

## What they do

1. Read pftw's page on the package index (PyPI by default).
2. Pick the newest final release for your machine (64-bit Intel/AMD or ARM). Pre-releases and yanked
   releases are skipped.
3. Download that wheel and check its SHA-256 against the index. On a mismatch they stop without running anything.
4. Extract `pftw.exe` to a temporary folder and run it, then delete the temporary folder.

With no arguments, pftw opens its wizard, exactly as if you had double-clicked it.

**The scripts never change PATH.** pftw does that itself, with your consent: it keeps `%VAR%` entries
working and asks before taking over from another Python. If you let it manage PATH, pftw itself ends up
on PATH too (`%LOCALAPPDATA%\pftw\shims\pftw.exe`). Open a new terminal afterwards.

## Passing arguments to pftw

```bash
curl -LsSf https://raw.githubusercontent.com/matthewdeanmartin/python_for_the_win/main/install.sh | bash -s -- install 3.15 --path none
```
```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/matthewdeanmartin/python_for_the_win/main/install.ps1))) install 3.15 --path none
```

Without a terminal (CI, scheduled tasks) and without arguments, the scripts run `pftw install`. pftw then stops
with exit code 4 if it needs a decision about PATH; pass `--path first` or `--path none`.
pftw's exit code is passed through. See [scripting](scripting.md).

## Settings

| Variable | Effect |
|---|---|
| `PFTW_VERSION` | Install this pftw version (e.g. `0.3.0`, or a pre-release such as `0.4.0rc1`) |
| `PFTW_INDEX_URL` | Use another PEP 503 "simple" index, such as a company mirror (default `https://pypi.org/simple`) |
| `PFTW_KEEP` | Save `pftw.exe` (and `pftww.exe`) into this folder instead of running it |
| `PFTW_ALLOW_INSECURE` | `1` allows a non-HTTPS index. For testing only: the index supplies the checksum |

## Safety

- Piping a script into a shell runs whatever that URL serves at that moment. To pin it, use a release tag
  in the URL instead of `main` (for example `.../python_for_the_win/v0.3.0/install.ps1`) and set `PFTW_VERSION`.
  The scripts are also attached to each GitHub release.
- To read before running: download the script, look at it, then run it
  (`bash install.sh`, or `powershell -ExecutionPolicy Bypass -File install.ps1`).
- The checksum comes from the same index as the file. It proves the download is intact, not who built it.
  PyPI releases can't be replaced after upload.
- Files fetched this way don't get Windows' "downloaded from the internet" mark, so SmartScreen won't warn.
  That's why the scripts check the hash.

## Without PowerShell or Bash

Windows 10 and later include `curl.exe` and a `tar.exe` that can unpack ZIP files. For a fixed version
(take the wheel URL and SHA-256 from pftw's PyPI page), from any terminal:

```text
curl.exe -LO <wheel URL>
certutil -hashfile pftw-0.3.0-py3-none-win_amd64.whl SHA256
tar.exe -xf pftw-0.3.0-py3-none-win_amd64.whl
pftw-0.3.0.data\scripts\pftw.exe
```

Compare the `certutil` output with the SHA-256 on PyPI before running pftw.

## Testing the scripts

`go test ./internal/installscripts/` (Windows only) builds pftw and runs both scripts against a local fake
index with Git Bash, Windows PowerShell and PowerShell 7. It checks version selection, pinning, refusing a
tampered hash, `PFTW_KEEP`, exit codes, and the piped one-liners.
