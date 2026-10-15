package wheel

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func inputs(t *testing.T) Spec {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if e := os.WriteFile(p, []byte(content), 0o644); e != nil {
			t.Fatal(e)
		}
		return p
	}
	return Spec{Version: "0.3.0", Platform: "win_amd64", Manager: write("pftw.exe", "MZ manager"),
		Windowed: write("pftww.exe", "MZ windowed"), Readme: write("README.md", "See [docs](docs/README.md) and [site](https://example.com)."),
		License: write("LICENSE", "MIT")}
}

func read(t *testing.T, path string) map[string]string {
	r, e := zip.OpenReader(path)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = r.Close() }()
	files := map[string]string{}
	for _, f := range r.File {
		rc, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(rc)
		_ = rc.Close()
		if e != nil {
			t.Fatal(e)
		}
		files[f.Name] = string(b)
	}
	return files
}

func TestBuildProducesInstallableLayout(t *testing.T) {
	path, e := Build(t.TempDir(), inputs(t))
	if e != nil {
		t.Fatal(e)
	}
	if filepath.Base(path) != "pftw-0.3.0-py3-none-win_amd64.whl" {
		t.Fatal(path)
	}
	files := read(t, path)
	for _, name := range []string{"pftw/__init__.py", "pftw/__main__.py", "pftw-0.3.0.data/scripts/pftw.exe", "pftw-0.3.0.data/scripts/pftww.exe",
		"pftw-0.3.0.dist-info/METADATA", "pftw-0.3.0.dist-info/WHEEL", "pftw-0.3.0.dist-info/licenses/LICENSE", "pftw-0.3.0.dist-info/RECORD"} {
		if _, ok := files[name]; !ok {
			t.Error("missing", name)
		}
	}
	if !strings.Contains(files["pftw-0.3.0.dist-info/WHEEL"], "Tag: py3-none-win_amd64\n") || !strings.Contains(files["pftw-0.3.0.dist-info/WHEEL"], "Root-Is-Purelib: false") {
		t.Error(files["pftw-0.3.0.dist-info/WHEEL"])
	}
	meta := files["pftw-0.3.0.dist-info/METADATA"]
	if !strings.Contains(meta, "Name: pftw\nVersion: 0.3.0\n") || !strings.Contains(meta, "/blob/main/docs/README.md)") || !strings.Contains(meta, "(https://example.com)") {
		t.Error(meta)
	}
	// Every entry except RECORD itself is listed with a correct hash and size.
	lines := strings.Split(strings.TrimSpace(files["pftw-0.3.0.dist-info/RECORD"]), "\n")
	if len(lines) != len(files) {
		t.Fatalf("RECORD lists %d of %d files", len(lines), len(files))
	}
	for _, line := range lines {
		parts := strings.Split(line, ",")
		content, ok := files[parts[0]]
		if !ok {
			t.Fatal("RECORD names unknown file", parts[0])
		}
		if parts[0] == "pftw-0.3.0.dist-info/RECORD" {
			continue
		}
		sum := sha256.Sum256([]byte(content))
		want := fmt.Sprintf("sha256=%s,%d", base64.RawURLEncoding.EncodeToString(sum[:]), len(content))
		if parts[1]+","+parts[2] != want {
			t.Errorf("%s: %s, want %s", parts[0], line, want)
		}
	}
}

func TestBuildIsReproducibleAndRefusesOverwrite(t *testing.T) {
	s := inputs(t)
	a, e := Build(t.TempDir(), s)
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	b, e := Build(dir, s)
	if e != nil {
		t.Fatal(e)
	}
	x, _ := os.ReadFile(a)
	y, _ := os.ReadFile(b)
	if string(x) != string(y) {
		t.Error("identical inputs produced different wheels")
	}
	if _, e = Build(dir, s); e == nil {
		t.Error("existing wheel was overwritten")
	}
}

func TestBuildRejectsBadInputs(t *testing.T) {
	for _, mutate := range []func(*Spec){
		func(s *Spec) { s.Version = "dev" },
		func(s *Spec) { s.Version = "0.3.0-rc.x" },
		func(s *Spec) { s.Version = "0.3.0+local" },
		func(s *Spec) { s.Version = "0.3.0-dev.abc123" },
		func(s *Spec) { s.Platform = "manylinux_2_17_x86_64" },
		func(s *Spec) { s.Manager = filepath.Join(t.TempDir(), "missing.exe") },
	} {
		s := inputs(t)
		mutate(&s)
		dir := t.TempDir()
		if _, e := Build(dir, s); e == nil {
			t.Errorf("%+v accepted", s)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 0 {
			t.Errorf("%+v left files behind", s)
		}
	}
	for _, v := range []string{"0.3.0", "1.0.0rc1", "0.3.0.dev4", "2!1.0.post2"} {
		if !pep440.MatchString(v) {
			t.Error("rejected", v)
		}
	}
}

func TestAbsoluteLinks(t *testing.T) {
	got := AbsoluteLinks("[a](docs/x.md) [b](./spec/) [c](#top) [d](https://x.y/z)")
	want := "[a](" + repository + "/blob/main/docs/x.md) [b](" + repository + "/tree/main/spec/) [c](#top) [d](https://x.y/z)"
	if got != want {
		t.Fatal(got)
	}
}

func TestNormalizeTagVersions(t *testing.T) {
	for in, want := range map[string]string{"0.3.0": "0.3.0", "v0.3.0": "0.3.0", "0.4.0-rc.1": "0.4.0rc1", "v0.4.0-beta.2": "0.4.0b2",
		"0.4.0-alpha.01": "0.4.0a1", "0.4.0-dev.3": "0.4.0.dev3", "0.4.0rc1": "0.4.0rc1"} {
		if got, e := Normalize(in); e != nil || got != want {
			t.Errorf("%s: %q %v, want %q", in, got, e, want)
		}
	}
	for _, in := range []string{"dev", "0.3.0-dev.abc123", "0.3.0+local", "0.3.0-rc", "0.3.0-preview.1"} {
		if got, e := Normalize(in); e == nil {
			t.Errorf("%s accepted as %q", in, got)
		}
	}
	path, e := Build(t.TempDir(), func() Spec { s := inputs(t); s.Version = "v0.4.0-rc.1"; return s }())
	if e != nil || filepath.Base(path) != "pftw-0.4.0rc1-py3-none-win_amd64.whl" {
		t.Fatal(path, e)
	}
}
