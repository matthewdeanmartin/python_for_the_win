package resolve

import (
	"encoding/json"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"os"
	"testing"
)

func fixture(t *testing.T) []feed.Entry {
	t.Helper()
	b, e := os.ReadFile("../../testdata/index-windows.sample.json")
	if e != nil {
		t.Fatal(e)
	}
	var f feed.Index
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	out := []feed.Entry{}
	for _, v := range f.Versions {
		if v.Company == "PythonCore" && v.Schema == 1 {
			out = append(out, v)
		}
	}
	return out
}
func TestSelection(t *testing.T) {
	entries := fixture(t)
	for _, tc := range []struct {
		spec, arch, want string
		pre              bool
	}{{"3.16", "amd64", "3.16.0a1", false}, {"3.15", "amd64", "3.15.0", false}, {"3.15", "amd64", "3.15.0", true}, {"3.16t", "arm64", "3.16.0a1", false}, {"3.16.0a1", "amd64", "3.16.0a1", false}, {"latest", "amd64", "3.15.0", false}, {"", "amd64", "3.15.0", false}, {"3", "amd64", "3.15.0", false}} {
		t.Run(tc.spec+tc.arch, func(t *testing.T) {
			v, e := Pick(entries, Options{Spec: tc.spec, Arch: tc.arch, Prerelease: tc.pre})
			if e != nil || v.Version != tc.want {
				t.Fatalf("%s %v", v.Version, e)
			}
		})
	}
}
func TestNoImplicitAlpha(t *testing.T) {
	entries := fixture(t)
	onlyPre := []feed.Entry{}
	for _, v := range entries {
		if v.Version == "3.16.0a1" {
			onlyPre = append(onlyPre, v)
		}
	}
	for _, s := range []string{"", "3", "latest"} {
		if _, e := Pick(onlyPre, Options{Spec: s, Arch: "amd64"}); e == nil {
			t.Fatal("implicit alpha", s)
		}
	}
	final := onlyPre[0]
	final.Version = "3.16.0"
	final.Hash = nil
	final.Tag = "3.16-32"
	final.InstallFor = []string{"3.16-32"}
	onlyPre = append(onlyPre, final)
	if _, e := Pick(onlyPre, Options{Spec: "3.16", Arch: "amd64"}); e == nil {
		t.Fatal("fell back after final exists")
	}
}
func TestMissingHashAndInvalidSpec(t *testing.T) {
	v := fixture(t)
	for i := range v {
		v[i].Hash = nil
	}
	if _, e := Pick(v, Options{Spec: "latest", Arch: "amd64"}); e == nil {
		t.Fatal("missing hashes accepted")
	}
	if _, e := Pick(v, Options{Spec: "../bad", Arch: "amd64"}); e == nil {
		t.Fatal("invalid spec accepted")
	}
}

func TestAvailablePrefersUsableOlderBuild(t *testing.T) {
	v := fixture(t)
	var base feed.Entry
	for _, x := range v {
		if x.Key() == "3.15-64" && x.Version == "3.15.0" {
			base = x
			break
		}
	}
	bad := base
	bad.Version = "3.15.1"
	bad.Hash = nil
	for _, show := range []bool{false, true} {
		got := Available([]feed.Entry{bad, base}, "amd64", false, false, show)
		if len(got) != 1 || got[0].Version != "3.15.0" {
			t.Fatal(got)
		}
	}
	if got := Available([]feed.Entry{bad}, "amd64", false, false); len(got) != 0 {
		t.Fatal(got)
	}
	if got := Available([]feed.Entry{bad}, "amd64", false, false, true); len(got) != 1 {
		t.Fatal(got)
	}
}
