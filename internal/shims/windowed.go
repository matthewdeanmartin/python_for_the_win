package shims

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/overlay"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
)

func Windowed(name string) bool { return strings.HasPrefix(strings.ToLower(name), "pythonw") }

// Optional companion: go install builds can omit it without breaking console aliases.
func syncWindowed(root, self string) (string, error) {
	source := filepath.Join(filepath.Dir(self), "pftww.exe")
	dest, e := layout.Safe(root, "shims", "pftww.exe")
	if e != nil {
		return "", e
	}
	marker, e := layout.Safe(root, "shims", "windowed-owned.json")
	if e != nil {
		return "", e
	}
	if info, err := overlay.Inspect(self); err != nil {
		return "", err
	} else if info != nil {
		source = overlay.Source(self, "pftww.exe")
	}
	r, e := openSource(source)
	if os.IsNotExist(e) {
		return "", nil
	}
	if e != nil {
		return "", e
	}
	r.Close()

	if strings.EqualFold(filepath.Clean(source), filepath.Clean(dest)) {
		if _, e = os.Stat(marker); e != nil {
			return "", fmt.Errorf("windowed companion has no ownership record")
		}
		return dest, nil
	}
	sourceHash, e := digest(source)
	if e != nil {
		return "", e
	}
	oldHash, de := digest(dest)
	if de == nil {
		if _, e = os.Stat(marker); e != nil {
			return "", fmt.Errorf("unowned windowed companion exists: %s", dest)
		}
		if oldHash == sourceHash {
			return dest, nil
		}
	} else if !os.IsNotExist(de) {
		return "", de
	}
	for _, n := range []string{"pftww.exe.new", "pftww.exe.old"} {
		if _, e = layout.Safe(root, "shims", n); e != nil {
			return "", e
		}
	}
	if e = copyFile(source, dest+".new"); e != nil {
		return "", e
	}
	defer os.Remove(dest + ".new")
	if e = state.Write(filepath.Dir(dest), "windowed-owned.json", map[string]bool{"owned": true}); e != nil {
		return "", e
	}
	if de == nil {
		if e = os.Remove(dest + ".old"); e != nil && !os.IsNotExist(e) {
			return "", e
		}
		if e = os.Rename(dest, dest+".old"); e != nil {
			return "", e
		}
	}
	if e = os.Rename(dest+".new", dest); e != nil {
		if de == nil {
			_ = os.Rename(dest+".old", dest)
		}
		return "", e
	}
	_ = os.Remove(dest + ".old")
	return dest, nil
}
