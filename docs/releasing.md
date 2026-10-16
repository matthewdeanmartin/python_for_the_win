# Releasing

Releases are cut on GitHub by publishing a draft release. Nobody tags, builds or uploads from a workstation.
The first release is **0.1.0** (`initial_version` in `changelogmanager.toml`). After that, the version follows
the changelog: `Added`/`Security` entries bump the minor version, otherwise the patch (`Removed` would bump
major).

## How it flows

1. **Day to day:** every change adds an entry to `[Unreleased]` in `CHANGELOG.md`
   (`uvx --from keepachangelog-manager-fork==6.10.1 kaclm add -t fixed -m "..."`). CI runs `make changelog-check`.
2. **On every push to `main`:** `create_draft_release.yml` runs `kaclm github-release`. It creates or updates a
   **draft** GitHub release named after the next version (for example `v0.1.0`), with notes generated from
   `[Unreleased]`. Drafts are invisible to the public and create no tag.
3. **To release:** on GitHub, open **Releases**, open the draft, check the title, notes and target branch
   (`main`), and click **Publish release**. GitHub creates the tag at that moment.
4. **Publishing starts `release.yml`:**

   | Job | What it does |
   |---|---|
   | `bump` | `kaclm release-bump` moves `[Unreleased]` into `## [0.1.0] - date` on a `release/bump-…` branch and opens a release PR |
   | `build` | Checks out the tag, runs the tests, runs goreleaser (stamping the version into the exe and its Windows file properties), attaches zips, exes, `checksums.txt` and the install scripts to the release, builds PyPI wheels from those same exes, and records build-provenance attestations |
   | `check-wheels` | On Windows: `pip install` the wheels, run `pftw version` and `python -m pftw version`, and check the version |
   | `pypi` | Uploads the wheels with PyPI trusted publishing. **Skipped until you set `PYPI_PUBLISH`** (below) |
   | `scoop` | Adds the generated Scoop manifest (`bucket/pftw.json`) to the release PR |

5. **Merge the release PR.** That puts the dated changelog section and the Scoop manifest on `main`.
   `[Unreleased]` is then empty, so the next push leaves no draft until new entries arrive.

The git tag is the only version source: goreleaser stamps it into the binaries, the wheels use the same
version (pre-releases translated to PyPI's format), and `go install …@v0.1.0` reports it too.

## Tags are permanent

Publishing a release pushes a tag, and Go treats tags as module versions:

- The first time anyone runs `go install …@v0.1.0`, the public Go proxy caches that version forever, and
  `sum.golang.org` records its checksum.
- **Never delete and re-create a tag with different contents.** Users who already fetched it get checksum
  errors, and the proxy keeps serving the original anyway.
- To withdraw a bad version, release a new one, and list the bad one in a `retract` directive in `go.mod`
  (for example `retract v0.1.0 // broken PATH handling`).
- PyPI versions can't be re-uploaded either. Fix forward with the next version.

## One-time setup (repository owner)

1. **Let Actions open pull requests:** Settings → Actions → General → Workflow permissions → tick
   *Allow GitHub Actions to create and approve pull requests*. Without it, the `bump` job can't open the release PR.
2. **Branch protection** (if enabled on `main`): the bump and Scoop jobs push only to `release/bump-…` branches,
   and their changes reach `main` through the PR, so normal protection rules work.
3. **PyPI**, when you're ready (see [pypi.md](pypi.md)):
   1. On PyPI, add a *pending trusted publisher* for project `pftw`: owner `matthewdeanmartin`, repository
      `python_for_the_win`, workflow `release.yml`, environment `pypi`.
   2. In GitHub, create an environment named `pypi` (optionally requiring your approval).
   3. Add a repository **variable** (not a secret) `PYPI_PUBLISH` = `true`.

   Until then, releases still build, test and attach everything, and the `pypi` job shows as skipped.
   The wheels remain downloadable from the run's artifacts for 7 days.

## Pre-releases

Set the draft's tag to a pre-release such as `v0.2.0-rc.1` and tick *Set as a pre-release* before publishing.
goreleaser marks it as a pre-release, the wheel version becomes `0.2.0rc1`, and pip and the install scripts
skip it unless asked for that version. `go install …@latest` also ignores it.

## Before the first release

1. Do the one-time setup above.
2. On a fresh Windows account, run the release candidate (download the `pftw-windows-amd64` artifact from CI):
   double-click → defaults → finish. In a new terminal, check `python --version`,
   `python -m pip install --only-binary=:all: requests` and `python -m venv .venv`. Repeat with another Python
   already on PATH.
3. Push to `main`, confirm the draft release says `v0.1.0` with sensible notes, then publish it.
4. Afterwards, check the release assets, `checksums.txt`, the attestations (`gh attestation verify`), the
   release PR, and (if enabled) the PyPI page. Then merge the release PR.
5. Test `scoop bucket add pftw https://github.com/matthewdeanmartin/python_for_the_win`, `scoop install pftw/pftw`,
   and `go install github.com/matthewdeanmartin/python_for_the_win/cmd/pftw@v0.1.0` (`pftw version` should say 0.1.0).

## Local rehearsal

`go run github.com/goreleaser/goreleaser/v2@v2.18.0 release --snapshot --clean` builds everything into
`dist/release` without publishing, and leaves the committed Windows resources untouched.
`make release-wheels VERSION=0.1.0` then builds wheels from those binaries. CI also runs a snapshot on every push.
