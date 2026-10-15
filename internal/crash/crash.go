package crash

import (
	"fmt"
	"os"
	"runtime/debug"
)

// Recover is deferred at process and worker boundaries so errors survive console closure.
func Recover(err *error) {
	if value := recover(); value != nil {
		f, e := os.CreateTemp("", "pftw-crash-*.log")
		if e != nil {
			*err = fmt.Errorf("something went wrong: %v (cannot create crash log: %v)", value, e)
			return
		}
		_, _ = fmt.Fprintf(f, "%v\n%s", value, debug.Stack())
		_ = f.Close()
		*err = fmt.Errorf("something went wrong - details in %s", f.Name())
	}
}
