package runner

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// projectVersion uses the closest file in the working directory or its ancestors.
func projectVersion(dir string) (string, error) {
	for {
		name := filepath.Join(dir, ".python-version")
		f, err := os.Open(name)
		if err == nil {
			b, e := io.ReadAll(io.LimitReader(f, 4097))
			_ = f.Close() // Read-only or already-failed operation; preserve the primary error.
			if e != nil {
				return "", e
			}
			if len(b) > 4096 {
				return "", errs.New(2, "%s exceeds 4 KiB", name)
			}
			values := []string{}
			for _, line := range strings.Split(strings.TrimPrefix(string(b), "\ufeff"), "\n") {
				line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
				if line != "" {
					values = append(values, line)
				}
			}
			if len(values) != 1 {
				return "", errs.New(2, "%s must contain one Python version", name)
			}
			v := values[0]
			if strings.ContainsAny(v, "/"+string(rune(92))) {
				return "", errs.New(2, "%s must contain a version, not an executable path", name)
			}
			if _, _, e := resolve.Normalize(resolve.Options{Spec: v, Arch: platform.NativeArch()}); e != nil {
				return "", errs.Wrap(2, "invalid .python-version", e)
			}
			return v, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}
