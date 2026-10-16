// Package wheel packages prebuilt pftw executables as platform wheels for PyPI.
// pip copies the executables into the environment's Scripts folder; no Python runs pftw itself.
package wheel

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	Distribution = "pftw"
	repository   = "https://github.com/matthewdeanmartin/python_for_the_win"
)

//go:embed launcher/__init__.py launcher/__main__.py
var launcher embed.FS

// Platforms maps Go architectures to wheel platform tags.
var Platforms = map[string]string{"amd64": "win_amd64", "arm64": "win_arm64", "386": "win32"}

// PEP 440 public versions, without local segments (PyPI rejects those).
var pep440 = regexp.MustCompile(`^([1-9][0-9]*!)?(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))*((a|b|rc)(0|[1-9][0-9]*))?(\.post(0|[1-9][0-9]*))?(\.dev(0|[1-9][0-9]*))?$`)

// Fixed timestamps keep wheels reproducible.
var stamp = time.Date(1980, 1, 2, 0, 0, 0, 0, time.UTC)

type Spec struct {
	Version  string
	Platform string // win_amd64, win_arm64 or win32
	Manager  string // pftw.exe
	Windowed string // optional pftww.exe
	Readme   string
	License  string
}

type file struct {
	name string
	data []byte
}

// Build writes the wheel into dir and returns its path. Existing wheels are not overwritten.
func Build(dir string, s Spec) (string, error) {
	version, e := Normalize(s.Version)
	if e != nil {
		return "", e
	}
	s.Version = version
	known := false
	for _, tag := range Platforms {
		known = known || tag == s.Platform
	}
	if !known {
		return "", fmt.Errorf("unsupported platform tag %q", s.Platform)
	}
	read := func(path string) ([]byte, error) {
		b, e := os.ReadFile(path)
		if e != nil {
			return nil, fmt.Errorf("read %s: %w", path, e)
		}
		return b, nil
	}
	manager, e := read(s.Manager)
	if e != nil {
		return "", e
	}
	readme, e := read(s.Readme)
	if e != nil {
		return "", e
	}
	license, e := read(s.License)
	if e != nil {
		return "", e
	}
	data := Distribution + "-" + s.Version + ".data/scripts/"
	info := Distribution + "-" + s.Version + ".dist-info/"
	files := []file{}
	for _, name := range []string{"__init__.py", "__main__.py"} {
		b, e := launcher.ReadFile("launcher/" + name)
		if e != nil {
			return "", e
		}
		files = append(files, file{"pftw/" + name, b})
	}
	files = append(files, file{data + "pftw.exe", manager})
	if s.Windowed != "" {
		windowed, e := read(s.Windowed)
		if e != nil {
			return "", e
		}
		files = append(files, file{data + "pftww.exe", windowed})
	}
	files = append(files,
		file{info + "METADATA", Metadata(s.Version, readme)},
		file{info + "WHEEL", []byte("Wheel-Version: 1.0\nGenerator: pftw-wheel\nRoot-Is-Purelib: false\nTag: py3-none-" + s.Platform + "\n")},
		file{info + "licenses/LICENSE", license},
	)
	record := &bytes.Buffer{}
	for _, f := range files {
		sum := sha256.Sum256(f.data)
		fmt.Fprintf(record, "%s,sha256=%s,%d\n", f.name, base64.RawURLEncoding.EncodeToString(sum[:]), len(f.data))
	}
	fmt.Fprintf(record, "%sRECORD,,\n", info)
	files = append(files, file{info + "RECORD", record.Bytes()})

	if e = os.MkdirAll(dir, 0o755); e != nil {
		return "", e
	}
	path := filepath.Join(dir, fmt.Sprintf("%s-%s-py3-none-%s.whl", Distribution, s.Version, s.Platform))
	out, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if e != nil {
		return "", e
	}
	w := zip.NewWriter(out)
	for _, f := range files {
		entry, e := w.CreateHeader(&zip.FileHeader{Name: f.name, Method: zip.Deflate, Modified: stamp})
		if e == nil {
			_, e = entry.Write(f.data)
		}
		if e != nil {
			_ = out.Close()
			_ = os.Remove(path)
			return "", e
		}
	}
	e = w.Close()
	if ce := out.Close(); e == nil {
		e = ce
	}
	if e != nil {
		_ = os.Remove(path)
		return "", e
	}
	return path, nil
}

// Metadata returns core metadata 2.1 with the README as the PyPI project description.
func Metadata(version string, readme []byte) []byte {
	b := &strings.Builder{}
	for _, line := range []string{
		"Metadata-Version: 2.1",
		"Name: " + Distribution,
		"Version: " + version,
		"Summary: Install official Python on Windows without the Store, MSI or Python Install Manager",
		"Author: Matthew Dean Martin",
		"License: MIT",
		"Project-URL: Homepage, " + repository,
		"Project-URL: Documentation, https://python-for-the-win.readthedocs.io/",
		"Project-URL: Changelog, " + repository + "/blob/main/CHANGELOG.md",
		"Classifier: Development Status :: 3 - Alpha",
		"Classifier: Environment :: Console",
		"Classifier: Intended Audience :: Developers",
		"Classifier: License :: OSI Approved :: MIT License",
		"Classifier: Operating System :: Microsoft :: Windows",
		"Classifier: Topic :: System :: Installation/Setup",
		"Classifier: Topic :: Software Development :: Interpreters",
		"Description-Content-Type: text/markdown",
	} {
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(AbsoluteLinks(string(readme)))
	return []byte(b.String())
}

var markdownLink = regexp.MustCompile(`\]\(([^)\s]+)\)`)

// AbsoluteLinks points relative README links at GitHub, since PyPI cannot resolve repository paths.
func AbsoluteLinks(markdown string) string {
	return markdownLink.ReplaceAllStringFunc(markdown, func(m string) string {
		target := m[2 : len(m)-1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
			return m
		}
		kind := "blob"
		if strings.HasSuffix(target, "/") {
			kind = "tree"
		}
		return "](" + repository + "/" + kind + "/main/" + strings.TrimPrefix(target, "./") + ")"
	})
}

var semverPre = regexp.MustCompile(`^(\d+\.\d+\.\d+)-(alpha|a|beta|b|rc|dev)\.?(\d+)$`)

// Normalize maps a release tag's semver version to PEP 440, so one git tag versions both the
// Go binary and its wheels: 0.4.0-rc.1 -> 0.4.0rc1, 0.4.0-beta.2 -> 0.4.0b2, 0.4.0-dev.3 -> 0.4.0.dev3.
func Normalize(version string) (string, error) {
	version = strings.TrimPrefix(version, "v")
	if pep440.MatchString(version) {
		return version, nil
	}
	if m := semverPre.FindStringSubmatch(version); m != nil {
		n := strings.TrimLeft(m[3], "0")
		if n == "" {
			n = "0"
		}
		label := map[string]string{"alpha": "a", "a": "a", "beta": "b", "b": "b", "rc": "rc", "dev": ".dev"}[m[2]]
		if normalized := m[1] + label + n; pep440.MatchString(normalized) {
			return normalized, nil
		}
	}
	return "", fmt.Errorf("version %q is neither PEP 440 nor a supported semver prerelease (-alpha.N, -beta.N, -rc.N, -dev.N)", version)
}
