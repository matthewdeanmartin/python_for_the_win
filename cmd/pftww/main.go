// pftww is the GUI-subsystem companion used only by versioned pythonw aliases.
package main

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/shims"
	"os"
)

func main() {
	self, e := os.Executable()
	if e != nil {
		os.Exit(1)
	}
	os.Exit(shims.Run(self))
}
