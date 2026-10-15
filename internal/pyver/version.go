package pyver

import (
	"fmt"
	"regexp"
	"strconv"
)

var pattern = regexp.MustCompile(`^(3)(?:\.(\d+))?(?:\.(\d+))?(?:(a|b|rc)(\d+))?(t)?$`)

type Version struct {
	Major, Minor, Patch, Stage, Serial, Parts int
	Threaded                                  bool
}

func Parse(s string) (Version, error) {
	m := pattern.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("invalid Python version %q", s)
	}
	var err error
	v := Version{Major: 3, Stage: 3, Parts: 1, Threaded: m[6] != ""}
	if m[2] != "" {
		v.Minor, err = strconv.Atoi(m[2])
		if err != nil {
			return Version{}, fmt.Errorf("invalid Python version %q: %w", s, err)
		}
		v.Parts = 2
	}
	if m[3] != "" {
		v.Patch, err = strconv.Atoi(m[3])
		if err != nil {
			return Version{}, fmt.Errorf("invalid Python version %q: %w", s, err)
		}
		v.Parts = 3
	}
	if m[4] != "" {
		if v.Parts != 3 {
			return v, fmt.Errorf("prerelease needs a full version")
		}
		v.Stage = map[string]int{"a": 0, "b": 1, "rc": 2}[m[4]]
		v.Serial, err = strconv.Atoi(m[5])
		if err != nil {
			return Version{}, fmt.Errorf("invalid Python version %q: %w", s, err)
		}
	}
	return v, nil
}
func (v Version) Pre() bool { return v.Stage < 3 }
func (v Version) Compare(w Version) int {
	a := []int{v.Major, v.Minor, v.Patch, v.Stage, v.Serial}
	b := []int{w.Major, w.Minor, w.Patch, w.Stage, w.Serial}
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
