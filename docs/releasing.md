# First release checklist (human operator)

No tag, push or release was performed during implementation.

1. Review and commit the implementation, including generated Windows `.syso` resources. Merge it after CI passes.
2. On a fresh Windows user account, test double-click → defaults → finish; in a new terminal check
   `python --version`, `python -m pip install --only-binary=:all: requests`, and `python -m venv .venv`.
   Repeat with another Python on user PATH. Record conhost/Windows Terminal screenshots and results.
3. Run `make test vet lint integration build`, `goreleaser check`, and
   `goreleaser release --snapshot --clean` with Go 1.27.2. Inspect `dist/release/checksums.txt` and archives.
4. Check repository Actions permissions allow release creation, provenance attestations, and the Scoop manifest
   commit to `main`. Branch protection may require a dedicated token/workflow adjustment before release.
5. Choose a version, update the changelog and Windows resource version, regenerate resources, commit and push.
6. Create and push the chosen tag, for example `git tag v0.3.0` then `git push origin v0.3.0`.
   The prepared workflow runs GoReleaser, publishes artifacts and updates this repository's Scoop bucket.
7. Verify release artifacts, checksums, provenance and `bucket/pftw.json` before announcing availability.
   Test `self-update` from the preceding release and Scoop installation. Local Scoop PowerShell checks are
   optional human-run checks; no PowerShell is added to CI or Makefiles.

The bootstrap bucket manifest describes the local snapshot, whose release URL intentionally does not exist.
It is not an installable published package. The first human release replaces it with real release hashes.
Linux/macOS archives prove cross-compilation; they cannot install Python.

Configuration references: [GoReleaser archives](https://goreleaser.com/customization/package/archives/)
and [Scoop manifests](https://goreleaser.com/customization/publish/scoop/).

Before enabling signing, follow [the signing handoff](signing.md). The draft workflow is intentionally outside
`.github/workflows`. Finalize signatures before checksums and Scoop hashes. Executable offline bundles
remain unsigned and require separate post-signing format acceptance.
