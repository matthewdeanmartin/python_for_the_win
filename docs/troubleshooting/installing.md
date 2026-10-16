# Installing

## "Windows protected your PC"

SmartScreen warns about programs that aren't code-signed or aren't yet widely downloaded. pftw isn't signed
yet ([plan](../signing.md)). Make sure you downloaded it from this project's GitHub releases, PyPI or Scoop,
then choose **More info → Run anyway**.

If your organisation blocks unsigned programs entirely, you can't use the downloaded exe there. The
[PyPI route](../installing.md#already-have-python) and the [install scripts](../install-scripts.md) run the
same program, so the same policy may apply.

## "Run pftw as yourself, not as administrator"

pftw installs for your user only and refuses to run elevated: files created by an administrator process can
end up owned by the Administrators group, and a later normal-user upgrade could then fail. Open a normal
terminal (not "Run as administrator") and try again.

If you really mean to install as the administrator account itself, pass `--allow-admin`.

## "Not enough free disk space"

pftw needs at least 300 MB free on the drive holding its folder before it starts. After downloading, it checks
again for about three times the download size plus 150 MB, because it extracts, tests and keeps the previous
version until the new one works. Free some space, or install elsewhere with `--root D:\pftw`.

## "Installation target is not owned by pftw"

The folder pftw wants to use (for example `%LOCALAPPDATA%\pftw\versions\3.14-64`) already exists, but pftw's
records don't say it created it. pftw never takes over folders it didn't create. Move or delete that folder
yourself after checking what's in it, or use another `--root`.

## "another pftw is running"

Only one pftw can change an installation at a time; it holds `pftw.lock` in its folder while it works. Wait
for the other one to finish. If none is running (check Task Manager for `pftw.exe`), the lock is released
automatically when the process ends; a leftover lock file on its own does not block anything.

## An install or upgrade was interrupted

pftw records each step before doing it. If it's interrupted (closed window, Ctrl+C, power loss, crash), the
next pftw command that changes anything first finishes or undoes the interrupted work, then continues. You'll
end up with either the complete old installation or the complete new one.

- `pftw doctor` shows "Interrupted operation awaits recovery" until then.
- `--dry-run` previews refuse to guess while recovery is pending ("Pending recovery at ...").
- Don't delete `staging`, `trash` or `transaction.json` in pftw's folder by hand: they hold the data needed
  to recover.

Protection against power loss depends on the drive and filesystem flushing data as promised; ordinary
interruptions are tested, power loss is not.

## "PATH information is unavailable"

pftw couldn't read your user or system PATH (for example because of a group policy or a damaged registry
value), so it can't tell which Python would win. Install with `--path none` and run Python by its full path
or through `pftw run`; `pftw which` prints the path.

## "Installation root contains characters that cannot be represented safely in PATH"

PATH entries are separated by `;`, so a folder whose name contains `;` or `"` can't be added safely. Choose
another folder with `--root`, or use `--path none`.

## "Installation root is on a network share"

This is a warning, not an error. Running programs from network drives can be slow, and file locking and
antivirus behaviour differ. A local drive is strongly recommended.

## The checks after installing failed

Before reporting success, pftw runs the new Python, imports `ssl`, `sqlite3` and `venv`, installs and runs pip,
and creates a temporary venv. If any of that fails, the install is rolled back. Common causes:

- **Antivirus** quarantined or locked a file while it was being extracted. Check your antivirus history, allow
  the pftw folder if appropriate, and retry.
- **A broken pre-release.** Development builds (alpha, beta, rc) occasionally fail these checks. Use a final
  release.
- **Very long paths.** A deep `--root` can push files past old Windows path limits. Use a short root such as
  `C:\pftw`.

To install without pip (and skip the pip checks), pass `--no-pip`.

## "Use an explicit command when input or output is redirected"

Running `pftw` with no command opens the wizard, which needs a terminal or a desktop. When output is piped or
redirected (in scripts, CI, or some editors' task runners), pass a command such as `pftw install`.
