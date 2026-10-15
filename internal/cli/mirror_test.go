package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMirrorPreservesHashesAndRefusesExistingOutput(t *testing.T) {
	root := t.TempDir()
	payload := filepath.Join(root, "runtime.zip")
	content := []byte("verified payload")
	_ = os.WriteFile(payload, content, 0600)
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	v := feed.Entry{Schema: 1, Company: "PythonCore", Tag: "3.15-64", Version: "3.15.0", InstallFor: []string{"3.15-64"}, URL: payload, Hash: map[string]string{"sha256": hash}}
	b, _ := json.Marshal(feed.Index{Versions: []feed.Entry{v}})
	source := filepath.Join(root, "source.json")
	_ = os.WriteFile(source, b, 0600)
	out := filepath.Join(root, "mirror")
	run := func(dry bool) error {
		c := New()
		c.SetOut(new(bytes.Buffer))
		a := []string{"mirror", "3.15-64", "3.15-64", "--out", out, "--source", source, "--root", filepath.Join(root, "unused")}
		if dry {
			a = append(a, "--dry-run")
		}
		c.SetArgs(a)
		return c.Execute()
	}
	if e := run(true); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		t.Fatal("preview wrote output")
	}
	if e := run(false); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(out, "index.json"))
	if e != nil {
		t.Fatal(e)
	}
	var idx feed.Index
	_ = json.Unmarshal(b, &idx)
	if len(idx.Versions) != 1 || idx.Versions[0].URL != hash+".zip" || idx.Versions[0].Hash["sha256"] != hash {
		t.Fatal(idx)
	}
	server := httptest.NewServer(http.FileServer(http.Dir(out)))
	defer server.Close()
	client, err := download.New("", true)
	if err != nil {
		t.Fatal(err)
	}
	it, err := feed.Load(client, server.URL+"/index.json", "", true)
	if err != nil {
		t.Fatal(err)
	}
	served, err := it.All(context.Background())
	if err != nil || len(served) != 1 {
		t.Fatal(served, err)
	}
	if _, err = client.Fetch(context.Background(), served[0].URL, served[0].Hash["sha256"], t.TempDir()); err != nil {
		t.Fatal(err)
	}
	before := tree(t, out)
	if e := run(false); e == nil {
		t.Fatal("overwrote mirror")
	}
	after := tree(t, out)
	if len(before) != len(after) {
		t.Fatal("changed existing output")
	}
	if _, e := os.Stat(filepath.Join(root, "unused")); !os.IsNotExist(e) {
		t.Fatal("created install root")
	}
}
