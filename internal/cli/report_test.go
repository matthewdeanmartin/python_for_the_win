package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
)

func TestReportErrorJSON(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code int
		kind string
	}{
		{errs.New(9, "Python 3.12 isn't installed"), 9, "not_installed"},
		{errs.Wrap(2, "invalid arguments", errors.New("bad")), 2, "usage"},
		{errors.New("boom"), 1, "unexpected"},
	} {
		b := new(bytes.Buffer)
		if e := ReportError(b, tc.err, true); e != nil {
			t.Fatal(e)
		}
		var got struct {
			Error struct {
				Code    int
				Kind    string
				Message string
			}
		}
		if e := json.Unmarshal(b.Bytes(), &got); e != nil {
			t.Fatal(b.String(), e)
		}
		if got.Error.Code != tc.code || got.Error.Kind != tc.kind || got.Error.Message != tc.err.Error() {
			t.Errorf("%v: %+v", tc.err, got)
		}
	}
}

func TestReportErrorText(t *testing.T) {
	b := new(bytes.Buffer)
	if e := ReportError(b, errs.New(2, "nope"), false); e != nil || strings.TrimSpace(b.String()) != "nope" {
		t.Fatal(b.String(), e)
	}
}

func TestJSONRequestedSeesRunnerFlags(t *testing.T) {
	c := New()
	c.SetOut(new(bytes.Buffer))
	c.SetErr(new(bytes.Buffer))
	c.SetArgs([]string{"run", "--json", "--root", t.TempDir(), "--python", "3.99", "-c", "1"})
	if e := c.Execute(); errs.CodeOf(e) != 9 || !JSONRequested(c) {
		t.Fatal(e, JSONRequested(c))
	}
	c = New()
	c.SetOut(new(bytes.Buffer))
	c.SetArgs([]string{"run", "--root", t.TempDir(), "--python", "3.99", "script.py", "--json"})
	if e := c.Execute(); e == nil || JSONRequested(c) {
		t.Fatal("script arguments must not enable pftw JSON output")
	}
}
