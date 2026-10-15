package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	r := t.TempDir()
	s := New()
	s.Raw["future"] = json.RawMessage(`{"enabled":true}`)
	s.Config["future"] = json.RawMessage(`42`)
	v := Install{Version: "3.15.0", Dir: filepath.Join("versions", "3.15-64")}
	if e := s.Set("3.15-64", v); e != nil {
		t.Fatal(e)
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(s.Installs["3.15-64"], &raw)
	raw["future"] = json.RawMessage(`"keep"`)
	s.Installs["3.15-64"], _ = json.Marshal(raw)
	if e := s.Save(r); e != nil {
		t.Fatal(e)
	}
	_ = os.WriteFile(filepath.Join(r, "pftw.json.tmp"), []byte("broken"), 0600)
	s, e := Load(r)
	if e != nil {
		t.Fatal(e)
	}
	v.Pip = true
	_ = s.Set("3.15-64", v)
	if e = s.Save(r); e != nil {
		t.Fatal(e)
	}
	s, e = Load(r)
	if e != nil || string(s.Raw["future"]) == "" || string(s.Config["future"]) != "42" {
		t.Fatal(s, e)
	}
	_ = json.Unmarshal(s.Installs["3.15-64"], &raw)
	if string(raw["future"]) != `"keep"` {
		t.Fatal(raw)
	}
}
func TestExclusiveLock(t *testing.T) {
	r := t.TempDir()
	unlock, e := Lock(r)
	if e != nil {
		t.Fatal(e)
	}
	if release, e := Lock(r); e == nil {
		release()
		t.Fatal("second lock succeeded")
	}
	unlock()
	release, e := Lock(r)
	if e != nil {
		t.Fatal(e)
	}
	release()
}
func TestUnsafeState(t *testing.T) {
	r := t.TempDir()
	_ = os.WriteFile(filepath.Join(r, "pftw.json"), []byte(`{"schema":1,"installs":{"../bad":{}}}`), 0600)
	if _, e := Load(r); e == nil {
		t.Fatal("unsafe key")
	}
}
