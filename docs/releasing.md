# Releasing

Releases are cut on GitHub by publishing a draft release. Nobody tags, builds or uploads from a workstation.
The version follows the changelog: `Added`/`Security` entries bump the minor version, otherwise the patch
(`Removed` would bump major). The first version came from `initial_version` in `changelogmanager.toml`.

**Releases are immutable** (a GitHub repository setting): once published, a release's files and tag can't be
changed or replaced. So the draft gets every file *before* you publish it, and publishing only verifies and
distributes what's already there.

## How it flows

1. **Day to day:** every user-visible change adds an entry to `[Unreleased]` in `CHANGELOG.md`
   (`uvx --from keepachangelog-manager-fork==6.10.1 kaclm add -t added -m "..."`). CI runs `make changelog-check`.
2. **On every push to `main`, `create_draft_release.yml`:**

   | Job | What it does |
   |---|---|
   | `draft` | `kaclm github-release` creates or updates a **draft** release for the next version (for example `v0.1.1`), with notes from `[Unreleased]`. Drafts are private and create no tag |
   | `assets` | Runs the checks (`go mod verify`, `make security`, vet, tests), builds with goreleaser for the upcoming tag (stamping the version into the exe and its Windows file properties) without publishing, builds the PyPI wheels and Scoop manifest from those exes (`make release-assets`), checks everything against `checksums.txt`, records build-provenance attestations, attaches all files to the draft, and pins the draft to this exact commit |

   Each push replaces the draft's files with a fresh build, so the draft always matches the newest `main`.
3. **To release:** first **wait until the latest *Create Draft Release* run is green**. The draft appears within
   seconds, but its files take a few minutes, and GitHub will let you publish an empty draft. Then open
   **Releases**, open the draft, check the notes and that all 14 files are there (6 zips, 2 exes,
   `checksums.txt`, `install.sh`, `install.ps1`, 2 `.whl` files, `pftw.scoop.json`), and click **Publish
   release**. GitHub creates the tag at the commit the files were built from.
4. **Publishing starts `release.yml`.** It builds nothing:

   | Job | What it does |
   |---|---|
   | `bump` | `kaclm release-bump` moves `[Unreleased]` into `## [0.1.1] - date` on a `release/bump-…` branch and opens a release PR |
   | `verify` | On Windows: downloads the release's files, checks them against `checksums.txt`, `pip install`s the wheels, and checks that `pftw version` reports this version **and the tag's commit** |
   | `pypi` | Uploads the release's own wheels with PyPI trusted publishing (setup below) |
   | `scoop` | Adds the release's `pftw.scoop.json` to the release PR as `bucket/pftw.json` |

5. **Merge the release PR.** That puts the dated changelog section and the Scoop manifest on `main`.
   `[Unreleased]` is then empty, so no new draft appears until new entries arrive.

The git tag is the only version source: the binaries, the wheels (pre-releases translated to PyPI's format) and
`go install …@v0.1.1` all report it.

## Tags and releases are permanent

- **Immutable releases:** a published release can't gain, lose or replace files, and its tag can't be moved.
  If something is wrong, fix it and release the next version.
- **Go:** the first `go install …@v0.1.0` makes the public Go proxy cache that version forever and records its
  checksum in `sum.golang.org`. Never re-create a tag with different contents. To warn people off a bad
  version, release a new one and add `retract v0.1.0 // reason` to `go.mod`.
- **PyPI** versions can't be re-uploaded either.

## If a release goes wrong

Fix forward. Examples: v0.1.0 was published before the flow attached files at draft time, and v0.1.1 was
published while its draft was still empty. Both are immutable source-only releases, and `release.yml` refused
to continue for them ("no assets to download"). To recover:

1. Merge that release's PR anyway, so the changelog records what was released.
2. Pull `main`, add an entry describing the fix for users, and push. For example:
   `kaclm add -t fixed -m "Release downloads (exe, zips, wheels, install scripts) are attached; 0.1.0 and 0.1.1 shipped source only."`
3. A new draft (here `v0.1.2`) appears. Wait for its files (step 3 above), check them, then publish it.

A failed `release.yml` run can be re-run from the Actions tab: `verify`, `pypi` and `scoop` only read the
release, so re-running them is safe.

## One-time setup (repository owner)

1. **Let Actions open pull requests:** Settings → Actions → General → Workflow permissions → tick
   *Allow GitHub Actions to create and approve pull requests*. Without it, the `bump` job can't open the release PR.
2. **Branch protection** (if enabled on `main`): the bump and Scoop jobs push only to `release/bump-…` branches,
   and their changes reach `main` through the PR, so normal protection rules work.
3. **PyPI trusted publishing** (see [pypi.md](pypi.md)): on PyPI, add a trusted publisher for project `pftw`:
   owner `matthewdeanmartin`, repository `python_for_the_win`, workflow `release.yml`, environment `pypi`
   (or leave the environment blank). GitHub creates the `pypi` environment on first use; add required
   reviewers to it if you want to approve each upload.

   To publish an already-released version (for example one released before PyPI was set up), run
   `gh workflow run release.yml -f tag=v0.1.2` or use **Actions → Release → Run workflow**. That verifies the
   release's files and uploads its wheels, without opening another PR.

## Pre-releases

Edit the draft's tag to a pre-release such as `v0.2.0-rc.1` and tick *Set as a pre-release* before publishing,
then wait for the `assets` job to rebuild for the new tag (re-run *Create Draft Release* from the Actions tab).
The wheel version becomes `0.2.0rc1`; pip, the install scripts and `go install …@latest` skip it unless asked
for that version.

## Before publishing

1. Do the one-time setup above.
2. On a fresh Windows account, try the draft's `pftw_<version>_windows_amd64.exe`: double-click → defaults →
   finish. In a new terminal, check `python --version`, `python -m pip install --only-binary=:all: requests`
   and `python -m venv .venv`. Repeat with another Python already on PATH.
3. Publish, then check the `release.yml` run, the release PR, the attestations (`gh attestation verify`), and
   (if enabled) the PyPI page. Merge the release PR.
4. Test `scoop bucket add pftw https://github.com/matthewdeanmartin/python_for_the_win`, `scoop install pftw/pftw`,
   and `go install github.com/matthewdeanmartin/python_for_the_win/cmd/pftw@<tag>` (`pftw version` should match).

## Local rehearsal

`GORELEASER_CURRENT_TAG=v0.1.1 go run github.com/goreleaser/goreleaser/v2@v2.18.0 release --clean --skip=publish,validate`
builds exactly what the `assets` job builds, then `make release-assets VERSION=v0.1.1` assembles and checks
`dist/assets`. That build stamps the version into the committed Windows resource files; restore them with
`make resources`. A plain `--snapshot` build (as CI runs on every push) leaves them untouched.
