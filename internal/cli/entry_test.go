package cli

import (
	"runtime"
	"testing"
)

func TestEntryPrecedence(t *testing.T) {
	for _, tc := range []struct {
		args []string
		gui  bool
	}{
		{[]string{"gui"}, true}, {[]string{"--gui"}, true},
		{[]string{"--gui", "--tui"}, false}, {[]string{"--gui", "--json"}, false},
		{[]string{"--gui", "--dry-run"}, false}, {[]string{"--tui"}, false},
		{[]string{"run", "-c", "print('--gui')"}, false}, {[]string{"version"}, false},
		{[]string{"install", "--gui"}, false}, {[]string{"--root", "gui", "list"}, false},
	} {
		want := tc.gui && (runtime.GOOS == "windows" || tc.args[0] == "gui")
		if got := Graphical(New(), tc.args); got != want {
			t.Fatal(tc.args, got, want)
		}
	}
}
