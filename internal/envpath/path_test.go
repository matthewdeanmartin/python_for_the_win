package envpath

import (
	"strings"
	"testing"
)

func TestPreservePath(t *testing.T) {
	for _, value := range []string{`C:\Other;%USERPROFILE%\bin;`, "", `;C:\Other;;`, strings.Repeat("x", 4000), strings.Repeat(`C:\Tools\工具;`, 2000) + `%USERPROFILE%\bin;;TAIL;`} {
		entries := []string{`C:\pftw\versions\3.15-64`, `C:\pftw\shims`}
		added := AddFront(value, entries)
		if got := Remove(added, entries); got != value {
			t.Fatalf("round trip %q -> %q", value, got)
		}
		replaced := Replace(added, entries, []string{`C:\pftw\versions\3.16-64`})
		if got := Remove(replaced, []string{`C:\pftw\versions\3.16-64`}); got != value {
			t.Fatal(got, value)
		}
	}
}
func TestReplacePosition(t *testing.T) {
	got := Replace(`A;C:\pftw\old;B;;`, []string{`c:\PFTW\old\`}, []string{`C:\pftw\new`})
	if got != `A;C:\pftw\new;B;;` {
		t.Fatal(got)
	}
	if got := AddFront(`A;B;`, []string{"a", "C"}); got != "C;A;B;" {
		t.Fatal(got)
	}
}
