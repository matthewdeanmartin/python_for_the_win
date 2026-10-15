package runner

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectionAndPrecedence(t *testing.T) {
	root := t.TempDir()
	s := state.New()
	suffix := resolve.Suffix(platform.NativeArch())
	for _, minor := range []string{"3.14", "3.15", "3.16"} {
		key := minor + "-" + suffix
		dir := filepath.Join("versions", key)
		_ = os.MkdirAll(filepath.Join(root, dir), 0700)
		_ = os.WriteFile(filepath.Join(root, dir, "python.exe"), nil, 0700)
		version := minor + ".0"
		if minor == "3.16" {
			version += "a1"
		}
		_ = s.Set(key, state.Install{Dir: dir, Version: version, Arch: platform.NativeArch()})
	}
	s.Default = "3.14-" + suffix
	for _, tc := range []struct {
		name, script, explicit, reason, minor string
		bad                                   bool
	}{
		{"default", "print(1)", "", "default", "3.14", false},
		{"shebang", "#!/usr/bin/env -S python3.15 -X dev -u\nprint(1)", "", "shebang", "3.15", false},
		{"flag wins", "#!/bin/bash", "3.14", "flag", "3.14", false},
		{"metadata", "# /// script\n# requires-python = '>=3.14'\n# dependencies = ['x']\n# ///", "", "723", "3.15", false},
		{"pre fallback", "# /// script\n# requires-python = '>=3.16a1'\n# ///", "", "723", "3.16", false},
		{"nonpython", "#!/bin/bash", "", "", "", true},
		{"unclosed", "# /// script\n# requires-python = '>=3.14'", "", "", "", true},
		{"invalid toml", "# /// script\n# requires-python = 42\n# ///", "", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := filepath.Join(root, "test.py")
			_ = os.WriteFile(script, []byte(tc.script), 0600)
			got, e := Select(root, s, tc.explicit, []string{script, "--dry-run", "snow 雪", "a b"})
			if tc.bad {
				if e == nil {
					t.Fatal("accepted malformed script")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if got.Reason != tc.reason || !strings.Contains(got.Interpreter, tc.minor) {
				t.Fatal(got)
			}
			if got.Args[len(got.Args)-3] != "--dry-run" {
				t.Fatal("lost script argument", got)
			}
		})
	}
}
func TestConstraints(t *testing.T) {
	for _, tc := range []struct {
		rule, version string
		want          bool
	}{
		{">=3.12,<4", "3.15.0", true}, {">=3.12,!=3.15.*", "3.15.1", false}, {"~=3.14.2", "3.15.0", false}, {"~=3.14", "3.15.0", true}, {"<3.16", "3.16.0a1", false}, {"==3.15.*", "3.15.0a1", true},
	} {
		got, e := MatchesRequirement(tc.rule, tc.version)
		if e != nil || got != tc.want {
			t.Fatal(tc, got, e)
		}
	}
	for _, rule := range []string{"wat", ">=3.14.*", "~=3", "==3.14,,<4"} {
		if _, e := MatchesRequirement(rule, "3.14.0"); e == nil {
			t.Fatal(rule)
		}
	}
}
