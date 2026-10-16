# Scoop

[Scoop](https://scoop.sh) is a command-line installer for Windows, popular with developers who avoid MSI
installers and admin prompts. pftw can be installed with it.

## For users

```powershell
scoop bucket add pftw https://github.com/matthewdeanmartin/python_for_the_win
scoop install pftw/pftw
```

Update with `scoop update pftw`. `pftw self-update` notices a Scoop-installed copy (it lives under
`scoop\apps\pftw\`) and tells you to use Scoop instead, so the two never fight over the file.

Scoop manages only the pftw program. Python itself is installed and put on PATH by pftw exactly as with the
downloaded exe; `pftw uninstall --all` before `scoop uninstall pftw` removes both.

!!! warning "Not usable before the first release"
    Until v0.1.0 is published, `bucket/pftw.json` is a placeholder from a local test build. It points at a
    release that doesn't exist, so `scoop install` fails with a download error.

## How it works

A **manifest** is a small JSON file that tells Scoop which version exists, where to download it, its SHA-256,
and which programs to put on PATH (`pftw.exe`, `pftww.exe`). Ours is `bucket/pftw.json`.

A **bucket** is a git repository holding manifests. `scoop bucket add` clones the repository and reads its
`bucket/` folder. This repository is its own bucket (spec decision D15), so there's no separate repository to
maintain. Scoop clones the whole repository, which is why large binaries must never be committed here.

## What happens at release time

Nothing manual. When a draft release is published (see [Releasing](releasing.md)):

1. goreleaser builds the release zips and generates a manifest pointing at them, with their hashes
   (`scoops:` in `.goreleaser.yaml`; goreleaser's own upload is turned off).
2. `internal/scoopmeta` adds two fields goreleaser can't produce:
   - `checkver: "github"`, which tells Scoop's tooling how to find newer versions;
   - `autoupdate`, URL templates such as `.../download/v$version/pftw_$version_windows_amd64.zip` with the
     hashes read from `checksums.txt`, so a future manifest can be generated from just a version number.
3. The `scoop` job in `release.yml` commits that manifest to the release PR.
4. Merging the release PR puts it on `main`; users get it with their next `scoop update`.

## Pieces involved

| File | Role |
|---|---|
| `bucket/pftw.json` | the manifest users install from |
| `.goreleaser.yaml` (`scoops:`) | generates the manifest from the release build |
| `internal/scoopmeta` | adds `checkver` and `autoupdate` |
| `.github/workflows/release.yml` (`scoop` job) | puts the manifest in the release PR |
| `internal/selfupdate` (`Scoop`) | makes `self-update` defer to Scoop |

## Later

Submitting pftw to a community bucket (such as Scoop's `extras`) would let people skip `scoop bucket add`.
That's a request to another project, so it's left to the maintainer.

## Removing Scoop support

Scoop is optional; the exe, PyPI and the install scripts work without it. To drop it, delete `bucket/`, the
`scoops:` section of `.goreleaser.yaml`, `internal/scoopmeta`, the `scoop` job in `release.yml`, and the Scoop
check in `internal/selfupdate` and `internal/cli/selfupdate.go`.
