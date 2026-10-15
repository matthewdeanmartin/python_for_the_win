package errs

type ChildExit struct{ Code int }

func (e *ChildExit) Error() string { return "Python exited unsuccessfully" }
