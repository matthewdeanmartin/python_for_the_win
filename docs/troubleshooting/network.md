# Networks, proxies and downloads

pftw downloads two things: python.org's list of Windows builds (`index-windows.json`, cached for an hour) and
the chosen Python ZIP. `self-update` also talks to GitHub. Everything uses HTTPS.

## "download failed" / "cannot read Python index"

- Check that <https://www.python.org/ftp/python/index-windows.json> opens in a browser on the same machine.
- Downloads are retried three times. A temporary outage usually clears; try again later.
- Exit code 5.

## Proxies

pftw uses, in order: `--proxy URL`, then the `HTTPS_PROXY` / `NO_PROXY` environment variables, then
`pftw config set proxy URL`.

If none is set and a direct connection fails, pftw tries once more through the proxy set in Windows' Internet
Settings. It only reads that setting, never changes it, and respects its bypass list.

**Automatic proxy configuration (PAC/WPAD) is not supported.** If your network uses an automatic configuration
script, ask IT for the proxy address and pass it with `--proxy` or `pftw config set proxy`. Setting `NO_PROXY`
disables the Windows-settings fallback.

## Certificate errors

pftw checks HTTPS certificates using Windows' own certificate store, the same one browsers like Edge and Chrome
use. Networks that inspect HTTPS traffic install their own certificate there, so pftw normally works on them too.
If you see certificate errors, the inspecting certificate probably isn't installed for your user; ask IT.

## Integrity failures

**"runtime sha256 mismatch"** means the downloaded file isn't the one python.org listed. pftw deletes it and
installs nothing. Causes: a proxy or filter that alters downloads, a corrupted cache, or a tampered mirror.
Retry with `--refresh`; if it persists on your network, try another network.

**"cannot safely extract runtime"** means the ZIP contains a path that could write outside its folder, a link,
or a reserved Windows name. pftw refuses it. This shouldn't happen with python.org's files; check your source.

**"missing or invalid sha256"**: the index entry has no checksum, so pftw won't install it.
`pftw list --available --show-unavailable` lists such entries with reasons.

Exit code 6 for all three.

## Using another source

`--source`, `PFTW_SOURCE` and `pftw config set source` point pftw at a mirror, a folder or an offline bundle.
**Whoever controls the source controls what gets installed**, because the source also supplies the checksums.
Plain `http://` sources are refused unless you pass `--allow-insecure-source`.
See [Offline, mirrors and proxies](../offline-and-mirrors.md).

## "GitHub refused the request (403)"

`self-update` uses GitHub's public API, which limits how often one address can ask. Wait an hour and retry, or
download the new version from the releases page.
