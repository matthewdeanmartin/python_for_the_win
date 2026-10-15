package shims

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	s := state.New()
	s.Default = "3.15-64"
	for _, k := range []string{"3.15-64", "3.16t-arm64", "3.16-32"} {
		_ = s.Set(k, state.Install{Dir: filepath.Join("versions", k)})
	}
	for _, tc := range []struct{ name, arch, want string }{{"python3.exe", "amd64", "3.15-64"}, {"python3.15.exe", "amd64", "3.15-64"}, {"python3.16t.exe", "arm64", "3.16t-arm64"}, {"python3.16-32.exe", "amd64", "3.16-32"}} {
		got, e := Resolve(s, tc.name, tc.arch)
		if e != nil || got != tc.want {
			t.Fatal(got, e)
		}
	}
	if _, e := Resolve(s, "python3.16.exe", "amd64"); e == nil {
		t.Fatal("selected wrong architecture")
	}
}
func TestRawTail(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{{`"C:\a b\python3.exe" -c "print(1)" "a b"`, ` -c "print(1)" "a b"`}, {`C:\x.exe "c\"d"`, ` "c\"d"`}, {`"C:\x.exe"`, ""}} {
		if got := Tail(tc.raw); got != tc.want {
			t.Fatal(got, tc.want)
		}
	}
}
func TestSyncPreservesForeignFiles(t *testing.T) {
	r := t.TempDir()
	self := filepath.Join(t.TempDir(), "pftw.exe")
	_ = os.WriteFile(self, []byte("exe"), 0700)
	s := state.New()
	s.Default = "3.15-64"
	_ = s.Set(s.Default, state.Install{Dir: filepath.Join("versions", s.Default), Arch: "amd64"})
	if e := Sync(r, self, s); e != nil {
		t.Fatal(e)
	}
	foreign := filepath.Join(r, "shims", "foreign.txt")
	_ = os.WriteFile(foreign, []byte("keep"), 0600)
	if e := Sync(r, self, s); e != nil {
		t.Fatal(e)
	}
	s.Default = ""
	delete(s.Installs, "3.15-64")
	if e := Sync(r, self, s); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(foreign); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(r, "shims", "python3.exe")); !os.IsNotExist(e) {
		t.Fatal("stale shim remains")
	}
}

type noLinks struct{}

func (noLinks) Link(string, string) error { return os.ErrPermission }
func TestCopyFallback(t *testing.T) {
	r := t.TempDir()
	self := filepath.Join(t.TempDir(), "pftw.exe")
	_ = os.WriteFile(self, []byte("exe"), 0700)
	s := state.New()
	s.Default = "3.15-64"
	_ = s.Set(s.Default, state.Install{Dir: filepath.Join("versions", s.Default), Arch: "amd64"})
	if e := SyncWith(r, self, s, noLinks{}); e != nil {
		t.Fatal(e)
	}
	a, e := os.Stat(filepath.Join(r, "shims", "pftw.exe"))
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.Stat(filepath.Join(r, "shims", "python3.exe"))
	if e != nil {
		t.Fatal(e)
	}
	if os.SameFile(a, b) {
		t.Fatal("expected copy")
	}
	data, e := os.ReadFile(filepath.Join(r, "shims", "python3.exe"))
	if e != nil || string(data) != "exe" {
		t.Fatal(e)
	}
}

func TestNeverOwnsPyNamespace(t *testing.T) {
	for _, name := range []string{"py.exe", "pyw.exe", "py3.exe", "PY.EXE"} {
		if IsName(name) {
			t.Fatalf("must not dispatch %s as a pftw shim", name)
		}
	}
}

func TestUpdateManagerWithNoRuntimes(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(t.TempDir(), "pftw.exe")
	second := filepath.Join(t.TempDir(), "pftw.exe")
	if e := os.WriteFile(first, []byte("first"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(second, []byte("second"), 0700); e != nil {
		t.Fatal(e)
	}
	s := state.New()
	if e := Sync(root, first, s); e != nil {
		t.Fatal(e)
	}
	if e := Sync(root, second, s); e != nil {
		t.Fatal(e)
	}
}
