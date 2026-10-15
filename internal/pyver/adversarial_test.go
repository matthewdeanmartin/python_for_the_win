package pyver

import (
	"strings"
	"testing"
)

func TestRejectNumericOverflow(t *testing.T) {
	n := strings.Repeat("9", 100)
	for _, s := range []string{"3." + n, "3.15." + n, "3.15.0rc" + n} {
		if _, e := Parse(s); e == nil {
			t.Errorf("accepted overflow: %s", s)
		}
	}
}
func FuzzParse(f *testing.F) {
	for _, s := range []string{"3.15.0", "3.16.0a1", "3.14t", "", "3." + strings.Repeat("9", 30)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		v, e := Parse(s)
		if e == nil && (v.Minor < 0 || v.Patch < 0 || v.Serial < 0 || v.Compare(v) != 0) {
			t.Fatal(v)
		}
	})
}
