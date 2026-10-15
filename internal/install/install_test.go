package install

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/proc"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"testing"
)

type fakeRunner struct {
	calls, fail int
	paths       []string
}

func (f *fakeRunner) Run(_ context.Context, exe string, args []string, o proc.Options) (string, string, int, error) {
	f.calls++
	f.paths = append(f.paths, exe)
	if f.calls == f.fail {
		return "", "injected failure", 1, fmt.Errorf("injected")
	}
	return "3.15.0\n", "", 0, nil
}
func payload(t *testing.T) feed.Entry {
	t.Helper()
	p := filepath.Join(t.TempDir(), "payload.zip")
	f, e := os.Create(p)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(f)
	w, _ := z.Create("python.exe")
	_, _ = w.Write([]byte("fixture"))
	_ = z.Close()
	_ = f.Close()
	b, _ := os.ReadFile(p)
	return feed.Entry{ID: "pythoncore-3.15-64", Version: "3.15.0", Tag: "3.15-64", URL: p, Hash: map[string]string{"sha256": fmt.Sprintf("%x", sha256.Sum256(b))}}
}
func TestInstallRollbackEveryCheck(t *testing.T) {
	entry := payload(t)
	for fail := 1; fail <= 8; fail++ {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			r := t.TempDir()
			d, _ := download.New("", false)
			runner := &fakeRunner{fail: fail}
			m := Manager{Root: r, Download: d, Runner: runner}
			if _, e := m.Install(context.Background(), entry, "fixture", false); e == nil {
				t.Fatal("failure not reached", runner.calls)
			}
			s, e := state.Load(r)
			if e != nil || len(s.Installs) != 0 {
				t.Fatal(s, e)
			}
			if _, e = os.Stat(filepath.Join(r, "versions", "3.15-64")); !os.IsNotExist(e) {
				t.Fatal("failed install retained")
			}
		})
	}
}
func TestInstallAndUninstall(t *testing.T) {
	r := t.TempDir()
	d, _ := download.New("", false)
	runner := &fakeRunner{}
	m := Manager{Root: r, Download: d, Runner: runner}
	entry := payload(t)
	if _, e := m.Install(context.Background(), entry, "fixture", false); e != nil {
		t.Fatal(e)
	}
	s, e := state.Load(r)
	if e != nil || s.Default != "3.15-64" {
		t.Fatal(s, e)
	}
	if filepath.Dir(runner.paths[5]) != filepath.Join(r, "versions", "3.15-64", "Scripts") {
		t.Fatal("pip not tested at final path", runner.paths)
	}
	before := runner.calls
	if _, e = m.Install(context.Background(), entry, "fixture", false); e != nil || runner.calls != before {
		t.Fatal("not idempotent", e)
	}
	if e = m.Uninstall("3.15-64"); e != nil {
		t.Fatal(e)
	}
	s, e = state.Load(r)
	if e != nil || len(s.Installs) != 0 {
		t.Fatal(s, e)
	}
}
func TestRecoveryBoundaries(t *testing.T) {
	for _, op := range []string{"install", "uninstall"} {
		for _, published := range []bool{false, true} {
			for _, committed := range []bool{false, true} {
				if committed && !published {
					continue
				}
				t.Run(fmt.Sprintf("%s/%t/%t", op, published, committed), func(t *testing.T) {
					r := t.TempDir()
					key := "3.15-64"
					id := "pftw-crashed"
					st := state.New()
					v := state.Install{Version: "3.15.0", Dir: filepath.Join("versions", key), Transaction: id}
					dest := filepath.Join(r, "versions", key)
					stage := filepath.Join(r, "staging", id)
					trash := filepath.Join(r, "trash", id)
					for _, p := range []string{filepath.Dir(dest), filepath.Dir(stage), filepath.Dir(trash)} {
						_ = os.MkdirAll(p, 0700)
					}
					if op == "install" {
						if published {
							_ = os.Mkdir(dest, 0700)
						} else {
							_ = os.Mkdir(stage, 0700)
						}
						if committed {
							_ = st.Set(key, v)
						}
					} else {
						if published {
							_ = os.Mkdir(trash, 0700)
						} else {
							_ = os.Mkdir(dest, 0700)
						}
						if !committed {
							_ = st.Set(key, v)
						}
					}
					if e := st.Save(r); e != nil {
						t.Fatal(e)
					}
					if e := saveTransaction(r, transaction{Operation: op, Key: key, Stage: id, ID: id}); e != nil {
						t.Fatal(e)
					}
					if e := Recover(r); e != nil {
						t.Fatal(e)
					}
					_, e := os.Stat(dest)
					want := (op == "install" && committed) || (op == "uninstall" && !committed)
					if (e == nil) != want {
						t.Fatal("wrong recovered tree", e)
					}
					if e := Recover(r); e != nil {
						t.Fatal("recovery not idempotent", e)
					}
				})
			}
		}
	}
}

func TestRecoveryRetainsDataOnInvalidJournal(t *testing.T) {
	r := t.TempDir()
	p := filepath.Join(r, "staging", "pftw-only-copy")
	_ = os.MkdirAll(p, 0700)
	_ = os.WriteFile(filepath.Join(p, "package"), []byte("data"), 0600)
	_ = os.WriteFile(filepath.Join(r, "transaction.json"), []byte(`{"Operation":"install","Key":"../bad"}`), 0600)
	if e := Recover(r); e == nil {
		t.Fatal("invalid journal accepted")
	}
	if e := Cleanup(r); e == nil {
		t.Fatal("cleaned with pending recovery")
	}
	if _, e := os.Stat(filepath.Join(p, "package")); e != nil {
		t.Fatal("lost recovery data", e)
	}
}
