# pftw — python-for-the-win: Specification

Status: draft v0.2 (2026-10-15); Sprints 1–5 implemented; Sprint 6 code delivered for dogfooding; Sprint 7 quality hardening implemented with local validation. Desktop/accessibility acceptance remains pending. Signing deferred until dogfooding is complete. Decisions below came out of a requirements interview with the project owner.
Implementation is planned in `spec/sprints/`.

## 1. Purpose

`pftw` is a single-binary Python installer and version manager for Windows, written in Go. It installs official
CPython builds (default: **newest available precompiled final release**) without the MSI installer, MSIX, the Microsoft Store, the
Python Install Manager, admin rights, `Program Files`, Add/Remove Programs, or any registry writes other than
the user's own `PATH`.

Context: PEP 773 retires the traditional python.org installer by Python 3.16 in favour of the Python Install
Manager (distributed through MSIX/Store, with an MSI option). python.org now publishes plain ZIP runtimes plus a JSON index with sha256
hashes. `pftw` consumes that index directly.

The 3.16 transition motivates the project; it is not a special-case runtime target. Stable and explicitly
requested development releases use the same installation machinery.

### 1.1 Guiding principles (from README, binding)

1. "Next, next, next, finish" is always a correct path. The defaults must produce a working install, or the
   installer **refuses to run** and explains, in plain words, what to fix.
2. No Microsoft Store, MSI, MSIX, WSL, Edge, Active Directory, Control Panel/Settings/Add-Remove Programs
   integration. No cross-promotion. No Microsoft trademarks in UI text or assets beyond the unavoidable
   "Windows".
3. No registry usage, with exactly **one** exception: the per-user `PATH` value (`HKCU\Environment\Path`).
4. No `Program Files`. Default to platform user dirs (`%LOCALAPPDATA%`) and allow overriding them.
5. Never shell out to `cmd.exe` or PowerShell. Every child process is started directly (CreateProcess via the Go
   stdlib).
6. Pure Go. `CGO_ENABLED=0` always. No C/C++ in the tool or its build (so no NSIS).
7. The default experience requires reading nothing.
8. The tool itself is not Python.
9. **Registry and PowerShell are last resorts, not tools.** Reach for files, the Go stdlib, environment variables,
   and plain Win32 calls first. Read the registry only when it's the *only* source of truth (e.g. user vs system
   PATH, which exist nowhere else). Every registry access needs a one-line comment saying why there's no
   alternative. PowerShell and `cmd.exe` are never used (principle 5).
10. **Assume impermanence.** Microsoft and python.org features come and go (MSI → Store → Install Manager in under a
    decade). Every external mechanism pftw relies on, or merely *detects*, sits behind a small interface, and its
    absence or changed behaviour degrades gracefully instead of breaking pftw. Concretely:
    - Detection code (Store aliases, pymanager, legacy `py.exe`) treats absent features as informational and unknown formats as unknown, never as a blocker
      or a claim of verified absence. Details belong in Advanced user info.
    - The feed is parsed tolerantly (unknown fields ignored, missing optional fields defaulted), and `--source` lets
      a user or a bundle substitute any index with the same schema. If python.org changes or drops
      `index-windows.json`, pftw users can keep working from a bundle or a self-hosted index while a new source
      adapter ships.
    - pftw depends on nothing Microsoft ships as an *app or service* (Store, winget, MSIX, Windows Terminal features,
      Dev Mode symlinks, Edge, PowerShell). It uses only long-lived Win32 APIs (CreateProcess, Job objects, hard
      links, file locks, `SendMessageTimeout`), and even those go through interfaces with fakes.
    - No step assumes a Windows notification is honoured (e.g. `WM_SETTINGCHANGE`). User text always says "open a
      new terminal".

## 2. Decision log

