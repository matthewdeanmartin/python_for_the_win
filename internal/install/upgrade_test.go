package install

import (
	"context"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func oldInstall(t *testing.T, root string) {
	t.Helper()
	key := "3.15-64"
	s := state.New()
	s.Default = key
	if e := s.Set(key, state.Install{Version: "3.15.0a1", Arch: "amd64", Dir: filepath.Join("versions", key), Pip: true, Transaction: "old"}); e != nil {
		t.Fatal(e)
	}
	if e := s.Save(root); e != nil {
		t.Fatal(e)
	}
	for _, rel := range []string{"python.exe", filepath.Join("Lib", "site-packages", "mine.py"), filepath.Join("Scripts", "mine.exe")} {
		p := filepath.Join(root, "versions", key, rel)
		if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte("old-data"), 0600); e != nil {
			t.Fatal(e)
		}
	}
}
func assertUpgrade(t *testing.T, root string, committed, fresh bool) {
	t.Helper()
	s, e := state.Load(root)
	if e != nil {
		t.Fatal(e)
	}
	v, e := s.Get("3.15-64")
	if e != nil {
		t.Fatal(e)
	}
	want := "3.15.0a1"
	if committed {
		want = "3.15.0"
	}
	if v.Version != want {
		t.Fatal(v.Version, want)
	}
	for _, rel := range []string{filepath.Join("Lib", "site-packages", "mine.py"), filepath.Join("Scripts", "mine.exe")} {
		b, e := os.ReadFile(filepath.Join(root, "versions", "3.15-64", rel))
		if committed && fresh {
			if !os.IsNotExist(e) {
				t.Fatal("fresh retained old data", e)
			}
		} else if e != nil || string(b) != "old-data" {
			t.Fatal("lost package", string(b), e)
		}
	}
}
func TestUpgradeBoundaries(t *testing.T) {
	entry := payload(t)
	for _, phase := range []string{"prepared", "backed-up", "published", "validated", "committed", "success"} {
		for _, fresh := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", phase, fresh), func(t *testing.T) {
				root := t.TempDir()
				oldInstall(t, root)
				d, _ := download.New("", false)
				m := Manager{Root: root, Download: d, Runner: &fakeRunner{}, Fresh: fresh, Checkpoint: func(p string) error {
					if p == phase {
						return fmt.Errorf("injected %s", p)
					}
					return nil
				}}
				_, e := m.Install(context.Background(), entry, "fixture", false)
				if (e == nil) != (phase == "success") {
					t.Fatal(e)
				}
				if e = Recover(root); e != nil {
					t.Fatal(e)
				}
				if e = Recover(root); e != nil {
					t.Fatal(e)
				}
				assertUpgrade(t, root, phase == "committed" || phase == "success", fresh)
			})
		}
	}
}
func TestUpgradeChecksRollback(t *testing.T) {
	for fail := 1; fail <= 8; fail++ {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			root := t.TempDir()
			oldInstall(t, root)
			d, _ := download.New("", false)
			m := Manager{Root: root, Download: d, Runner: &fakeRunner{fail: fail}}
			if _, e := m.Install(context.Background(), payload(t), "fixture", false); e == nil {
				t.Fatal("missing failure")
			}
			assertUpgrade(t, root, false, false)
		})
	}
}
func TestUpgradeProcessInterruption(t *testing.T) {
	if phase := os.Getenv("PFTW_UPGRADE_CRASH"); phase != "" {
		root := os.Getenv("PFTW_CRASH_ROOT")
		_, e := state.Lock(root)
		if e != nil {
			os.Exit(81)
		}
		d, _ := download.New("", false)
		m := Manager{Root: root, Download: d, Runner: &fakeRunner{}, Checkpoint: func(p string) error {
			if p == phase {
				os.Exit(77)
			}
			return nil
		}}
		_, _ = m.Install(context.Background(), payload(t), "fixture", false)
		os.Exit(82)
	}
	exe, _ := os.Executable()
	for _, phase := range []string{"prepared", "backed-up", "published", "validated", "committed"} {
		t.Run(phase, func(t *testing.T) {
			root := t.TempDir()
			oldInstall(t, root)
			cmd := exec.Command(exe, "-test.run=^TestUpgradeProcessInterruption$")
			cmd.Env = append(os.Environ(), "PFTW_UPGRADE_CRASH="+phase, "PFTW_CRASH_ROOT="+root)
			out, e := cmd.CombinedOutput()
			if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 77 {
				t.Fatal(e, string(out))
			}
			unlock, e := state.Lock(root)
			if e != nil {
				t.Fatal(e)
			}
			defer unlock()
			if e = Recover(root); e != nil {
				t.Fatal(e)
			}
			assertUpgrade(t, root, phase == "committed", false)
		})
	}
}

func TestUpgradePublicationIOFailures(t *testing.T) {
	for _, boundary := range []string{"backup-rename", "publish-rename", "state-save"} {
		t.Run(boundary, func(t *testing.T) {
			root := t.TempDir()
			oldInstall(t, root)
			d, _ := download.New("", false)
			count := 0
			m := Manager{Root: root, Download: d, Runner: &fakeRunner{}}
			m.Rename = func(a, b string) error {
				count++
				if boundary == "backup-rename" && count == 1 || boundary == "publish-rename" && count == 2 {
					return fmt.Errorf("injected rename failure")
				}
				return os.Rename(a, b)
			}
			m.SaveState = func(s *state.State, r string) error {
				if boundary == "state-save" {
					return fmt.Errorf("injected durable state write failure")
				}
				return s.Save(r)
			}
			if _, e := m.Install(context.Background(), payload(t), "fixture", false); e == nil {
				t.Fatal("missing error")
			}
			if e := Recover(root); e != nil {
				t.Fatal(e)
			}
			assertUpgrade(t, root, false, false)
		})
	}
}
