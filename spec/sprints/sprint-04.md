# Sprint 4 — Runner and offline use (implemented; validation record below)

Prerequisite: Sprint 3 acceptance checks. Read spec sections 7.2, 9, 10, 11 and 13.

## Namespace

`py` and `pyw` belong to other applications. Never create aliases, command parsing, registry registrations,
or file associations for them. `python` remains the actual interpreter; pftw's runner is `pftw run`.

## Work

- Implement `pftw run` exactly as section 7.2 specifies: explicit version, shebang and PEP 723 selection,
  installed runtime resolution, arguments/stdin/exit status/Ctrl+C, with no surprise installs or PATH writes.
  Test precedence, malformed metadata, unavailable versions, Unicode/quoting and redirected streams.
- Build offline ZIP bundles with `pftw.exe`, a minimal `index.json` and verified runtime ZIPs. Preserve hashes,
  support adjacent-source discovery and test installation with network unavailable. Self-extracting EXEs wait.
- Add `doctor --fix` for precisely identified pftw-owned state/shims/PATH issues. Preview proposed changes,
  preserve foreign files and unrelated PATH edits, and reuse locks/recovery. Unknown information stays unknown.
- Use Restart Manager to name processes preventing an upgrade/uninstall. Never terminate them automatically.
  Keep exit 7 and safe recovery when files remain in use.
- Add a windowed shim build and versioned `pythonw3.X` aliases. Test console-free GUI programs plus argument,
  environment and working-directory propagation. Do not introduce a `pyw` alias.

## Acceptance and handoff

Use Go 1.27.2. Run vet, unit tests, lint, Windows integration tests, cross-builds and release snapshot validation.
Build `dist/pftw.exe` and provide exact commands the owner can try. Record actual manual checks and unmet
acceptance criteria. No tag, release or upstream communication without the owner's instruction.
The explicit Gio GUI was brought forward; this sprint adds Previous navigation and browser Help.
Automatic desktop GUI selection and the broader accessibility acceptance matrix remain Sprint 6.

Owner additions: global `--dry-run` / `--dry-mode` (read-only commands execute normally), grouped GUI
controls, documentation Help. See `spec/sprint_validation/sprint-04-validation.md` for measured results and limitations.