| # | Topic | Decision |
|---|-------|----------|
| D1 | Payload source | python.org ZIP feed: `https://www.python.org/ftp/python/index-windows.json` (follows `next` paging). Only `company == "PythonCore"` entries are installable. `PythonEmbed` / `PythonTest` are ignored. |
| D2 | Registry | Only `HKCU\Environment\Path`. No PEP 514 keys, no Uninstall keys, no file associations, no App Paths. |
| D3 | UI | Initially a Bubble Tea TUI plus CLI. Roadmap: Gio GUI in the same distributed Windows exe; no-argument terminal launch selects TUI, desktop launch selects GUI. Explicit commands remain CLI; `--tui`/`--gui` override detection. See 8.1. |
| D4 | Scope | Per-user only. Never requests elevation. Refuses to run elevated unless `--allow-admin` is passed. |
| D5 | Versions | Multi-version side-by-side manager. Default and `latest` select the newest available precompiled final 3.x release for the requested architecture/variant. Explicit minor requests may fall back to prereleases only when that minor has no final release. Exact prerelease pins and explicit prerelease opt-in are supported through advanced options. |
| D6 | Builds | amd64, arm64, x86 (32-bit), and free-threaded (`t`) variants of each. |
| D7 | Delivery | Online by default (small exe). `pftw bundle` produces an offline, self-contained installer (zip, later exe). |
| D8 | Layout / upgrades | One directory per *minor + variant + arch* (e.g. `3.16-64`). Upgrades retain its path and use a recoverable transaction. Existing venvs are not discovered, migrated, or guaranteed compatible. |
| D9 | Dispatch | User PATH gets the *default* install dir plus its `Scripts\` (the real `python.exe`), plus a `shims\` dir with tiny Go trampolines for versioned names (`python3.15.exe`, `python3.16t.exe`, ...). |
| D10 | PATH clashes | Ask whether to add Python to PATH; default Yes. Choosing No skips PATH conflict decisions. Another Python on user PATH requires a choice before putting pftw first. A system Python that would shadow pftw requires explicit consent to continue without PATH changes, default No. CLI `--path none` supplies that intent; `--yes` alone does not. |
| D11 | Blocking preflight | Store app-execution aliases only when they'd actually shadow pftw's PATH entries (an alias conflict), a non-pftw directory at the target, running elevated, low disk, or the feed unreachable with no offline payload. |
| D21 | Feed source | User-configurable (flag, env var, adjacent bundle index, or `pftw config set source`) with python.org as the default. This supports compatible mirrors and replacement indexes, not arbitrary schema changes: no built-in second source adapter is planned. |
| D12 | Extras | pip via `ensurepip` (default on). **No `py`/`pyw` launcher**: pftw stays out of the `py` namespace, which belongs to the legacy launcher and the Python Install Manager. Version picking is covered by the `python3.X` shims, and shebang-driven version picking by `pftw run` (7.2). **No** Start Menu shortcuts. **No** `.py` file association, ever: most `.py` files are importable modules, not programs, so "double-click = run" is the wrong default (it would also need the registry). |
| D13 | Signing | Unsigned for early releases. Document SmartScreen's "More info → Run anyway". Publish sha256 + GitHub build provenance. Apply to SignPath Foundation (signpath.org, free for OSI-licensed OSS) before 1.0. A human does the application. |
| D14 | Lifecycle | `uninstall`, `self-update`, `doctor`, and `--portable` mode. |
| D15 | Distribution | GitHub Releases (exe + zip, via goreleaser), Scoop, PyPI Windows wheels carrying the same executables (`pip install pftw`; section 12), and `go install`. NSIS dropped (it's C). **Monorepo**: this repo is itself the Scoop bucket (manifest at `bucket/pftw.json`; Scoop looks in a `bucket/` subdir). No separate bucket repo unless Scoop forces one. |
| D20 | Registry/PowerShell posture | Last resort only (principles 9, 10). Reads allowed when there's no other source of truth. The only write is HKCU PATH. |
| D16 | Cross-platform | Compiles and unit-tests on Linux/macOS behind OS interfaces, but installs only on Windows. Linux installation is an optional future direction, not a committed release requirement. |
| D17 | License | MIT. |
| D18 | Name | Binary `pftw`. Go module `github.com/matthewdeanmartin/python_for_the_win`. |
| D19 | State | `pftw.json` in the install root, so a portable install carries its state and deleting the root is a near-complete uninstall. |

## 3. Glossary

- **Root**: the pftw install root. Default `%LOCALAPPDATA%\pftw`.
- **Feed / index**: python.org `index-windows.json` (schema 1), or a local `index.json` with the same schema.
- **Entry**: one object in the feed's `versions` array.
- **Install key**: the directory and identity name for an install, derived from the feed `tag` with `-dev` stripped:
  `3.16-64`, `3.16t-64`, `3.16-arm64`, `3.16-32`, `3.16t-arm64`, etc.
- **Default install**: the install whose dir and `Scripts\` are on PATH, so it answers to `python`, `pip`, etc.
- **Spec** (user input): a version selector such as `3.16`, `3.16t`, `3.16-32`, `3.16.0a1`, `3`, `3.15-arm64`.

## 4. The feed

Observed shape (verified 2026-10-15; capture a trimmed snapshot as a test fixture in Sprint 1):

```json
{ "versions": [ {
  "schema": 1,
  "id": "pythoncore-3.16-64",
  "sort-version": "3.16.0a1",
  "company": "PythonCore",
  "tag": "3.16-dev-64",
  "install-for": ["3.16.0a1-64", "3.16-64", "3-64", "3.16-dev-64", "3-dev-64"],
  "alias": [ {"name": "python3.16.exe", "target": "python.exe"},
             {"name": "pythonw3.16.exe", "target": "pythonw.exe", "windowed": 1}, "..." ],
  "display-name": "Python 3.16.0a1",
  "executable": ".\\python.exe",
  "url": "https://www.python.org/ftp/python/3.16.0/python-3.16.0a1-amd64.zip",
  "hash": { "sha256": "<hex>" }
} ], "next": "<optional url of older index page>" }
```

Notes for implementers:
- Free-threaded entries have `t` in id, tag, and install-for (`pythoncore-3.16t-64`, `3.16t-dev-64`), and
  `executable` is still `.\python.exe`.
- Architecture suffixes are `-64` (amd64), `-arm64`, and `-32` (x86). URL filenames use `amd64` / `arm64` / `win32`.
- The same `id` appears several times with different `sort-version`s (e.g. 3.15.0, 3.15.0rc3, ...). Group by
  `id` and pick by version ordering (PEP 440 ordering: a < b < rc < final).
- Fields not listed here must be ignored, not rejected. Unknown `schema` values > 1 cause that entry to be skipped
  with a debug log line.
- The feed client exposes lazy paging. Sprint 1 resolution reads all pages (bounded to 100 and cycle-checked)
  to establish the newest final and avoid prerelease fallback when a later page contains a final. An early-stop
  optimization requires an explicit upstream ordering guarantee; correctness takes precedence.
- Cache the first page in `<root>\cache\index-<source-hash>.json` for 1 hour. `--refresh` bypasses the cache.
- Relative `url`s in a local index resolve against the index's location.

### 4.0 Choosing the feed source

The feed source is user-configurable (principle 10: if python.org changes or drops the feed, users can repoint pftw to a compatible mirror or replacement index). This does not adapt to a changed schema: that requires a compatible index producer or a code update. Any index with the schema above works: python.org's, a mirror, a
corporate/self-hosted copy, a local directory, or a bundle.

Precedence (first one set wins):
1. `--source <url|path>` flag on the command;
2. `PFTW_SOURCE` environment variable;
3. an `index.json` next to the running `pftw.exe` (offline bundle);
4. `config.source` in `pftw.json` (set with `pftw config set source ...`);
5. the built-in default, `https://www.python.org/ftp/python/index-windows.json`.

Rules:
- Allowed forms: `https://` URLs, local file paths, or directories containing `index.json`. Plain `http://` is
  rejected unless `--allow-insecure-source` is passed, and even then the per-package sha256 must be present.
- The index supplies the hashes, so **whoever controls the source controls what gets installed**. Setting a
  non-python.org source prints a one-line warning that says exactly that, every time it's set (not on every run).
- `pftw list`, `doctor` and the wizard summary show the effective source when it's not the default.
- Each install records the source it came from (`installs.<key>.source`). `upgrade` uses the *current* effective
  source, and if it differs from the recorded one, the upgrade output mentions it.
- The cache is keyed by source URL (hashed file name), so switching sources never serves a stale index from the
  other source.

### 4.1 Spec resolution

1. Map omitted spec and `latest` to stable-only `3`, then normalise the user spec. If it has no arch suffix, append the native-arch suffix (`-64`, `-arm64`, `-32`). Native
   arch means the *machine's* arch (IsWow64Process2 / GetNativeSystemInfo), not pftw's GOARCH.
2. An entry matches if the normalised spec is in its `install-for`. (This gives `3`, `3.16`, `3.16.0a1`,
   `3.16t` the same meaning they have in the Python Install Manager.)
3. Among matches, prefer the highest final version. An explicit minor (e.g. `3.16` or `3.33`) may fall back
   to its newest prerelease only if that minor has no final release. Show its development-release status.
   Exact prerelease pins are intentional advanced requests. `--prerelease` explicitly permits development
   builds. Do not silently fall back to an old alpha of a minor whose final exists but is unavailable for the
   requested architecture/variant; explain that no suitable final build is available instead.
4. The default spec (wizard, or `pftw install` with no argument) is `latest`, equivalent to stable-only `3`.
   Neither selector silently falls back to a prerelease. If no eligible final exists, report that clearly.
   Resolve only entries with a downloadable precompiled payload and required integrity metadata. Paging must
   establish the final-release choice; finding a prerelease on an early page is not a reason to stop searching.
5. `pftw list --available` shows, by default, the newest installable final for each minor/variant/arch, plus the newest installable pre-release of any minor that has no
   stable release yet (e.g. 3.16.0a1, 3.17.0a1), each marked `pre`. `--prerelease` shows every pre-release. `--show-unavailable` includes entries with exact reasons
   (missing URL, absent/invalid SHA-256, unsupported NuGet/non-ZIP layout); prefer an older usable final
   to a newer unusable entry. JSON includes `installable` and `unavailable_reasons`.
6. Exact pre-release pins work: `pftw install 3.16.0a1` installs exactly that build, if it's still in the feed.

## 5. On-disk layout

