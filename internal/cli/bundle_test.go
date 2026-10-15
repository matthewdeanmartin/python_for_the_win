package cli

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
)

func TestBundleContainsVerifiedRelativePayload(t *testing.T) {
	root := t.TempDir()
	payload := filepath.Join(root, "runtime.zip")
	content := []byte("payload bytes verified independently of runtime extraction")
	_ = os.WriteFile(payload, content, 0600)
	entry := feed.Entry{Schema: 1, Company: "PythonCore", Tag: "3.15-64", Version: "3.15.0", InstallFor: []string{"3.15-64"}, URL: payload, Hash: map[string]string{"sha256": fmt.Sprintf("%x", sha256.Sum256(content))}}
	source := filepath.Join(root, "index.json")
	b, _ := json.Marshal(feed.Index{Versions: []feed.Entry{entry}})
	_ = os.WriteFile(source, b, 0600)
	out := filepath.Join(root, "offline.zip")
	c := New()
	c.SetOut(new(bytes.Buffer))
	c.SetArgs([]string{"bundle", "3.15-64", "--out", out, "--source", source, "--root", filepath.Join(root, "unused")})
	if e := c.Execute(); e != nil {
		t.Fatal(e)
	}
	z, e := zip.OpenReader(out)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	found := map[string]bool{}
	for _, f := range z.File {
		found[f.Name] = true
		if f.Name == "index.json" {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			var index feed.Index
			e = json.NewDecoder(r).Decode(&index)
			r.Close()
			if e != nil {
				t.Fatal(e)
			}
			v := index.Versions[0]
			if v.URL != entry.Hash["sha256"]+".zip" || v.Hash["sha256"] != entry.Hash["sha256"] {
				t.Fatal(v)
			}
		}
	}
	if !found["pftw.exe"] || !found[entry.Hash["sha256"]+".zip"] {
		t.Fatal(found)
	}
	if _, e := os.Stat(filepath.Join(root, "unused")); !os.IsNotExist(e) {
		t.Fatal("bundle mutated install root")
	}
	_ = os.WriteFile(payload, []byte("tampered"), 0600)
	c = New()
	c.SetOut(new(bytes.Buffer))
	bad := filepath.Join(root, "bad.zip")
	c.SetArgs([]string{"bundle", "3.15-64", "--out", bad, "--source", source})
	if e := c.Execute(); e == nil {
		t.Fatal("accepted bad hash")
	}
	if _, e := os.Stat(bad); !os.IsNotExist(e) {
		t.Fatal("published bad bundle")
	}
}
