package cli

import (
	"bytes"
	"errors"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"testing"
)

func TestUsage(t *testing.T) {
	for _, args := range [][]string{{}, {"install", "--path", "invalid"}, {"install", "bad"}, {"uninstall"}, {"config", "set", "unknown", "x"}, {"--bogus"}, {"frobnicate"}} {
		c := New()
		c.SetOut(new(bytes.Buffer))
		c.SetErr(new(bytes.Buffer))
		c.SetArgs(args)
		e := c.Execute()
		var coded *errs.Error
		if !errors.As(e, &coded) || coded.Code != 2 {
			t.Fatal(args, e)
		}
	}
}
func TestEmptyList(t *testing.T) {
	c := New()
	b := new(bytes.Buffer)
	c.SetOut(b)
	c.SetArgs([]string{"list", "--json", "--root", t.TempDir()})
	if e := c.Execute(); e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(b.Bytes(), []byte(`"installs":{}`)) {
		t.Fatal(b.String())
	}
}
