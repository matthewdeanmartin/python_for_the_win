package cli

import (
	"bytes"
	"encoding/json"
	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeHost struct {
	platform.Native
	system string
}

func (f fakeHost) SystemPath() (envpath.Value, error) {
	return envpath.Value{Text: f.system, Kind: 2, Exists: true}, nil
}
func (fakeHost) IsElevated() (bool, error)          { return false, nil }
func (fakeHost) FreeBytes(string) (uint64, error)   { return 1 << 40, nil }
func (fakeHost) IsNetworkPath(string) (bool, error) { return false, nil }
func (fakeHost) IsAppExecLink(string) (bool, error) { return false, nil }
func TestConflictRequiresExplicitChoice(t *testing.T) {
	r := t.TempDir()
	other := t.TempDir()
	_ = os.WriteFile(filepath.Join(other, "python.exe"), []byte("fake"), 0600)
	store := &envpath.MemStore{Value: envpath.Value{Text: other, Kind: 2, Exists: true}}
	o := &options{root: r, info: fakeHost{}, store: store, path: "auto", yes: true}
	c := NewWith(o.info, store, "")
	if _, e := o.choosePath(c, state.New()); e == nil {
		t.Fatal("--yes silently accepted clash")
	}
	o.path = "first"
	if mode, e := o.choosePath(c, state.New()); e != nil || mode != "first" {
		t.Fatal(mode, e)
	}
	o.info = fakeHost{system: other}
	if _, e := o.choosePath(c, state.New()); e == nil {
		t.Fatal("system clash accepted")
	}
	o.path = "none"
	if mode, e := o.choosePath(c, state.New()); e != nil || mode != "none" {
		t.Fatal(mode, e)
	}
}
func TestDoctorReadOnlyFindings(t *testing.T) {
	r := t.TempDir()
	s := state.New()
	s.Default = "3.15-64"
	_ = s.Set(s.Default, state.Install{Dir: filepath.Join("versions", s.Default), Arch: "amd64"})
	_ = s.Save(r)
	before, _ := os.ReadFile(filepath.Join(r, "pftw.json"))
	store := &envpath.MemStore{Value: envpath.Value{Text: "somewhere", Kind: 2, Exists: true}}
	c := NewWith(fakeHost{}, store, "")
	b := new(bytes.Buffer)
	c.SetOut(b)
	c.SetArgs([]string{"doctor", "--json", "--root", r})
	if e := c.Execute(); e != nil {
		t.Fatal(e)
	}
	var report struct{ Checks []struct{ ID, Status string } }
	if e := json.Unmarshal(b.Bytes(), &report); e != nil {
		t.Fatal(e)
	}
	findings := map[string]string{}
	for _, x := range report.Checks {
		findings[x.ID] = x.Status
	}
	if findings["missing-runtime"] != "Warn" || findings["missing-shim"] != "Warn" || findings["resolve-python"] != "Info" {
		t.Fatal(findings)
	}
	after, _ := os.ReadFile(filepath.Join(r, "pftw.json"))
	if !bytes.Equal(before, after) || store.Value.Text != "somewhere" {
		t.Fatal("doctor mutated state")
	}
}
func TestPortableRequiresExplicitRoot(t *testing.T) {
	c := New()
	c.SetArgs([]string{"install", "--portable"})
	if e := c.Execute(); e == nil || !strings.Contains(e.Error(), "requires --root") {
		t.Fatal(e)
	}
}

func TestFinishDoesNotPromiseShadowedPython(t *testing.T) {
	r := t.TempDir()
	s := state.New()
	s.Default = "3.15-64"
	s.SetPath(state.PathState{Mode: "first"})
	own := filepath.Join(r, "versions", s.Default)
	_ = os.MkdirAll(own, 0700)
	_ = os.WriteFile(filepath.Join(own, "python.exe"), []byte("own"), 0600)
	other := t.TempDir()
	_ = os.WriteFile(filepath.Join(other, "python.exe"), []byte("other"), 0600)
	store := &envpath.MemStore{Value: envpath.Value{Text: own}}
	o := &options{root: r, info: fakeHost{system: other}, store: store}
	if o.pathResolvesDefault(s) {
		t.Fatal("promised shadowed interpreter")
	}
	o.info = fakeHost{}
	if !o.pathResolvesDefault(s) {
		t.Fatal("missed managed interpreter")
	}
}

func TestManagerCleanupPreservesForeignFiles(t *testing.T) {
	r := t.TempDir()
	s := state.New()
	if e := s.Save(r); e != nil {
		t.Fatal(e)
	}
	foreign := filepath.Join(r, "notes.txt")
	_ = os.WriteFile(foreign, []byte("keep"), 0600)
	o := &options{root: r, quiet: true}
	if e := o.removeManager(New()); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(foreign)
	if e != nil || string(b) != "keep" {
		t.Fatal("foreign file lost", e)
	}
}
