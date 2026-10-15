//go:build windows

// Package installscripts tests install.sh and install.ps1 against a local PEP 503 index.
package installscripts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matthewdeanmartin/python_for_the_win/internal/wheel"
)

type index struct {
	server *httptest.Server
	dir    string
}

// serve publishes wheels for several versions; bad maps a version to a wrong advertised hash.
func serve(t *testing.T, exe string, bad map[string]bool) index {
	t.Helper()
	dir := t.TempDir()
	repo, _ := filepath.Abs("../..")
	var page strings.Builder
	page.WriteString("<!DOCTYPE html><html><body>\n")
	for _, v := range []string{"0.2.0", "0.3.0", "0.4.0rc1", "0.5.0"} {
		for _, plat := range []string{"win_amd64", "win_arm64"} {
			path, e := wheel.Build(dir, wheel.Spec{Version: v, Platform: plat, Manager: exe,
				Readme: filepath.Join(repo, "README.md"), License: filepath.Join(repo, "LICENSE")})
			if e != nil {
				t.Fatal(e)
			}
			b, _ := os.ReadFile(path)
			sum := sha256.Sum256(b)
			hash := hex.EncodeToString(sum[:])
			if bad[v] {
				hash = strings.Repeat("0", 64)
			}
			name := filepath.Base(path)
			href := "/files/" + name // Root-relative, like many mirrors.
			if v == "0.3.0" {
				href = "../../files/" + name // Relative, like devpi.
			}
			extra := ""
			if v == "0.5.0" {
				extra = ` data-yanked="broken"`
			}
			fmt.Fprintf(&page, "<a href=\"%s#sha256=%s\"%s>%s</a><br />\n", href, hash, extra, name)
		}
	}
	page.WriteString("</body></html>\n")
	mux := http.NewServeMux()
	mux.HandleFunc("/simple/pftw/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(page.String())) })
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(dir))))
	mux.HandleFunc("/install.sh", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, filepath.Join(repo, "install.sh")) })
	mux.HandleFunc("/install.ps1", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, filepath.Join(repo, "install.ps1")) })
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return index{s, dir}
}

// buildPftw compiles the real CLI once; every wheel carries it.
func buildPftw(t *testing.T) string {
	if testing.Short() {
		t.Skip("builds pftw and starts shells")
	}
	exe := filepath.Join(t.TempDir(), "pftw.exe")
	out, e := exec.Command("go", "build", "-o", exe, "-ldflags", "-X github.com/matthewdeanmartin/python_for_the_win/internal/cli.Version=from-wheel", "../../cmd/pftw").CombinedOutput()
	if e != nil {
		t.Fatal(string(out), e)
	}
	return exe
}

// shells returns runnable launchers for each script; WSL's bash is never used.
func shells(t *testing.T) map[string][]string {
	repo, _ := filepath.Abs("../..")
	found := map[string][]string{}
	if bash, e := exec.LookPath("bash"); e == nil && !strings.Contains(strings.ToLower(bash), `\windows\system32\`) {
		found["bash"] = []string{bash, filepath.Join(repo, "install.sh")}
	}
	for _, name := range []string{"powershell", "pwsh"} {
		if p, e := exec.LookPath(name); e == nil {
			found[name] = []string{p, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(repo, "install.ps1")}
		}
	}
	if len(found) == 0 {
		t.Skip("no Git Bash or PowerShell available")
	}
	return found
}

func run(t *testing.T, ix index, command []string, env ...string) (stdout, stderr string, code int) {
	t.Helper()
	c := exec.Command(command[0], command[1:]...)
	c.Env = append(os.Environ(), "PFTW_INDEX_URL="+ix.server.URL+"/simple", "PFTW_ALLOW_INSECURE=1", "PFTW_VERSION=", "PFTW_KEEP=")
	c.Env = append(c.Env, env...)
	var out, errOut strings.Builder
	c.Stdout, c.Stderr = &out, &errOut
	e := c.Run()
	if exit, ok := e.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if e != nil {
		t.Fatal(e)
	}
	return out.String(), errOut.String(), code
}

func TestScriptsInstallVerifiedNewestFinal(t *testing.T) {
	exe := buildPftw(t)
	ix := serve(t, exe, nil)
	for name, command := range shells(t) {
		t.Run(name, func(t *testing.T) {
			out, errOut, code := run(t, ix, append(command, "version"))
			if code != 0 || !strings.Contains(out, "pftw from-wheel") || !strings.Contains(errOut, "pftw 0.3.0 (") {
				t.Fatalf("code %d\nstdout: %s\nstderr: %s", code, out, errOut)
			}
			out, errOut, code = run(t, ix, append(command, "version"), "PFTW_VERSION=0.4.0rc1")
			if code != 0 || !strings.Contains(errOut, "pftw 0.4.0rc1 (") {
				t.Fatalf("pinned pre-release: code %d\nstdout: %s\nstderr: %s", code, out, errOut)
			}
			// The exit status of pftw itself reaches the caller.
			if _, _, code = run(t, ix, append(command, "no-such-command")); code != 2 {
				t.Fatalf("pftw usage error became exit %d", code)
			}
			keep := t.TempDir()
			if _, errOut, code = run(t, ix, append(command, "version"), "PFTW_KEEP="+keep); code != 0 {
				t.Fatal(errOut)
			}
			if _, e := os.Stat(filepath.Join(keep, "pftw.exe")); e != nil {
				t.Fatal("PFTW_KEEP did not save pftw.exe")
			}
		})
	}
}

func TestScriptsRefuseHashMismatch(t *testing.T) {
	exe := buildPftw(t)
	ix := serve(t, exe, map[string]bool{"0.3.0": true})
	for name, command := range shells(t) {
		t.Run(name, func(t *testing.T) {
			out, errOut, code := run(t, ix, append(command, "version"))
			if code == 0 || strings.Contains(out, "pftw from-wheel") || !strings.Contains(errOut, "SHA-256 mismatch") {
				t.Fatalf("tampered wheel ran: code %d\nstdout: %s\nstderr: %s", code, out, errOut)
			}
		})
	}
}

// The documented one-liners: curl | bash and irm | iex, with arguments.
func TestPipedOneLiners(t *testing.T) {
	exe := buildPftw(t)
	ix := serve(t, exe, nil)
	found := shells(t)
	if bash, ok := found["bash"]; ok {
		out, errOut, code := run(t, ix, []string{bash[0], "-c", "curl -LsSf " + ix.server.URL + "/install.sh | bash -s -- version"})
		if code != 0 || !strings.Contains(out, "pftw from-wheel") {
			t.Fatalf("curl | bash: code %d\nstdout: %s\nstderr: %s", code, out, errOut)
		}
	}
	for _, name := range []string{"powershell", "pwsh"} {
		ps, ok := found[name]
		if !ok {
			continue
		}
		script := "& ([scriptblock]::Create((irm " + ix.server.URL + "/install.ps1))) version"
		out, errOut, code := run(t, ix, []string{ps[0], "-NoProfile", "-NonInteractive", "-Command", script})
		if code != 0 || !strings.Contains(out, "pftw from-wheel") {
			t.Fatalf("%s irm | iex: code %d\nstdout: %s\nstderr: %s", name, code, out, errOut)
		}
	}
}
