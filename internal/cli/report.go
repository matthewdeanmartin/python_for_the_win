package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/spf13/cobra"
)

// JSONRequested reports whether the parsed command line asked for machine-readable output.
func JSONRequested(root *cobra.Command) bool {
	value, _ := root.PersistentFlags().GetBool("json")
	return value
}

// ReportError describes a failed command on w. JSON callers receive one object instead of prose,
// so scripts in any language can branch on "kind" without parsing messages.
func ReportError(w io.Writer, e error, asJSON bool) error {
	if !asJSON {
		_, err := fmt.Fprintln(w, e)
		return err
	}
	code := errs.CodeOf(e)
	return json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": code, "kind": errs.Kind(code), "message": e.Error()}})
}
