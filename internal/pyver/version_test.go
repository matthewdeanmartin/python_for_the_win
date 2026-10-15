package pyver

import "testing"

func TestOrdering(t *testing.T) {
	ss := []string{"3.15.0a1", "3.15.0b3", "3.15.0b4", "3.15.0rc1", "3.15.0rc3", "3.15.0", "3.15.1"}
	for i, s := range ss {
		v, e := Parse(s)
		if e != nil {
			t.Fatal(e)
		}
		if i > 0 {
			p, _ := Parse(ss[i-1])
			if p.Compare(v) >= 0 {
				t.Fatal(s)
			}
		}
	}
	for _, s := range []string{"3", "3.16", "3.16t", "3.16.0a1t"} {
		if _, e := Parse(s); e != nil {
			t.Fatal(e)
		}
	}
	for _, s := range []string{"", "4.1", "3.16-alpha", "3.16a1", "3.1.2.3"} {
		if _, e := Parse(s); e == nil {
			t.Fatal(s)
		}
	}
}