```
%LOCALAPPDATA%\pftw\                 <root>
  pftw.json                         state (section 6)
  pftw.lock                         exclusive lock while a mutating command runs
  shims\                            on PATH (unless path mode = none)
    pftw.exe                        copy of the tool itself (self-update target); also the shim binary
    python3.exe                     hard link to pftw.exe -> default install
    python3.16.exe, python3.16-64.exe, python3.16t.exe, ...   hard links to pftw.exe
    pythonw3.16.exe ...             hard links to pftww.exe (GUI build, Sprint 4)
                                    (no py.exe / pyw.exe: pftw doesn't use the `py` namespace, see D12)
  versions\
    3.16-64\                        extracted CPython zip (python.exe, Lib\, DLLs\, Scripts\ ...)
    3.15-64\
    3.16t-64\
  cache\                            downloaded zips + cached index (safe to delete)
  staging\                          extraction in progress (cleanup only after transaction recovery)
  trash\                            superseded dirs awaiting delete (cleanup only after transaction recovery)
  logs\pftw-YYYYMMDD.log            rotating, last 10 kept
```

Rules:
- `--root <dir>` overrides the root. The root must be on a local NTFS/ReFS volume (warn on network paths).
- Every path pftw creates is inside the root. Nothing goes in `Program Files`, the Start Menu, the Desktop, or
  `%APPDATA%`.
- All renames happen within the root (same volume), so they are atomic.

## 6. State file `pftw.json`

```json
{
  "schema": 1,
  "portable": false,
  "default": "3.16-64",
  "installs": {
    "3.16-64": {
      "id": "pythoncore-3.16-64", "version": "3.16.0a1", "arch": "amd64", "freethreaded": false,
      "dir": "versions\\3.16-64", "url": "...", "sha256": "...",
      "aliases": ["python3.16.exe", "python3.16-64.exe", "pythonw3.16.exe", "pythonw3.16-64.exe"],
      "installed_at": "2026-10-15T12:00:00Z", "pip": true,
      "source": "https://www.python.org/ftp/python/index-windows.json"
    }
  },
  "config": {
    "source": null,
    "proxy": null
  },
  "path": {
    "mode": "first",
    "added": ["C:\\Users\\me\\AppData\\Local\\pftw\\versions\\3.16-64",
              "C:\\Users\\me\\AppData\\Local\\pftw\\versions\\3.16-64\\Scripts",
              "C:\\Users\\me\\AppData\\Local\\pftw\\shims"],
    "backup": "<user PATH value before pftw first changed it>"
  }
}
```

- Writes are atomic: write `pftw.json.tmp`, fsync, rename over the old file.
- Mutating commands take `pftw.lock` (exclusive file lock). A second concurrent pftw fails fast with a clear message.
- Paths inside the state are relative to the root, except `path.added` and `path.backup`, which are literal PATH
  strings.
- Unknown fields are preserved on rewrite (forward compatibility).

## 7. Commands (CLI)

All commands accept `--root`, `--yes` (assume "yes" on non-decision prompts), `--quiet`, `--verbose`, `--json`
(machine-readable output where applicable) and `--allow-admin`.

| Command | Behaviour |
|---------|-----------|
| `pftw` (no args) | Interactive console: launch the wizard (section 8). Non-interactive: print help, exit 2. |
| `pftw install [SPEC] [--path first\|none\|auto] [--arch amd64\|arm64\|x86] [--freethreaded] [--prerelease] [--source X] [--portable] [--no-pip] [--set-default]` | Resolve, preflight, download, verify, extract, ensurepip, smoke test, register, shims, PATH. The first install becomes the default. If the key already exists: upgrade if a newer patch exists, else "already installed", exit 0. |
| `pftw list [--available] [--prerelease] [--all-arch]` | Installed versions (default marked `*`), or what the feed offers. |
| `pftw default SPEC` | Make an installed version the default and rewrite the pftw PATH entries in place (positions preserved). |
| `pftw upgrade [SPEC\|--all]` | In-place patch upgrade (section 10). Never crosses minors. |
| `pftw uninstall SPEC\|--all` | Remove the version dir(s), its shims, and its PATH entries. `--all` also removes the shims dir, the state, and the root itself (if empty of foreign files). Uninstalling the default with others remaining asks which one becomes the new default (non-interactive: newest stable, native arch). |
| `pftw doctor [--fix]` | Diagnostics (section 11). `--fix` repairs only what pftw owns. |
| `pftw self-update` | Updates `shims\pftw.exe` (and the running exe if it lives elsewhere and is writable) from GitHub Releases (section 12). |
| `pftw bundle SPEC --out FILE.zip|FILE.exe` | Builds an offline bundle: `pftw.exe` + `index.json` + runtime zip(s). Running the bundled `pftw.exe` auto-detects `index.json` next to itself and works offline. The `.exe` form reads an attached archive directly; it never extracts a temporary launcher. |
| `pftw mirror SPEC [SPEC...] --out DIRECTORY` | Write a new self-hostable index, verified payload ZIPs and a manifest. Existing output is refused; no server is launched. |
| `pftw config get\|set\|unset KEY [VALUE]` / `pftw config list` | Persistent settings in `pftw.json` `config`. Keys: `source` (section 4.0) and `proxy` (section 13). `set source` validates by fetching and parsing the index before saving. Unknown keys → exit 2. |
| `pftw run [--python SPEC\|PATH] [--install] [--dry-run] (SCRIPT \| -m MODULE \| -c CODE) [ARGS...]` | Run Python with a version chosen from the flag, the script's shebang, PEP 723 `requires-python`, `.python-version`, or the default (section 7.2). |
| `pftw version` | pftw version, commit, Go version, GOARCH, native arch. |

`--path` semantics:
- `first`: prepend the pftw entries to user PATH. Rejected (exit 4) if a system-PATH clash exists.
- `none`: don't touch PATH.
- `auto` (default): `first` if there's no clash. With a clash, interactive runs ask (D10) and non-interactive runs
  exit 4 ("needs decision; pass --path first or --path none").

`--portable`: implies `--path none`, requires `--root`, sets `"portable": true`, writes nothing outside the
root (no PATH, no cache elsewhere, logs inside the root).

Portable means containment of pftw-managed files and settings, not relocation support. Moving the root may
break Python, installed package launchers, or venvs; pftw does not rewrite them to make relocation work.

Global preview: every CLI command accepts `--dry-run` (also `--dry-mode`). Mutations report planned
filesystem and settings changes without writing caches, logs, locks or recovery files. Read-only commands
run normally with cache writes suppressed. Remote queries are allowed. The runner reports its invocation
without starting Python. GUI previews use the equivalent `install --dry-run` command.

### 7.2 `pftw run`

The shebang-aware runner. It replaces what people used `py script.py` for, without claiming the `py` name.

