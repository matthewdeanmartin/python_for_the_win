package cli

import "github.com/spf13/cobra"

// Explorer launches are a supported wizard entry, not a command-line misuse.
func init() { cobra.MousetrapHelpText = "" }
