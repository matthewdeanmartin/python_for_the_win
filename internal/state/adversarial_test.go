package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRejectMalformedKnownSettings(t *testing.T) {
	for _, field := range []string{`"path":"first"`, `"path":{"mode":"maybe"}`, `"path":{"added":42}`, `"portable":"yes"`, `"config":{"proxy":42}`, `"config":{"source":[]}`, `"default":"../escape"`} {
		t.Run(field, func(t *testing.T) {
			root := t.TempDir()
			b := []byte(`{"schema":1,` + field + `}`)
			if e := os.WriteFile(filepath.Join(root, "pftw.json"), b, 0600); e != nil {
				t.Fatal(e)
			}
			if _, e := Load(root); e == nil {
				t.Fatal("accepted malformed state")
			}
			after, e := os.ReadFile(filepath.Join(root, "pftw.json"))
			if e != nil || string(after) != string(b) {
				t.Fatal("load mutated state", e)
			}
		})
	}
}
func TestSetNullInstallDoesNotPanic(t *testing.T) {
	s := New()
	s.Installs["3.15-64"] = json.RawMessage(`null`)
	if e := s.Set("3.15-64", Install{}); e == nil {
		t.Fatal("accepted null install")
	}
}
