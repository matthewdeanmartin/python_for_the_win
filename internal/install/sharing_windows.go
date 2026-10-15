package install

import (
	"errors"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"golang.org/x/sys/windows"
	"strings"
)

func sharingError(e error) error {
	var coded *errs.Error
	if errors.As(e, &coded) {
		return e
	}
	if errors.Is(e, windows.ERROR_SHARING_VIOLATION) || errors.Is(e, windows.ERROR_ACCESS_DENIED) {
		message := "Python files are in use or access is denied; close Python processes and retry"
		if names := lockingProcesses(e); len(names) > 0 {
			message += ". Using these files: " + strings.Join(names, ", ")
		}
		return errs.Wrap(7, message, e)
	}
	return e
}
