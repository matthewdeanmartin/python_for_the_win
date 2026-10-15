# Signing handoff (not activated)

Owner decision: third-party signing is deferred until the current features have been dogfooded.
No application, credentials or signing request is needed for this sprint.

The project remains unsigned. No SignPath account, application, credential, signing request or release
was created by the implementing agent. Spec D13 and section 15 reserve the application for the owner.

The local [workflow draft](signing-workflow.yml.example) builds both Windows executables, uploads them,
requests signing, and downloads the result for inspection. It is outside `.github/workflows` and cannot run
as a repository workflow. It requires a configured SignPath project/policy and API token. The provider's
[GitHub integration documentation](https://docs.signpath.io/trusted-build-systems/github) describes the
artifact upload and request parameters.

Owner setup:

1. Apply to SignPath Foundation and obtain acceptance for this MIT-licensed repository.
2. Configure the GitHub trusted build connection and an artifact configuration for the uploaded ZIP's
   pftw.exe and pftww.exe. Restrict the signing policy to reviewed builds and authorized approvers.
3. Configure a protected `signing` environment, the token secret, and organization/project/policy variables
   used by the draft. Review and pin action revisions before moving it into the workflow directory.
4. Run a signing rehearsal, inspect Windows signatures and timestamps, and test installation, shims,
   self-update and Scoop with the signed binaries.
5. Integrate signing before release archives, checksums, provenance and Scoop hashes are finalized.
   Never replace published unsigned bytes with signed bytes under unchanged checksums.

Executable offline bundles are currently unsigned as well. Appending an archive changes an executable's
signed content. Do not assume a signed manager confers a signature on a generated bundle. Signed overlay
support and post-signing format validation need a separate acceptance check once credentials exist.
The standalone-manager 15 MB limit excludes bundled runtime data.
