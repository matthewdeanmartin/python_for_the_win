package errs

import "fmt"

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
