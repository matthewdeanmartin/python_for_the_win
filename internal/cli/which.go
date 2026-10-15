package cli

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/runner"
	"github.com/spf13/cobra"
)

func (o *options) which(c *cobra.Command, args []string) error {
	s, err := o.load()
	if err != nil {
		return err
	}
	spec := ""
	if len(args) > 0 {
		spec = args[0]
	}
	// An omitted selector means the managed default, independent of project files.
	if spec == "" {
		spec = s.Default
	}
	selected, err := runner.SelectInstalled(o.root, s, spec)
	if err != nil {
		return err
	}
	return o.output(c, map[string]string{"exe": selected.Interpreter}, selected.Interpreter)
}
