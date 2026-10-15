// Package runner selects an interpreter without installing packages or changing PATH.
package runner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/pyver"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"github.com/pelletier/go-toml/v2"
)

type Selection struct {
	Interpreter, Reason, Spec string
	Args                      []string
	Dependencies              bool
}

var pythonName = regexp.MustCompile(`^python(3(?:\.\d+)?t?)?$`)

func Select(root string, s *state.State, explicit string, args []string) (Selection, error) {
	return selectInterpreter(root, s, explicit, args, true)
}

// SelectInstalled ignores script and project hints for machine-readable lookup.
func SelectInstalled(root string, s *state.State, spec string) (Selection, error) {
	return selectInterpreter(root, s, spec, nil, false)
}

func selectInterpreter(root string, s *state.State, explicit string, args []string, project bool) (Selection, error) {
	result := Selection{Reason: "default", Args: append([]string{}, args...)}
	spec := explicit
	requirement := ""
	if explicit != "" {
		result.Reason = "flag"
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		f, e := os.Open(args[0])
		if e != nil {
			return result, e
		}
		defer func() { _ = f.Close() }() // Read-only handle; read errors are checked above.
		reader := bufio.NewReaderSize(f, 1025)
		lineBytes, e := reader.ReadSlice('\n')
		if e != nil && e != io.EOF && e != bufio.ErrBufferFull {
			return result, e
		}
		line := strings.TrimPrefix(string(lineBytes), "\ufeff")
		if strings.HasPrefix(line, "#!") {
			if len(line) > 1024 {
				return result, errs.New(2, "script shebang exceeds 1 KiB")
			}
			fields, e := words(strings.TrimSpace(line[2:]))
			if e != nil {
				return result, e
			}
			if len(fields) > 0 && fields[0] == "/usr/bin/env" {
				fields = fields[1:]
				if len(fields) > 0 && fields[0] == "-S" {
					fields = fields[1:]
				}
			}
			if len(fields) == 0 {
				return result, errs.New(2, "not a Python script shebang")
			}
			name := fields[0]
			if strings.HasPrefix(name, "/usr/local/bin/") {
				name = strings.TrimPrefix(name, "/usr/local/bin/")
			} else {
				name = strings.TrimPrefix(name, "/usr/bin/")
			}
			match := pythonName.FindStringSubmatch(name)
			if match == nil {
				return result, errs.New(2, "not a Python script shebang")
			}
			spec = match[1]
			if spec == "3" {
				spec = ""
			}
			result.Reason = "shebang"
			result.Args = append(fields[1:], args...)
		} else {
			if _, e = f.Seek(0, 0); e != nil {
				return result, e
			}
			requirement, result.Dependencies, e = metadata(f)
			if e != nil {
				return result, e
			}
			if requirement != "" {
				result.Reason = "723"
			}
		}
	}
	if project && explicit == "" && spec == "" && requirement == "" {
		cwd, e := os.Getwd()
		if e != nil {
			return result, e
		}
		spec, e = projectVersion(cwd)
		if e != nil {
			return result, e
		}
		if spec != "" {
			result.Reason = ".python-version"
		}
	}
	if project && explicit != "" && (strings.ContainsAny(explicit, `/\`) || strings.HasSuffix(strings.ToLower(explicit), ".exe")) {
		p, e := filepath.Abs(explicit)
		if e != nil {
			return result, e
		}
		i, e := os.Stat(p)
		if e != nil {
			return result, e
		}
		if !i.Mode().IsRegular() {
			return result, errs.New(2, "interpreter is not a regular file")
		}
		result.Interpreter = p
		return result, nil
	}
	key := s.Default
	if requirement != "" {
		check, e := constraint(requirement)
		if e != nil {
			return result, e
		}
		keys := []string{}
		for k := range s.Installs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		key = ""
		var best pyver.Version
		for _, k := range keys {
			v, e := s.Get(k)
			if e != nil {
				return result, e
			}
			if v.Arch != platform.NativeArch() || v.Threaded {
				continue
			}
			pv, e := pyver.Parse(v.Version)
			if e != nil {
				return result, e
			}
			if !check(pv) {
				continue
			}
			if key == "" || (best.Pre() && !pv.Pre()) || best.Pre() == pv.Pre() && pv.Compare(best) > 0 {
				key = k
				best = pv
			}
		}
		result.Spec = requirement
	} else if spec != "" {
		result.Spec = spec
		normalized, w, e := resolve.Normalize(resolve.Options{Spec: spec, Arch: platform.NativeArch()})
		if e != nil {
			return result, errs.Wrap(2, "invalid Python selector", e)
		}
		key = ""
		var best pyver.Version
		keys := []string{}
		for k := range s.Installs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v, e := s.Get(k)
			if e != nil {
				return result, e
			}
			pv, e := pyver.Parse(v.Version)
			if e != nil {
				return result, e
			}
			if !strings.HasSuffix(normalized, "-"+resolve.Suffix(v.Arch)) || v.Threaded != w.Threaded {
				continue
			}
			match := k == normalized || w.Parts == 3 && pv.Compare(w) == 0 || w.Parts == 1 && !pv.Pre()
			if match && (key == "" || pv.Compare(best) > 0) {
				key = k
				best = pv
			}
		}

	}
	if key == "" {
		if result.Spec == "" {
			result.Spec = "latest"
		}
		return result, errs.New(9, "Python %s isn't installed: use pftw install with the desired version", result.Spec)
	}
	v, e := s.Get(key)
	if e != nil {
		return result, e
	}
	result.Interpreter, e = layout.Safe(root, filepath.FromSlash(v.Dir), "python.exe")
	if e != nil {
		return result, e
	}
	info, statErr := os.Stat(result.Interpreter)
	if statErr != nil || !info.Mode().IsRegular() {
		return result, errs.New(9, "Python %s is registered but not a regular interpreter file at %s", key, result.Interpreter)
	}
	return result, nil
}

func words(s string) ([]string, error) {
	out := []string{}
	var b strings.Builder
	quote := rune(0)
	escaped, started := false, false
	for _, r := range s {
		if escaped {
			b.WriteRune(r)
			escaped = false
			started = true
			continue
		}
		if r == rune(92) && quote != rune(39) {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
			continue
		}
		if r == ' ' || r == '\t' {
			if started {
				out = append(out, b.String())
				b.Reset()
				started = false
			}
			continue
		}
		b.WriteRune(r)
		started = true
	}
	if quote != 0 || escaped {
		return nil, errs.New(2, "malformed shebang quoting")
	}
	if started {
		out = append(out, b.String())
	}
	return out, nil
}

func metadata(r io.Reader) (string, bool, error) {
	scanner := bufio.NewScanner(io.LimitReader(r, 4<<20))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	active, found, closed := false, false, false
	var b strings.Builder
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "# /// script" {
			if found {
				return "", false, errs.New(2, "duplicate script metadata")
			}
			found = true
			active = true
			continue
		}
		if !active {
			continue
		}
		if line == "# ///" {
			active = false
			closed = true
			continue
		}
		if line != "#" && !strings.HasPrefix(line, "# ") {
			return "", false, errs.New(2, "malformed script metadata comment")
		}
		b.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "#"), " "))
		b.WriteByte('\n')
	}
	if e := scanner.Err(); e != nil {
		return "", false, e
	}
	if found && !closed {
		return "", false, errs.New(2, "unclosed script metadata")
	}
	var meta struct {
		Requires     string   `toml:"requires-python"`
		Dependencies []string `toml:"dependencies"`
	}
	if e := toml.Unmarshal([]byte(b.String()), &meta); e != nil {
		return "", false, errs.Wrap(2, "invalid script metadata", e)
	}
	return meta.Requires, len(meta.Dependencies) > 0, nil
}

var clause = regexp.MustCompile(`^(~=|===|==|!=|<=|>=|<|>)\s*(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:(a|b|rc)(\d+))?(\.\*)?$`)

func constraint(s string) (func(pyver.Version) bool, error) {
	checks := []func(pyver.Version) bool{}
	for _, part := range strings.Split(s, ",") {
		m := clause.FindStringSubmatch(strings.TrimSpace(part))
		if m == nil {
			return nil, errs.New(2, "invalid requires-python constraint %q", part)
		}
		v := pyver.Version{Stage: 3, Parts: 1}
		_, _ = fmt.Sscan(m[2], &v.Major)
		if m[3] != "" {
			_, _ = fmt.Sscan(m[3], &v.Minor)
			v.Parts = 2
		}
		if m[4] != "" {
			_, _ = fmt.Sscan(m[4], &v.Patch)
			v.Parts = 3
		}
		if m[5] != "" {
			v.Stage = map[string]int{"a": 0, "b": 1, "rc": 2}[m[5]]
			_, _ = fmt.Sscan(m[6], &v.Serial)
		}
		op := m[1]
		wildcard := m[7] != ""
		if wildcard && (op != "==" && op != "!=" || v.Pre()) || op == "~=" && v.Parts < 2 {
			return nil, errs.New(2, "invalid requires-python constraint %q", part)
		}
		checks = append(checks, func(p pyver.Version) bool {
			n := p.Compare(v)
			equal := n == 0
			if wildcard {
				equal = p.Major == v.Major && (v.Parts < 2 || p.Minor == v.Minor) && (v.Parts < 3 || p.Patch == v.Patch)
			}
			switch op {
			case "==", "===":
				return equal
			case "!=":
				return !equal
			case ">=":
				return n >= 0
			case "<=":
				return n <= 0
			case ">":
				return n > 0 && !(p.Pre() && !v.Pre() && p.Major == v.Major && p.Minor == v.Minor && p.Patch == v.Patch)
			case "<":
				return n < 0 && !(p.Pre() && !v.Pre() && p.Major == v.Major && p.Minor == v.Minor && p.Patch == v.Patch)
			case "~=":
				return n >= 0 && p.Major == v.Major && (v.Parts < 3 || p.Minor == v.Minor)
			}
			return false
		})
	}
	return func(p pyver.Version) bool {
		for _, check := range checks {
			if !check(p) {
				return false
			}
		}
		return true
	}, nil
}

// MatchesRequirement validates the complete expression before evaluating a version.
func MatchesRequirement(requirement, version string) (bool, error) {
	check, e := constraint(requirement)
	if e != nil {
		return false, e
	}
	v, e := pyver.Parse(version)
	if e != nil {
		return false, e
	}
	return check(v), nil
}
