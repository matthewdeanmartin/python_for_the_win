# Security architecture and roadmap

This is a targeted source review and hardening pass, not a penetration-test
certification. Repository settings and production signing infrastructure were not
audited. The priorities below distinguish shipped defenses from proposed work.

## Trust boundaries

The operating system's permissions are the authority boundary. Locations the
current user cannot modify are OS-managed for pftw's purposes, regardless of who
published their contents. Code installed within the user's permissions is
user-installed code; pftw does not add a second publisher identity or enrollment
system. Permission failures are reported as operation failures, not requests to
judge trust or bypass the OS.

Shared-machine access policy belongs to the operating system and its administrator.
pftw does not attempt to defend the user from that administrator, classify other
accounts as trusted or untrusted, or impose its own ACL policy on shared folders.
It still confines its own writes and removals to the intended managed paths.

pftw downloads and executes Python runtimes. Its trusted inputs include the
manager binary, the selected index and its publisher, GitHub release metadata,
and the user's installation directory and configuration. A malicious index can
choose both an executable and its checksum. SHA-256 protects against corruption
or a payload that differs from the index; it does not authenticate the publisher.

The default index is served over HTTPS from python.org. Custom sources may come
from a flag, environment variable, saved setting, adjacent index, or offline
bundle. These are supported source choices, not grounds for generic trust warnings.
Source information remains available for diagnostics. HTTPS is the default and
HTTP requires explicit opt-in; checksums and input validation remain mandatory.
An HTTP index can have both its payload and checksum replaced by a network
attacker; pftw does not claim to authenticate arbitrary sources or police code
already installed under the user's permissions.

Scripts, installed packages, and copied site-packages are executable code. pftw
is not a sandbox. A compromised same-user account can also modify its runtimes
and configuration. Elevated use increases the consequences; ordinary installs
reject it unless explicitly allowed.

## Defenses in place

- HTTPS by default, redirect scheme checks, request timeouts and cancellation;
  mandatory SHA-256 before extracting downloaded runtimes.
- Downloads capped at 4 GiB, including streams without a declared length; failed
  downloads clean up their temporary files. Index pages, including cached pages,
  are bounded at 16 MiB, with a 100-page traversal limit.
- Archive path, reserved-name, duplicate-name, link and reparse checks; extraction
  entry and expansion limits; exclusive creation of extracted files.
- Absolute child executable paths, argument arrays rather than shell command
  strings, and transactional installation/recovery.
- Commit-pinned Actions, limited workflow token permissions, release build cache
  disabled, build provenance attestations, and optional PyPI trusted publishing.
- Go module verification and reachable-vulnerability scanning in CI and releases.
  Checks run with development and releases, or manually; no scheduled jobs or
  automated dependency-update PRs are configured.
- Staticcheck, errcheck, go vet, regression tests, actionlint and offline zizmor
  enforced in CI.

## Prioritized remaining work

| Priority | Feature/control | Acceptance criteria |
| --- | --- | --- |
| P1 | Authenticate manager updates and bootstrap installs | Verify release provenance against the exact repository and approved workflow before replacing executables; fail closed on wrong identity, missing proof, or modified bytes. Keep an explicit recovery procedure. Current self-update checksums come from the same release as the binary. |
| P1 | Windows publisher signing | Follow the deferred [signing plan](signing.md). Sign before checksums and attestations, validate timestamp and publisher, and test SmartScreen and overlay bundles separately. Signing has not been activated. |
| P2 | Credential-safe diagnostics | Centrally redact URL userinfo, sensitive query values and proxy credentials before logs, GUI errors and JSON reports. Test error paths, not only successful requests. Current errors may contain source URLs. |
| P2 | Bounded child output | Replace unbounded `bytes.Buffer` captures in `internal/proc/runner.go` with bounded output/tails and explicit truncation reporting. Test a noisy child and timeout cleanup. |
| P2 | Runtime health and lifecycle | Add on-demand doctor checks for end-of-support runtimes and update availability. Recommend project venvs and explain which packages an upgrade preserves. Offer a clear fresh-install path. |
| P2 | Supply-chain inventory and locks | Publish an SBOM for each manager/runtime bundle with provenance. Lock and hash transitive Python build/docs tools, not just their direct versions. Audit pinned Go tools as well as application modules. |
| P2 | Broader automated analysis | Add reviewed gosec and secret-scanning gates, avoiding blanket suppressions. Extend bounded fuzzing to feed, state, bundle and update parsers; run race tests on a supported runner with CGO enabled. Existing fuzz targets cover versions and archive names. |

## Repository controls to enable or verify

These are hosted settings, not effects of the committed YAML:

- Protect the default branch and release tags with rulesets and required CI checks;
  require review for workflow, installer and security-sensitive changes.
- Enable private vulnerability reporting, secret scanning and push protection
  where available.
- Restrict Actions to approved publishers and require full commit SHA pins.
  Protect release and PyPI environments and review who can publish releases.
- Review online zizmor findings during maintenance. The offline PR gate cannot
  query remote action/advisory metadata. Do not supply publishing secrets to PRs.
- Verify attestations during a release rehearsal, including a deliberately
  modified artifact. Publishing provenance alone does not enforce consumer checks.

GitHub's [secure use guidance](https://docs.github.com/en/actions/reference/security/secure-use)
describes immutable action pins and workflow trust boundaries. Go's
[vulnerability database documentation](https://go.dev/doc/security/vuln/database)
describes the advisory source used by govulncheck. A clean scan is limited to
known advisories and does not prove that dependencies or application logic are safe.