Forms (pftw's own flags must come before the script/`-m`/`-c`; everything after is passed through untouched):
- `pftw run foo.py a b`: run a script.
- `pftw run -m http.server 8000` / `pftw run -c "print(1)"`: module or code. No script, so no shebang/723 step.
- `pftw run`: REPL of the resolved interpreter.

Interpreter selection (first match wins):
1. `--python SPEC` (e.g. `3.15`, `3.16t`, `3.14-32`) resolved against *installed* versions, or `--python
   PATH\to\python.exe` used as-is (e.g. a venv's interpreter).
2. **Shebang**, read from the script's first line only (max 1 KiB, UTF-8 BOM stripped, must start with `#!`).
   Accepted interpreter forms: `/usr/bin/env [-S] NAME`, `/usr/bin/NAME`, `/usr/local/bin/NAME`, bare `NAME`.
   `NAME` must match `python(3(\.\d+)?t?)?`: `python`/`python3` → project version or default install, `python3.15` → 3.15,
   `python3.16t` → 3.16t. Words after `NAME` (e.g. `-X dev -u`) are passed to the interpreter before the script.
   A non-Python shebang (`/bin/bash`, `node`) → exit 2, "not a Python script shebang".
3. **PEP 723** inline metadata (`# /// script` block with `requires-python = ">=3.12"`): pick the newest
   installed final release satisfying it, falling back to a pre-release only if nothing final fits. If the block
   lists `dependencies`, print one line: "pftw run doesn't install dependencies; use a venv or a tool like
   uv/pipx". Then run anyway (pftw is an installer, not a package manager).
4. The nearest `.python-version` in the current working directory or its ancestors. One version selector,
   at most 4 KiB; blank lines and `#` comments allowed. Multiple versions, executable paths and malformed
   files are errors, not silently ignored. An unversioned Python shebang also falls through to this step.
5. The default install.

If the selected version isn't installed: exit 9 with "Python 3.12 isn't installed: `pftw install 3.12`". With
`--install`, or an interactive yes to "Install it now? [y/N]" (default **No**), install it and run.

Execution reuses the shim machinery (11.1): direct CreateProcess, inherited handles, a Job object, the child's
exit code, and the current working directory unchanged. `--dry-run` prints the chosen interpreter, *why* it was
chosen (flag/shebang/723/.python-version/default), and the final argv, then exits 0.

Non-goals: no `.venv` auto-discovery (use `--python .venv\Scripts\python.exe`), no `PY_PYTHON`-style env vars, no
`.ini` config, no file association, no `pyw` equivalent.

### 7.1 Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success / nothing to do |
| 1 | Unexpected error |
| 2 | Usage error |
| 3 | Preflight blocked (message lists each blocker and its fix) |
| 4 | A decision is required (non-interactive PATH clash) |
| 5 | Network / download failure |
| 6 | Integrity failure (hash mismatch, corrupt archive, unsafe zip path) |
| 7 | Files in use (a Python process is running from the target dir) |
| 8 | Not supported on this platform |
| 9 | Requested Python version isn't installed (`pftw run`) |

`pftw run` and the shims return the **child's** exit code once Python has started, so pftw's codes only mean
something when pftw fails before launching. Automation that needs to tell them apart should call
`pftw run --dry-run` first.

With `--json`, a pftw failure writes one object to stderr, `{"error":{"code":N,"kind":"...","message":"..."}}`,
and nothing to stdout. `kind` names the code above (`usage`, `preflight`, `decision_required`, `network`,
`integrity`, `files_in_use`, `unsupported_platform`, `not_installed`, otherwise `unexpected`). Shim and runner
launches put Python in a kill-on-close Job object with silent breakaway, so Python stops with pftw but
processes Python starts are not killed when it exits (matching the CPython launcher).

## 8. Wizard (TUI)

Shown when `pftw` runs with no args and stdin/stdout are a console. Keyboard only. Enter accepts the highlighted
choice, so "Enter, Enter, Enter" is a valid path whenever there's no forced choice.

1. **Welcome + checks.** Run checks quietly. Show only actionable blockers in plain language, with
   *Check again* / *Quit*. Missing optional integrations or unavailable probe information are not blockers;
   unknown findings remain unknown and must not be reported as verified success.
2. **Version and location.** Default to the newest precompiled final release, native arch, with GIL, in the
   default root. Keep these defaults on a short main flow. *Advanced options* exposes specific versions,
   prereleases, architectures, free-threading, and a custom root. Explicitly choosing a minor with only
   prereleases offers its newest development build with a clear status label. `Latest` always means final.
3. **PATH.** Ask "Make Python available as a command?" (explain PATH briefly), default Yes.
   - No → no PATH changes and no PATH-conflict consent required.
   - Yes, no clash → prepend pftw entries to user PATH.
   - Yes, user-PATH clash → explain that this changes which Python runs, and require a choice between
     putting pftw first and installing without PATH changes (no preselected takeover).
   - Yes, system-PATH clash → explain that the existing Python would still run. Ask "Continue installing
     without changing PATH?", default No. No returns to choices or quits; Yes explicitly authorizes it.
4. **Summary → Install.** Compact summary and progress; advanced settings remain visibly identifiable.
5. **Finish.** Show commands appropriate to the actual outcome. With effective PATH integration, say
   "Open a new terminal and type `python`." Without it, show the installed interpreter's full path and
   how to use it; never imply bare `python` selects pftw. Wait for Enter when pftw owns the console.

**Advanced user info** is a collapsed details area available on checks, progress, results, and failures.
Put probe results, integration findings, source details, paths, logs, and technical diagnostics there.
Keep essential consent, a development-release label, and actionable failure explanations on the main screen.
Do not drown the user in warnings about enterprise products or optional integrations. Advanced *options*
change behavior; Advanced user *info* explains it. Safe defaults require neither area to be opened.

Text rules: no jargon on default screens ("PATH" gets a one-line gloss). No Microsoft product names except
where needed to explain a blocker. Retain the keyboard TUI for the first release; Gio is a later roadmap phase (8.1).

### 8.1 Gio GUI and automatic entry mode (roadmap)

Ship one Windows `pftw.exe` containing CLI, TUI and Gio GUI. Both interactive frontends share installation
logic and safe defaults, including Advanced options and collapsed Advanced user info. No WebView2 or cgo.
Keep Gio behind Windows build constraints so Linux/macOS CLI builds and unit tests remain pure Go.

`pftw gui` now explicitly launches the Windows Gio wizard. This brings forward the forced graphical entry;
no-argument launches select GUI only with interactive handles, an exclusively owned console and an Explorer
parent. Existing terminals select TUI. `--gui` and `--tui` override this conservative detection. Both frontends drive the same state machine and service callbacks.
The graphical command rejects `--json`, `--yes` and `--quiet`; closing the window cancels outstanding work
and waits for cleanup/recovery. No PE subsystem switch or console detachment is used.

Entry precedence:
1. Shim dispatch runs first and never opens a wizard.
2. Explicit CLI commands, help, version and machine-readable requests remain CLI, including redirected I/O.
3. With no command, `--gui` or `--tui` overrides automatic choice; reject conflicting flags. Forced TUI without
   an interactive terminal reports a clear error. Do not open a GUI for a non-interactive automation invocation.
4. Otherwise an existing interactive terminal selects TUI; an ordinary Explorer/desktop launch selects GUI.
   Redirected/piped input or output selects non-interactive help (exit 2), never an unsolicited window.

Windows provides console attachment and handle APIs, not an infallible "double-clicked" flag. Prototype
console/parent-console detection behind `platform.Info`, using console membership, handle types and parent
attachment where appropriate. `GetConsoleProcessList` count alone is only a heuristic, not proof of launch
intent; preserve unknown results and provide the explicit overrides.

Before choosing the final PE subsystem, validate both designs: console subsystem with a newly created console
detached for GUI mode (possible flash), and GUI subsystem with `AttachConsole(ATTACH_PARENT_PROCESS)` for
terminal mode. Do not blindly switch to `-H windowsgui`: shell waiting, exit status, standard handles,
redirection and console shim behavior must keep working. Preserve inherited pipes when attaching a console.
Record the chosen design and any unavoidable console flash; no second user-facing launcher is required.

Acceptance: manually verify Explorer double-click, existing conhost and Windows Terminal sessions, Git Bash,
explicit overrides, redirected/piped CLI calls, exit codes, Ctrl+C, and Python shims. Shells may be test hosts;
pftw never launches a shell. Check keyboard navigation, DPI scaling, accessibility, rendering failure and the
existing size/startup budgets. Measure the Gio binary and revisit the size budget explicitly if necessary.
Initialize graphics only in GUI mode. A rendering failure must offer a useful error and TUI invocation guidance.

References: [Gio Windows build guidance](https://gioui.org/doc/install/windows),
[AttachConsole](https://learn.microsoft.com/en-us/windows/console/attachconsole),
[GetConsoleProcessList](https://learn.microsoft.com/en-us/windows/console/getconsoleprocesslist).

## 9. Preflight checks

| Check | Result |
|-------|--------|
| Running elevated (token elevation) and no `--allow-admin` | **Block**: "Run pftw as yourself, not as administrator." |
| Free space on the root volume < 300 MB (or < 3× zip size + 150 MB) | **Block** |
| Target dir exists and isn't pftw-owned (no matching entry in `pftw.json`) | **Block** |
| Target dir is pftw-owned | Not a block: route to upgrade/repair |
| Feed unreachable and no `--source` / adjacent `index.json` | **Block** (exit 5 in CLI) |
| Store app-execution aliases (`%LOCALAPPDATA%\Microsoft\WindowsApps\python.exe`, `python3.exe`, reparse tag `IO_REPARSE_TAG_APPEXECLINK`) | **Block only on an actual alias conflict**: pftw is adding itself to PATH, and after the planned change an alias stub would still resolve before pftw's `python` (e.g. WindowsApps is on system PATH). Otherwise it's an info line: prepending to user PATH already wins, and with `--path none` pftw isn't on PATH at all, so nothing conflicts. Fix text explains how to turn off "App execution aliases" for Python. pftw never modifies them. (Owner-confirmed 2026-10-15.) |
| Other Pythons on PATH (`python.exe`, `python3.exe` in any PATH dir that's not pftw's, excluding alias stubs) | Classified as user-PATH or system-PATH clash and fed into the PATH decision (D10). Not a block. |
| Legacy `py.exe` on PATH (e.g. `C:\Windows\py.exe`) | Info only (pftw has no `py`, so there's no conflict): "`py` is a different launcher and won't see pftw's Pythons. Use `python3.X` or `pftw run`." |
| Python Install Manager (pymanager) detected | Info only (it's a peer). Its aliases are treated like any other clash. |
| Root on a network share | Warning |

PATH clash detection reads the **unexpanded** user and system PATH values, expands `%VAR%`s for probing only, and
de-duplicates case-insensitively with trailing slashes normalised.

## 10. Install and upgrade pipeline

Install (new key):
1. Resolve the entry (4.1). Preflight (9).
2. Download to `cache\<filename>` (resume not required; retry ×3 with backoff; honour `HTTPS_PROXY`).
3. Verify sha256 against the feed **before** extraction. On mismatch: delete the file, exit 6.
4. Extract to `staging\<key>-<rand>\`. Reject absolute paths, `..` segments, drive letters, ADS (`:`), reparse
   entries, and names that are reserved on Windows (CON, NUL, ...). Exit 6 on any of these.
5. Smoke test: run `staging\...\python.exe -I -c "import sys,ssl,sqlite3,venv; print(sys.version)"` with a 30 s
   timeout, and compare the reported version with the entry.
6. pip (unless `--no-pip`): if `-I -m pip --version` fails, run `-I -m ensurepip --default-pip`. Failure prevents default-mode success; `--no-pip` is the explicit advanced opt-out.
7. Publish to `versions\<key>` under a durable transaction record. Generate or repair pip launchers for this
   final path, then verify both `python -I -m pip --version` and the installed `Scripts\pip.exe --version`
   from an empty working directory. Verify venv creation and execution at the final path. With `--no-pip`,
   omit pip checks. Commit state, shims and PATH with recorded recovery steps; report success only after
   required checks pass. Log everything.

Upgrade (existing key, newer build available, same minor/variant/arch):
1. Resolve, download, verify and extract a fresh tree. Smoke-test it before touching the installed tree.
2. Write a durable transaction journal recording the old state, old/new paths, phase and recovery actions.
   Preserve a complete recoverable old tree. Copy package data into staging instead of moving it out of
   the live installation; detect files-in-use failures and abort safely. Do not promise concurrent package
   writes are supported during upgrade.
3. Rename the old tree to a recorded backup and publish the new tree at the stable install path. Each rename
   is atomic; the multi-step operation is not an atomic directory exchange and has a brief unavailable interval.
4. Validate Python, pip launchers and a temporary venv at the final path. On failure restore the old tree and
   state. On success commit state and mark the transaction committed before deleting its backup.
5. At startup, under the mutation lock, recover any unfinished transaction BEFORE housekeeping. Preserve all
   journal-referenced staging/backup data until recovery succeeds. An interruption at any boundary must lead
   to a complete old or complete new installation, not deletion of the only recoverable package data.

Keep the base interpreter path stable across upgrades. Do not crawl for, upgrade, or repair users' existing
venvs. Stable paths avoid gratuitously breaking references; they do not guarantee ABI or package compatibility.
A compatibility failure in an external venv is outside pftw's upgrade guarantee.

**Pre-release upgrades:** explicitly selected development installs track their minor through prereleases to
final. Alpha/beta upgrades may invalidate compiled packages, including those in external venvs. Put the
technical explanation in Advanced user info; offer an advanced `--fresh` option to omit old site-packages
and package Scripts together. Preserve the old tree until the transaction commits. Do not promise that
`--fresh` repairs external venvs. Final installs stay stable-only unless prereleases are explicitly requested.

## 11. PATH, shims, and doctor

### 11.1 Shims

- **Single binary.** `pftw.exe` is also the shim. At startup, if its own base name (lower-cased, no `.exe`) matches
  `^pythonw?[0-9.]*t?(-(64|32|arm64))?$`, it runs in shim mode and skips CLI/TUI init entirely.
  Any other name (e.g. `pftw`, `pftw_1.2.0_windows_amd64`) runs the normal CLI. This keeps `go install` working with
  no generated or committed binaries.
- Each alias in `shims\` is a **hard link** to `shims\pftw.exe` (no admin, same volume). If hard links fail, fall
  back to copying. After `self-update` replaces `shims\pftw.exe`, regenerate all links.
- **Windowed aliases** (`pythonw3.16.exe`, ...) need a GUI-subsystem binary, or a console window flashes.
  They're deferred to Sprint 4: goreleaser will build a `-H windowsgui` variant (`pftww.exe`) and windowed aliases
  link to it. For `go install` builds without it, windowed aliases are skipped and `doctor` says so. The
  default install's real `pythonw.exe` is on PATH anyway.
- Alias names come from the feed entry's `alias` list, minus the generic names (`python3.exe`, `python3-64.exe`,
  `python3t.exe`, ...). pftw maps `python3.exe` to the default install itself.
- At runtime the shim takes its own base name, finds `pftw.json` in the parent of its dir (`shims\..`), and
  resolves alias → install → `python.exe`. If it can't, it prints one clear line to stderr and exits 1. Shim mode
  must add < 15 ms of startup overhead (measure in Sprint 2).
- Execution: CreateProcess with inherited std handles and the original command line tail (preserve quoting
  exactly: use the raw `GetCommandLineW` remainder, not re-joined argv). The child is assigned to a Job object with
  `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`. The shim ignores Ctrl+C/Ctrl+Break (the child in the same console handles
  them), waits, and exits with the child's exit code.
- `python.exe` itself is **never** shimmed: the default install's real dir is on PATH (D9).

### 11.2 PATH store

- Interface `envpath.Store { Get() (value string, kind ValueKind, err); Set(value string, kind ValueKind) error }`
  with the Windows implementation on `HKCU\Environment\Path` via `golang.org/x/sys/windows/registry`, and an
  in-memory fake for tests.
- Preserve the value type (`REG_EXPAND_SZ` vs `REG_SZ`). Never expand variables when writing. No length truncation
  (the `setx` 1024-char bug must not exist here). Keep empty segments and user order untouched except for pftw's own
  entries.
- Before the first modification, save the raw original value to `pftw.json` `path.backup` and to
  `<root>\logs\path-backup-<timestamp>.txt`.
- pftw entries are exactly `path.added`. They're added as a contiguous block at the front (mode `first`). `default`
  replaces them in place, and `uninstall` removes exactly those strings (case-insensitive match).
- After `Set`, broadcast `WM_SETTINGCHANGE` with `"Environment"` via `SendMessageTimeoutW(HWND_BROADCAST, ...,
  SMTO_ABORTIFHUNG, 5000)`.
- Non-Windows build: a `Store` that returns "unsupported" (exit 8) until v2.0.

### 11.3 doctor

Read-only report with ✓/⚠/✗ lines and a one-line fix for each:
- what `python`, `python3`, `pip`, `py` resolve to on the *effective* PATH (system + user), and whether that's pftw;
- Store aliases, legacy `py.exe`, other Pythons;
- state ↔ disk consistency (missing dirs, orphan dirs in `versions\`, dangling shims, stale `staging\`/`trash\`);
- PATH entries in the state that aren't in the registry value, or the reverse;
- whether the running pftw is older than the latest release (only when online; no nagging elsewhere).

`--fix` only repairs pftw-owned things: re-adds missing pftw PATH entries, removes stale ones, regenerates shims,
cleans staging/trash, and drops state entries whose dirs are gone (after confirmation).

## 12. Self-update and releases

- Release flow (docs/releasing.md): releases are immutable once published, so every file is attached to the draft
  first. On every push to `main`, `create_draft_release.yml` runs `kaclm github-release` to keep a draft release
  in sync with `[Unreleased]`, then builds with goreleaser for the upcoming tag without publishing (stamping the
  version into the binaries and Windows resources), builds the PyPI wheels and Scoop manifest from those exes,
  checks all files against `checksums.txt`, records build-provenance attestations, attaches the files to the
  draft and pins the draft to the build commit. A human publishes the draft, which creates the tag at that commit.
  `release.yml` then builds nothing: it opens the changelog release PR (`kaclm release-bump`), verifies the
  release files and their embedded version and commit on Windows, publishes the attached wheels to PyPI, and adds
  the attached Scoop manifest to the release PR. Nobody tags or builds releases on a workstation. Tags are
  permanent: fix forward and `retract`, never re-tag. Builds not stamped by goreleaser report a clean module tag
  from Go build info (`go install ...@v0.1.0`); local checkout builds stay `dev`.
- `self-update` queries the GitHub Releases API for the latest non-prerelease (`--prerelease` opts in), downloads
  the asset for the native arch plus `checksums.txt`, verifies sha256, and replaces the exe using
  rename-then-write (`pftw.exe` → `pftw.exe.old`, new → `pftw.exe`, delete `.old` on next start).
- Scoop: this repo is the bucket. The manifest is at `bucket/pftw.json` with `checkver`/`autoupdate` on GitHub
  releases. Users run `scoop bucket add pftw https://github.com/matthewdeanmartin/python_for_the_win` then
  `scoop install pftw/pftw`. (Scoop clones the whole repo for a bucket. That's acceptable; keep large binaries out of
  git.) A separate bucket repo is created only if Scoop turns out to require it, and a human creates it. Submitting
  to Scoop's community buckets is a human follow-up. When pftw is Scoop-managed (exe under `scoop\apps`),
  `self-update` says to use `scoop update pftw` instead.
- PyPI: `make wheels VERSION=X` builds `py3-none-win_amd64`/`win_arm64` wheels whose `.data/scripts` hold the
  release executables, plus a minimal `pftw` package so `python -m pftw` locates and starts `pftw.exe` when
  `Scripts` is not on PATH. The tool is still not Python. No sdist. A copy beside a `pftw-*.dist-info` in the
  environment's site-packages is pip-managed: `self-update` refuses (exit 3) and names the pip command. Publishing
  uses PyPI trusted publishing in `release.yml`, skipped until a human configures it and sets `PYPI_PUBLISH` (docs/pypi.md).
- One-line bootstrap scripts (`install.sh` for Git Bash, `install.ps1` for Windows PowerShell 5.1/7) are
  optional, separately distributed conveniences; pftw itself still never launches a shell. They read a PEP 503
  index (PyPI by default), choose the newest non-yanked final wheel unless `PFTW_VERSION` is set, verify its
  SHA-256 from the index, run the extracted `pftw.exe` from a temporary folder and delete it. They never edit
  PATH: pftw's own consent and PATH store do that. Release assets include both scripts (docs/install-scripts.md).
- `go install github.com/matthewdeanmartin/python_for_the_win/cmd/pftw@latest` must work. It does, because shims are
  the same binary (11.1). On first `install`, pftw copies itself to `shims\pftw.exe` if it's running from anywhere
  else.

## 13. Non-functional requirements

- **Pure Go**: `CGO_ENABLED=0` in all builds and CI. Allowed deps: `golang.org/x/sys`, `spf13/cobra`,
  `charmbracelet/bubbletea` + `bubbles` + `lipgloss` (or `huh`), and the stdlib; `gioui.org` and its required pure-Go Windows dependencies are allowed in the GUI phase. Any other new dependency needs a one-line
  justification in the PR description and must not need cgo.
- **Go version**: `go 1.27.2` in `go.mod`; pin CI and release builds to Go 1.27.2.
- **No shells**: a lint test fails the build if the non-test source contains `cmd.exe`, `cmd /c`, `powershell`,
  or `pwsh` as string literals.
- **No elevation**: the manifest embedded in the exe sets `requestedExecutionLevel level="asInvoker"`, so
  installer-detection heuristics don't trigger UAC for a binary named like an installer. Use a pure-Go resource
  tool, e.g. `github.com/tc-hib/go-winres`, run via `go generate`.
- **Size**: `pftw.exe` ≤ 15 MB.
- **Networking**: HTTPS only. TLS via Go's Windows system verifier. Proxy precedence: `--proxy` flag, then
  `HTTPS_PROXY`/`NO_PROXY`, then `config.proxy`. Fallback (sprint 5): if those are unset and a direct connection fails, read the user's static
  proxy setting (registry read allowed under principle 9, because the setting exists nowhere else). Ignore PAC/WPAD
  auto-config and say so in the error message.
- **Logging**: every mutating action goes to `<root>\logs`, including PATH before/after values.
- **Idempotency**: re-running any command after an interruption converges. unfinished transactions are recovered before unreferenced `staging\` and `trash\` data is cleaned.
- **Testability**: OS effects sit behind interfaces (`envpath.Store`, `platform.Info` [arch, elevation, disk,
  console], `proc.Runner`, `links.Linker`). Unit tests run on Linux/macOS/Windows. Windows-only integration tests
  use build tag `integration` and env `PFTW_INTEGRATION=1`, and they really install from python.org into a temp
  root with `--path none`, except for one test that uses a fake HKCU via a test-only Store pointed at a scratch
  registry key (`HKCU\Software\pftw-test\Environment`). That key is created and deleted by the test, and it's the
  only other registry key the project ever touches, in tests only.

## 14. Security

- Integrity is the feed's sha256 over HTTPS from python.org. The feed is trusted to the extent TLS + python.org are.
  Sigstore verification of runtime zips is a future option (track python.org's publishing).
- Zip extraction hardening as in section 10, step 4.
- Never execute anything from `cache\` or `staging\` other than the staged `python.exe` for the smoke test.
- Child processes are started with explicit absolute paths, never via PATH search, and never through a shell.
- Downloaded files are never executed with the cwd set to the download dir. The smoke test runs with `cwd` = a fresh
  empty temp dir and `-I`.

## 15. Rules for implementing agents

- Follow `C:\github\AGENTS.md`: Git Bash, no PowerShell/batch helpers, no external publishing. Do **not** push tags,
  publish releases, create the Scoop bucket repo, apply to SignPath, or file upstream issues. Prepare those as
  drafts for the human.
- Every sprint ends with `go vet ./...`, `go test ./...`, and `CGO_ENABLED=0 go build ./...` passing on Windows
  (local) and green CI on windows-latest, ubuntu-latest, macos-latest.
- At every sprint end, also build a runnable Windows exe at `dist/pftw.exe` using Go 1.27.2 and
  `CGO_ENABLED=0`. Run the sprint-specific integration checks. Report actual results, failures and checks not
  run; never imply CI or manual checks passed without evidence. Hand the owner the artifact path, a short
  "What you can try now" list with exact commands or clicks and expected outcomes, and remaining limitations.
- Keep `spec/spec.md` authoritative. If implementation forces a deviation, update the spec in the same PR and say
  why.

## 16. Roadmap

| Sprint | Theme | Outcome |
|--------|-------|---------|
| 1 (implemented; local checks pass, hosted CI pending) | Foundations | Repo skeleton, CI, feed + resolution, download/verify/extract, state, `install --path none`, `list`, `uninstall` (no PATH) |
| 2 (implemented; local checks, hosted CI pending) | System integration | Platform probes, preflight, PATH store + clash logic, shims (single-binary shim mode), `default`, `doctor` (read-only), `--portable`, exit codes |
| 3 (implemented; local validation, manual/hosted acceptance pending) | Wizard + lifecycle | TUI wizard, double-click UX, in-place upgrade, `self-update`, goreleaser + Scoop manifest draft, README rewrite |
| 4 (implemented; local validation, hosted/manual acceptance pending) | Runner + offline | `pftw run` (7.2: shebang + PEP 723 selection), windowed `pythonw3.X` shims, `bundle` (zip), `doctor --fix`, Restart Manager process naming |
| 5 (implemented locally; signing deferred by owner, hosted acceptance pending) | Polish | `bundle` executable overlay, SignPath signing handoff (unsigned until human onboarding), static system-proxy fallback, `pftw mirror` (write a self-hostable index + zips for chosen specs, the server side of D21) |
| 6 (implemented for dogfooding; desktop/accessibility acceptance pending) | Gio GUI | One distributed Windows exe; desktop GUI / terminal TUI selection, explicit overrides, shared safe wizard flow, console/shim compatibility validation (8.1); `which`, plain `install --target`, `.python-version`, dependency security maintenance |
| 7 (implemented; local checks, hosted CI pending) | Quality | Pinned Errcheck alongside Staticcheck; adversarial unit tests and bounded fuzzing; numeric parsing, state validation, output/prompt errors, cancellation and cleanup reporting; coverage report |
| Optional future | Linux CI | python-build-standalone source, Linux layout, PATH via `GITHUB_PATH`/shell rc guidance, Docker docs |

## 17. Open questions (owner to decide; implementers use the stated default until then)

None at the moment.

Resolved 2026-10-15:
- Default/latest means newest precompiled final; explicit minors may select development releases when no final exists (D5).
- Advanced options and Advanced user info keep the main wizard safe and concise.
- PATH defaults Yes; explicit No bypasses clashes; proceeding past a system clash requires consent, default No.
- Portable promises containment only. Existing venvs are not migrated or guaranteed compatible.
- Recoverable upgrades and final-path pip checks are required; Linux installation remains optional.
- Registry reads are allowed as a last resort (D20). The static proxy read is in sprint 5.
- Scoop uses a monorepo `bucket/` dir (D15).
- Store aliases block only on an actual alias conflict (D11, section 9).
- The fallback for feed changes is a user-configurable source, not a built-in second adapter (D21, section 4.0).

### Sprint 1 implementation notes

- `install` defaults to `--path none` until Sprint 2; other PATH modes fail explicitly.
- Runtime ZIP cache files use the expected sha256 as the filename, avoiding collisions across sources.
- Pip setup and offline launcher regeneration happen after publication at the final path, under the install
  journal. Final-path Python, pip module, pip launcher, and temporary venv checks precede state commit.
- Interrupted new installs and uninstalls recover under the root lock before staging/trash cleanup.
  Transactions support process-interruption recovery; full power-loss guarantees need filesystem-specific
  validation and are not claimed by the local restart tests.
- Sprint 1 `uninstall --all` removes all registered runtimes but retains cache/logs/settings; complete manager
  removal belongs with Sprint 2 integration. Foreign target directories are refused.
- Local verification uses Go 1.27.2: unit tests, vet, Staticcheck v0.8.2, Windows build, Linux/macOS cross-builds,
  a real Python 3.14.8 install/pip/venv/uninstall integration test, and explicit 3.16.0a1 CLI installation.
  Hosted CI is prepared but has not been run or published by the implementing agent.

### Sprint 2 implementation notes

- `install` defaults to `--path auto`. User clashes require explicit takeover consent; system clashes require
  an explicit no-PATH install. Unknown PATH data requires `--path none` instead of guessing which Python wins.
- `default`, `doctor`, versioned console shims, and `--portable --root` are implemented. A root's existing
  PATH policy persists when adding versions; `--path none` does not undo previously authorized integration.
  Explicit default changes update that root's already-managed PATH entries.
- PATH updates preserve value kind and unrelated strings. Added-entry ownership excludes pre-existing entries.
  A journal recovers writes interrupted before state commit, while preserving conflicting external edits.
  Backups and before/after logs stay inside the root. Notifications are sent only for the real user PATH store.
- Availability checks use the actual feed GET rather than an extra HEAD request, which some servers reject.
  Disk checks require 300 MB initially and enough for 3x ZIP size + 150 MB after the payload is downloaded.
- Alias detection uses the reparse tag. Package-family decoding is deferred; unknown probes remain informational.
- Uninstall removes owned artifacts and leaves foreign/in-use files. When running from the managed executable,
  its own locked file may need manual deletion after exit; no shell or background delete helper is launched.
- Windows resources use pinned go-winres v0.3.3 (build-time only, pure Go). The embedded XML declares asInvoker,
  long-path awareness and UTF-8; generated resource objects cover amd64, arm64 and x86.
- Local lifecycle tests exercise a clean PATH through an injected store, not the account's live PATH. A separate
  registry integration test uses the scratch key only. Fresh-account terminal UX and interactive Ctrl+C handling
  still need a human walkthrough; hosted CI has not been run here.

- Shim performance remains a tuning item: repeated local 20-run means were 17.66 ms and 15.34 ms overhead,
  slightly above the <15 ms target. Argument/stdio/exit-code behavior passed; do not claim the timing target passed.

### Sprint 3 implementation notes

- The terminal wizard uses Bubble Tea v1.3.10, with plain uncolored text and ASCII progress marks.
  No-argument interactive launches enter the wizard; redirected launches print help and exit 2.
  Advanced choices and information remain collapsed. `py`/`pyw` are never supplied.
- Same-key `install` upgrades and `upgrade SPEC`/`--all` use a durable journal. Old state stays in `pftw.json`
  until commit; its transaction ID distinguishes a committed new tree from a rollback. The complete old tree
  is kept in the journal-named trash directory until final-path checks and durable state save succeed.
  This is recoverable replacement, with a short unavailable interval, not an atomic directory exchange.
- Mutation startup and wizard preparation recover pending runtime/PATH transactions under the root lock.
  Read-only commands stay read-only. Package writes concurrent with upgrades are unsupported.
- GoReleaser v2.18.0 generates Scoop manifests but does not expose checkver/autoupdate fields. Its Scoop
  upload is disabled; the release workflow enriches the generated manifest and adds it to the release PR for this repository's
  `bucket/` (superseded: originally committed straight to main). The bootstrap manifest describes an unpublished snapshot and is not installable yet.
- Panic details go to an OS temporary `pftw-crash-*.log`; normal operations log under the chosen root.
- Local automated and real-runtime results, manual acceptance gaps and artifact details are recorded in
  `spec/sprint_validation/sprint-03-validation.md`. Hosted CI and fresh-account/conhost/Windows Terminal acceptance remain pending.


### Explicit GUI follow-up

- `pftw gui` opens a Gio v0.10.3 window on Windows, including from a terminal with redirected output.
  CLI and shim launches do not create a graphical window. Linux/macOS builds retain an unsupported-command
  response without compiling Gio. CGO remains disabled.
- Advanced-info hints are hidden while input is disabled, when there are no details, and after terminal exit.
- GUI PATH choices use the same model inputs as the TUI. User clashes have no default takeover action;
  system clashes retain default No. Selected Yes/No choices have explicit text as well as color.
- The combined stripped Windows executable is about 13.3 MB, below the 15 MB budget, using Gio's regular
  Go font face. Desktop auto-selection, DPI/accessibility coverage and the broader Sprint 6 acceptance matrix
  remain pending. See `spec/sprint_validation/gui-validation.md` for checks performed.

### Sprint 5 implementation notes

- Executable bundles carry a ZIP plus a bounded footer with offsets and an archive SHA-256. Embedded sources
  are read without extraction or index caching. Runtime hashes remain the original feed hashes. The archive
  checksum detects corruption; it is not a signature or a substitute for trusting the executable's origin.
- The embedded source precedes adjacent/configured indexes, but explicit flag/environment sources override it.
  Installation strips the overlay from the managed executable and obtains the optional windowed companion
  directly from the archive. The 15 MB budget applies to the standalone manager, not bundled runtime data.
- Mirrors create a new directory and publish the relative index after verified payloads. Output builders use
  temporary files beside their output and leave installation-root state/logs alone; the mirror manifest is
  their durable export record. This scopes the general mutation-log rule to installation/settings mutations.
- Static proxy fallback reads HKCU Internet Settings only after a direct transport failure, with no configured
  proxy or environment bypass. Explicit/environment/config proxy choices disable fallback. PAC/WPAD are not
  executed. Tests inject settings and transports; no user proxy settings are written.
- Existing golang.org/x/net/httpproxy is now a direct dependency: per-client environment snapshots avoid the
  stdlib proxy function's process-global cached environment while preserving NO_PROXY behavior; no cgo.
- SignPath onboarding/signing requires human-owned credentials and approval. A non-active workflow example
  is provided under docs; all local artifacts remain unsigned. Signed executable-overlay format acceptance
  remains pending until a real signing rehearsal is possible. No application or signing request was sent.


### Sprint 6 implementation notes

- `which [SPEC]` prints the absolute managed interpreter path; omitted SPEC uses the managed default,
  independently of project files. `--json` emits an `exe` field. It never installs or changes state.
- `install [SPEC] --target DIR` creates a new plain runtime folder. Its parent must already exist.
  Verified downloads and smoke-test scratch files use a temporary sibling, removed on success/failure.
  No manager state, shims, PATH, logging or pip bootstrap is added. Bundled payload contents remain intact.
  Conflicting managed-install flags are rejected. Dry-run resolves the feed but makes no filesystem changes.
  Failed checks remove the newly created target. A killed process may leave an incomplete target; it is
  never adopted automatically. Target installs are not managed by upgrade/uninstall.
- Keep the console PE subsystem for shell waiting, exit status and inherited handles. The conservative
  Explorer heuristic avoids guessing for unknown launchers. This iteration does not detach the console:
  an Explorer launch may show an accompanying console. GUI-subsystem/attach experiments and the desktop,
  screen-reader and DPI matrix remain acceptance work; no screen-reader compliance is claimed.
- `make security` scans with pinned govulncheck. `make deps-update` refreshes dependencies, tidies and runs
  security, vet, tests and builds. Review the module diff before committing updates. CI also scans.
- Validation records belong exclusively in `spec/sprint_validation/`; `docs/` serves users and contributors.
- Third-party signing is explicitly deferred until the owner has dogfooded these features.
