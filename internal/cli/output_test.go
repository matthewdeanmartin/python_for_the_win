package cli

import (
	"errors"
	"strings"
	"testing"
)

var errOutput = errors.New("injected output failure")

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errOutput }
func TestOutputErrorsPropagate(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"version", "--json"}, {"list"}, {"doctor"}, {"docs"}} {
		c := New()
		c.SetOut(failedWriter{})
		c.SetArgs(append(args, "--root", t.TempDir()))
		if e := c.Execute(); !errors.Is(e, errOutput) {
			t.Errorf("%v: %v", args, e)
		}
	}
}

func TestInstallConsentRequiresCompleteReply(t *testing.T) {
	for _, tc := range []struct {
		reply    string
		yes, bad bool
	}{
		{"yes", false, true}, {"", false, true}, {"yes" + string(rune(10)), true, false}, {"Y" + string([]byte{13, 10}), true, false}, {"no" + string(rune(10)), false, false}, {"maybe" + string(rune(10)), false, false},
	} {
		yes, e := readInstallConsent(strings.NewReader(tc.reply))
		if yes != tc.yes || (e != nil) != tc.bad {
			t.Errorf("%q: %v %v", tc.reply, yes, e)
		}
	}
}
