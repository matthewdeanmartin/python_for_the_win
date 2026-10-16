# TODO

Known follow-ups, roughly in priority order. Nothing here blocks local use.

## Pin the one-line install scripts to releases

**Problem.** The documented one-liners fetch the script from `main`:

```text
https://raw.githubusercontent.com/matthewdeanmartin/python_for_the_win/main/install.ps1
```

`irm | iex` and `curl | bash` run whatever that URL serves *at that moment*. Anyone who can push to `main`
(or a compromised account or token) changes what every new user runs. The script can't verify itself, and
the SHA-256 check inside it only protects the pftw wheel, not the script.

**Goal.** The URL people copy should name an immutable, reviewed release. The script from that release
should install the pftw from that same release by default.

**Plan.**

1. **Stamp the version into released copies.** At release time, rewrite the default `PFTW_VERSION` in the
   `install.sh`/`install.ps1` that are attached to the GitHub release (for example a goreleaser `before` hook
   writing `dist/release/install.ps1` with `PFTW_VERSION` defaulting to `{{ .Version }}`, converted to
   PEP 440 the way `internal/wheel.Normalize` does). The repo copy keeps "newest final release" behavior.
   Today the attached copies are unmodified, so a tagged script still installs the newest pftw.
2. **Document release-asset URLs as the main one-liner**, not `raw.githubusercontent.com/.../main/`:
   - pinned: `https://github.com/matthewdeanmartin/python_for_the_win/releases/download/v0.1.0/install.ps1`
   - current: `https://github.com/matthewdeanmartin/python_for_the_win/releases/latest/download/install.ps1`
     (follows the newest *published* release, never unreleased commits on `main`).

   Then update the README, `docs/install-scripts.md` and the PyPI description (it reuses the README).
3. ~~Turn on GitHub immutable releases.~~ Done: immutable releases are on, and the release flow attaches all
   files to the draft before publishing, so a pinned tag really is fixed.
4. **Publish the script checksums with the release.** `checksums.txt` already includes both scripts.
   Document the cautious path: download the script, compare its SHA-256 with `checksums.txt` (and the
   build-provenance attestation, `gh attestation verify`), then run it.
5. **Keep `main` working** but describe it as "development version" in the docs. Optionally make the
   `main` copy print a one-line note to stderr pointing at the release URL.
6. **Test it.** Extend `internal/installscripts` to check that a stamped script defaults to its own version
   and that `PFTW_VERSION` still overrides it.

Order: 1 and 2 together (they only make sense as a pair), then 4–6.

## Other follow-ups

- **Versioned commands without taking over `python`.** For "I already have Python 3.12 and want 3.15 too":
  a PATH choice that adds only pftw's `shims` folder (`python3.15`, `pftw`) and leaves `python` alone.
  This changes the PATH-conflict rules in the spec (D10, section 7 `--path`).
- **`which` with nothing installed** says "Python latest isn't installed"; say "No Python is installed by pftw yet".
- **Structured dry-run output.** `--dry-run --json` returns `changes` as English sentences; add machine-readable
  actions (`{"action": "download", "url": ..., "sha256": ...}`) alongside them.
- **`py` / PEP 514 compatibility gap.** Tools that discover Pythons through the registry don't see non-default
  pftw installs. Document `pftw which` as the workaround, or decide (spec D2) whether to register them.
- **Workflow polish from zizmor's auditor profile:** a concurrency group for CI, and `name:` on each job.
- **First release** (human operator): one-time setup in `docs/releasing.md`, then publish the v0.1.0 draft.
- **Signing** after dogfooding: `docs/signing.md`.
