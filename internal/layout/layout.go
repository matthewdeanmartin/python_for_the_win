package layout

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var keyPattern = regexp.MustCompile(`^3\.\d+t?-(64|32|arm64)$`)

func ValidKey(s string) bool { return keyPattern.MatchString(s) }
func Root(s string) (string, error) {
	if s == "" {
		base := os.Getenv("LOCALAPPDATA")
		if runtime.GOOS != "windows" {
			var err error
			base, err = os.UserCacheDir()
			if err != nil {
				return "", err
			}
		}
		if base == "" {
			return "", fmt.Errorf("cannot find user application directory; pass --root")
		}
		s = filepath.Join(base, "pftw")
	}
	return filepath.Abs(s)
}

// Safe rejects symlinks/junctions in existing ancestors before managed filesystem operations.
func Safe(root string, parts ...string) (string, error) {
	target := filepath.Join(append([]string{root}, parts...)...)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes installation root")
	}
	for p := target; ; p = filepath.Dir(p) {
		i, e := os.Lstat(p)
		if e == nil && i.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return "", fmt.Errorf("refusing linked/reparse path %s", p)
		}
		if e != nil && !os.IsNotExist(e) {
			return "", e
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return target, nil
}
func Remove(root string, parts ...string) error {
	p, e := Safe(root, parts...)
	if e != nil {
		return e
	}
	if p == root {
		return fmt.Errorf("refusing recursive root removal")
	}
	return os.RemoveAll(p)
}
