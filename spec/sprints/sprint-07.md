# Sprint 7 — Quality and adversarial testing

Implemented scope:

- Add pinned Errcheck v1.30.0 to `make lint` and CI, alongside Staticcheck. Inspect implicit error drops;
  distinguish checked mutations/results from intentional read-handle release and best-effort progress.
- Reproduce and repair integer overflow in Python version parsing, malformed saved settings, null install
  updates, failed text output, non-file interpreters and over-broad Windows device-name rejection.
- Reject invalid UTF-8 archive paths before Windows name conversion. Preserve foreign files during extraction.
- Honor cancellation for local downloads, including cancellation during progress callbacks.
- Require a completely read consent reply. Propagate prompt-output errors before requesting a decision.
- Report target rollback/scratch cleanup failures alongside the original failure, plus export/update scratch
  cleanup failures. Exercise a real Windows sharing violation, without killing processes or changing user PATH.
- Add fuzz entry points for version parsing/archive names and repeatable quality/fuzz/coverage commands.

Validation and limitations: [Sprint 7 record](../sprint_validation/sprint-07-validation.md).
Signing remains deferred for owner dogfooding; this sprint does not claim desktop/accessibility acceptance.
