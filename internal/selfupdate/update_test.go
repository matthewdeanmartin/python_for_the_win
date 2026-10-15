package selfupdate

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseSelection(t *testing.T) {
	var archive bytes.Buffer
	z := zip.NewWriter(&archive)
	w, _ := z.Create("pftw.exe")
	_, _ = w.Write([]byte("new executable"))
	_ = z.Close()
	for _, scenario := range []string{"newer", "same", "older", "missing", "bad-hash", "rate-limit", "pre", "dev"} {
		t.Run(scenario, func(t *testing.T) {
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if scenario == "rate-limit" {
					w.WriteHeader(403)
					return
				}
				name := "pftw_1.2.0_windows_amd64.zip"
				tag := "v1.2.0"
				if scenario == "pre" {
					name = "pftw_1.3.0-beta.1_windows_amd64.zip"
					tag = "v1.3.0-beta.1"
				}
				switch r.URL.Path {
				case "/zip":
					_, _ = w.Write(archive.Bytes())
				case "/checks":
					hash := fmt.Sprintf("%x", sha256.Sum256(archive.Bytes()))
					if scenario == "bad-hash" {
						hash = strings.Repeat("0", 64)
					}
					_, _ = fmt.Fprintf(w, "%s  %s\n", hash, name)
				default:
					release := Release{Tag: tag, Pre: scenario == "pre", Assets: []Asset{{Name: name, URL: server.URL + "/zip"}, {Name: "checksums.txt", URL: server.URL + "/checks"}}}
					if scenario == "missing" {
						release.Assets = nil
					}
					if scenario == "pre" {
						_ = json.NewEncoder(w).Encode([]Release{release, {Tag: "v1.1.0"}})
					} else {
						_ = json.NewEncoder(w).Encode(release)
					}
				}
			}))
			defer server.Close()
			d, _ := download.New("", true)
			client := Client{Download: d, API: server.URL}
			current := "1.1.0"
			if scenario == "same" {
				current = "1.2.0"
			}
			if scenario == "older" {
				current = "1.10.0"
			}
			if scenario == "dev" {
				current = "dev"
			}
			exe, _, err := client.Latest(context.Background(), current, "amd64", scenario == "pre", false, t.TempDir())
			bad := scenario == "missing" || scenario == "bad-hash" || scenario == "rate-limit" || scenario == "dev"
			if (err != nil) != bad {
				t.Fatal(exe, err)
			}
			if scenario == "newer" || scenario == "pre" {
				b, e := os.ReadFile(exe)
				if e != nil || string(b) != "new executable" {
					t.Fatal(string(b), e)
				}
			}
			if (scenario == "same" || scenario == "older") && exe != "" {
				t.Fatal("downgrade/update", exe)
			}
			if scenario == "rate-limit" && !strings.Contains(err.Error(), "rate limit") {
				t.Fatal(err)
			}
		})
	}
}
func TestReplaceAndScoop(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "pftw.exe")
	new := filepath.Join(root, "download.exe")
	_ = os.WriteFile(old, []byte("old"), 0700)
	_ = os.WriteFile(new, []byte("new"), 0700)
	if e := Replace(new, old); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(old)
	backup, _ := os.ReadFile(old + ".old")
	if string(b) != "new" || string(backup) != "old" {
		t.Fatal(string(b), string(backup))
	}
	if e := Replace(new, old); e == nil {
		t.Fatal("overwrote backup")
	}
	if !Scoop(`C:\Users\me\scoop\apps\pftw\current\pftw.exe`) || Scoop(`C:\apps\pftw.exe`) {
		t.Fatal("scoop detection")
	}
}
