package install

import (
	"context"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/archive"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/fsops"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/proc"
	"github.com/matthewdeanmartin/python_for_the_win/internal/pyver"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manager struct {
	Rename     func(string, string) error
	SaveState  func(*state.State, string) error
	Fresh      bool
	Checkpoint func(string) error
	FreeBytes  func(string) (uint64, error)
	Root       string
	Download   *download.Client
	Runner     proc.Runner
	Log        *log.Logger
	Step       func(string)
}

func (m *Manager) step(s string) {
	if m.Log != nil {
		m.Log.Print(s)
	}
	if m.Step != nil {
		m.Step(s)
	}
}
func (m *Manager) run(ctx context.Context, exe string, args []string, cwd string) (string, error) {
	env := []string{}
	for _, v := range os.Environ() {
		upper := strings.ToUpper(v)
		if !strings.HasPrefix(upper, "TEMP=") && !strings.HasPrefix(upper, "TMP=") && !strings.HasPrefix(upper, "PYTHON") {
			env = append(env, v)
		}
	}
	env = append(env, "TEMP="+cwd, "TMP="+cwd)
	out, errout, code, e := m.Runner.Run(ctx, exe, args, proc.Options{Dir: cwd, Timeout: 90 * time.Second, Env: env})
	if e != nil || code != 0 {
		return out, fmt.Errorf("python check failed (%d): %v\n%s%s", code, e, out, errout)
	}
	return out, nil
}
func (m *Manager) smoke(ctx context.Context, dir, version, cwd string) error {
	out, e := m.run(ctx, filepath.Join(dir, "python.exe"), []string{"-I", "-c", "import sys,ssl,sqlite3,venv; print(sys.version.split()[0])"}, cwd)
	if e != nil {
		return e
	}
	if strings.TrimSpace(out) != version {
		return fmt.Errorf("runtime version mismatch: expected %s, got %q", version, strings.TrimSpace(out))
	}
	return nil
}
func (m *Manager) Install(ctx context.Context, entry feed.Entry, source string, noPip bool) (message string, err error) {
	defer func() {
		if err != nil {
			err = sharingError(err)
		}
	}()
	key := entry.Key()
	if !layout.ValidKey(key) {
		return "", errs.New(6, "invalid runtime install key")
	}
	s, e := state.Load(m.Root)
	if e != nil {
		return "", e
	}
	upgrading := false
	if _, ok := s.Installs[key]; ok {
		v, e := s.Get(key)
		if e != nil {
			return "", e
		}
		a, _ := pyver.Parse(v.Version)
		b, _ := pyver.Parse(entry.Version)
		if b.Compare(a) > 0 {
			upgrading = true
			noPip = !v.Pip
		}
		if !upgrading {
			return "Already installed: " + v.Version, nil
		}
	}
	dest, e := layout.Safe(m.Root, "versions", key)
	if e != nil {
		return "", e
	}
	if _, e = os.Lstat(dest); !upgrading && !os.IsNotExist(e) {
		return "", errs.New(3, "target already exists or is inaccessible: %s", dest)
	}
	for _, d := range []string{"cache", "staging", "versions", "trash"} {
		p, e := layout.Safe(m.Root, d)
		if e != nil {
			return "", e
		}
		if e = os.MkdirAll(p, 0700); e != nil {
			return "", e
		}
	}
	m.step("Downloading and verifying Python " + entry.Version)
	zip, e := m.Download.Fetch(ctx, entry.URL, entry.Hash["sha256"], filepath.Join(m.Root, "cache"))
	if e != nil {
		return "", e
	}
	stage, e := os.MkdirTemp(filepath.Join(m.Root, "staging"), "pftw-")
	if e != nil {
		return "", e
	}
	name := filepath.Base(stage)
	journaled := false
	defer func() {
		if !journaled {
			_ = layout.Remove(m.Root, "staging", name)
		}
	}()
	if m.FreeBytes != nil {
		if size, err := os.Stat(zip); err == nil {
			if free, err := m.FreeBytes(m.Root); err == nil && free < uint64(3*size.Size()+150*1024*1024) {
				return "", errs.New(3, "Not enough free space to extract and validate this runtime; free space and retry")
			}
		}
	}
	m.step("Extracting runtime")
	if e = archive.Extract(zip, stage); e != nil {
		return "", e
	}
	cwd, e := os.MkdirTemp(filepath.Join(m.Root, "staging"), "check-")
	if e != nil {
		return "", e
	}
	defer func() { _ = layout.Remove(m.Root, "staging", filepath.Base(cwd)) }()
	m.step("Checking staged Python")
	if e = m.smoke(ctx, stage, entry.Version, cwd); e != nil {
		return "", e
	}
	if upgrading && !m.Fresh {
		m.step("Copying installed packages and Scripts")
		for _, rel := range []string{filepath.Join("Lib", "site-packages"), "Scripts"} {
			if e = copyPackages(m.Root, filepath.Join(dest, rel), filepath.Join(stage, rel)); e != nil {
				return "", e
			}
		}
	}
	t := transaction{Operation: "install", Key: key, Stage: name, ID: name}
	if upgrading {
		t.Operation = "upgrade"
		t.Phase = "prepared"
	}
	if e = saveTransaction(m.Root, t); e != nil {
		return "", e
	}
	journaled = true
	fail := func(cause error) (string, error) {
		if re := Recover(m.Root); re != nil {
			return "", fmt.Errorf("%w; recovery pending: %v", cause, re)
		}
		return "", cause
	}
	checkpoint := func(phase string) error {
		if upgrading {
			t.Phase = phase
			if e := saveTransaction(m.Root, t); e != nil {
				return e
			}
		}
		if m.Checkpoint != nil {
			return m.Checkpoint(phase)
		}
		return nil
	}
	if e = checkpoint("prepared"); e != nil {
		return fail(e)
	}
	if upgrading {
		if e = m.rename(dest, filepath.Join(m.Root, "trash", name)); e != nil {
			return fail(sharingError(e))
		}
		if e = checkpoint("backed-up"); e != nil {
			return fail(e)
		}
	}
	if e = m.rename(stage, dest); e != nil {
		return fail(sharingError(e))
	}
	if e = checkpoint("published"); e != nil {
		return fail(e)
	}
	m.step("Checking final installation")
	if e = m.smoke(ctx, dest, entry.Version, cwd); e != nil {
		return fail(e)
	}
	exe := filepath.Join(dest, "python.exe")
	if !noPip {
		m.step("Installing and checking pip at its final path")
		if _, e = m.run(ctx, exe, []string{"-I", "-m", "ensurepip", "--upgrade", "--default-pip"}, cwd); e != nil {
			return fail(e)
		}
		// ensurepip can find a bundled pip already installed. Regenerate launchers offline from its wheel.
		script := `import ensurepip, pathlib, subprocess, sys
wheel = next((pathlib.Path(ensurepip.__file__).parent / '_bundled').glob('pip-*.whl'))
subprocess.check_call([sys.executable, '-I', '-m', 'pip', 'install', '--isolated', '--no-index', '--no-deps', '--no-cache-dir', '--force-reinstall', str(wheel)])`
		if _, e = m.run(ctx, exe, []string{"-I", "-c", script}, cwd); e != nil {
			return fail(e)
		}
		if _, e = m.run(ctx, exe, []string{"-I", "-m", "pip", "--version"}, cwd); e != nil {
			return fail(e)
		}
		if _, e = m.run(ctx, filepath.Join(dest, "Scripts", "pip.exe"), []string{"--version"}, cwd); e != nil {
			return fail(e)
		}
	}
	venv := filepath.Join(cwd, "venv")
	if _, e = m.run(ctx, exe, []string{"-I", "-m", "venv", "--without-pip", venv}, cwd); e != nil {
		return fail(e)
	}
	if _, e = m.run(ctx, filepath.Join(venv, "Scripts", "python.exe"), []string{"-I", "-c", "import sys; assert sys.prefix != sys.base_prefix"}, cwd); e != nil {
		return fail(e)
	}
	arch := "amd64"
	if strings.HasSuffix(key, "-32") {
		arch = "x86"
	}
	if strings.HasSuffix(key, "-arm64") {
		arch = "arm64"
	}
	aliases := []string{}
	for _, a := range entry.Aliases {
		aliases = append(aliases, a.Name)
	}
	v := state.Install{ID: entry.ID, Version: entry.Version, Arch: arch, Threaded: strings.Contains(key, "t-"), Dir: filepath.Join("versions", key), URL: entry.URL, SHA256: entry.Hash["sha256"], Aliases: aliases, InstalledAt: time.Now().UTC().Format(time.RFC3339), Pip: !noPip, Source: source, Transaction: t.ID}
	if e = s.Set(key, v); e != nil {
		return fail(e)
	}
	if s.Default == "" {
		s.Default = key
	}
	if e = checkpoint("validated"); e != nil {
		return fail(e)
	}
	if e = m.save(s); e != nil {
		return fail(e)
	}
	if e = checkpoint("committed"); e != nil {
		return "", e
	}
	if e = Recover(m.Root); e != nil {
		return "", e
	}
	m.step("Installation committed")
	return fmt.Sprintf("Installed Python %s. PATH was not changed.\nRun: \"%s\" --version", entry.Version, exe), nil
}

func (m *Manager) rename(old, new string) error {
	if m.Rename != nil {
		return m.Rename(old, new)
	}
	return fsops.Rename(old, new)
}
func (m *Manager) save(s *state.State) error {
	if m.SaveState != nil {
		return m.SaveState(s, m.Root)
	}
	return s.Save(m.Root)
}
