package errs

import (
	"errors"
	"fmt"
)

type Error struct {
	Code    int
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}
func (e *Error) Unwrap() error { return e.Cause }
func New(code int, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}
func Wrap(code int, message string, err error) error {
	return &Error{Code: code, Message: message, Cause: err}
}

// Presented marks an error whose wizard screen has already been acknowledged.
type Presented struct{ Cause error }

func (e *Presented) Error() string { return e.Cause.Error() }
func (e *Presented) Unwrap() error { return e.Cause }

// CodeOf returns the process exit code for err: its Code, or 1 when uncoded.
func CodeOf(err error) int {
	var coded *Error
	if errors.As(err, &coded) {
		return coded.Code
	}
	return 1
}

// Kind names an exit code (spec 7.1) for machine-readable error reports.
func Kind(code int) string {
	kinds := map[int]string{2: "usage", 3: "preflight", 4: "decision_required", 5: "network", 6: "integrity", 7: "files_in_use", 8: "unsupported_platform", 9: "not_installed"}
	if kind, ok := kinds[code]; ok {
		return kind
	}
	return "unexpected"
}
