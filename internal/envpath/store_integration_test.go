//go:build integration && windows

package envpath

import (
	"golang.org/x/sys/windows/registry"
	"os"
	"testing"
)

func TestScratchRegistry(t *testing.T) {
	if os.Getenv("PFTW_INTEGRATION") != "1" {
		t.Skip("integration disabled")
	}
	const path = `Software\pftw-test\Environment`
	// Test-only scratch storage; never touch the real user PATH in tests.
	k, _, e := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
	if e != nil {
		t.Fatal(e)
	}
	_ = k.Close()
	t.Cleanup(func() { _ = registry.DeleteKey(registry.CURRENT_USER, path) })
	s := RegistryStore{registry.CURRENT_USER, path, "Path"}
	for _, kind := range []uint32{registry.SZ, registry.EXPAND_SZ} {
		want := Value{`%USERPROFILE%\bin;;C:\Tools;`, kind, true}
		if e = s.Set(want); e != nil {
			t.Fatal(e)
		}
		got, e := s.Get()
		if e != nil || got != want {
			t.Fatal(got, e)
		}
	}
	if e = s.Set(Value{Exists: false}); e != nil {
		t.Fatal(e)
	}
	v, e := s.Get()
	if e != nil || v.Exists {
		t.Fatal(v, e)
	}
}
