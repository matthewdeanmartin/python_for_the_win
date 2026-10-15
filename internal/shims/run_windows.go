package shims

import (
	"fmt"
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

func launch(exe string) (int, error) {
	raw := windows.UTF16PtrToString(windows.GetCommandLine())
	return launchLine(exe, syscall.EscapeArg(exe)+Tail(raw))
}
func Launch(exe string, args []string) (int, error) {
	line := syscall.EscapeArg(exe)
	for _, a := range args {
		line += " " + syscall.EscapeArg(a)
	}
	return launchLine(exe, line)
}
func launchLine(exe, command string) (int, error) {
	job, e := windows.CreateJobObject(nil, nil)
	if e != nil {
		return 0, e
	}
	defer func(h windows.Handle) { _ = windows.CloseHandle(h) }(job) // Release owned OS handle.
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, e = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); e != nil {
		return 0, e
	}
	app, e := windows.UTF16PtrFromString(exe)
	if e != nil {
		return 0, e
	}
	line, e := windows.UTF16PtrFromString(command)
	if e != nil {
		return 0, e
	}
	si := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{})), Flags: windows.STARTF_USESTDHANDLES}
	handles := []*windows.Handle{&si.StdInput, &si.StdOutput, &si.StdErr}
	ids := []uint32{windows.STD_INPUT_HANDLE, windows.STD_OUTPUT_HANDLE, windows.STD_ERROR_HANDLE}
	for i, id := range ids {
		h, e := windows.GetStdHandle(id)
		if e != nil {
			return 0, e
		}
		if h == 0 || h == windows.InvalidHandle {
			continue
		}
		e = windows.DuplicateHandle(windows.CurrentProcess(), h, windows.CurrentProcess(), handles[i], 0, true, windows.DUPLICATE_SAME_ACCESS)
		if e != nil {
			return 0, e
		}
		defer func(h windows.Handle) { _ = windows.CloseHandle(h) }(*handles[i]) // Release owned OS handle.
	}
	var pi windows.ProcessInformation
	if e = windows.CreateProcess(app, line, nil, nil, true, windows.CREATE_SUSPENDED, nil, nil, &si, &pi); e != nil {
		return 0, e
	}
	defer func(h windows.Handle) { _ = windows.CloseHandle(h) }(pi.Process) // Release owned OS handle.
	defer func(h windows.Handle) { _ = windows.CloseHandle(h) }(pi.Thread)  // Release owned OS handle.
	if e = windows.AssignProcessToJobObject(job, pi.Process); e != nil {
		_ = windows.TerminateProcess(pi.Process, 1)
		return 0, e
	}
	// Install our handler after creating the child so Python retains normal console control handling.
	handler := syscall.NewCallback(func(event uint32) uintptr {
		if event == 0 || event == 1 {
			return 1
		}
		return 0
	})
	ctrl := windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleCtrlHandler")
	_, _, _ = ctrl.Call(handler, 1)                    // A windowed process may have no console handler to register.
	defer func() { _, _, _ = ctrl.Call(handler, 0) }() // Best-effort OS resource release.
	if _, e = windows.ResumeThread(pi.Thread); e != nil {
		return 0, e
	}
	if _, e = windows.WaitForSingleObject(pi.Process, windows.INFINITE); e != nil {
		return 0, e
	}
	var code uint32
	if e = windows.GetExitCodeProcess(pi.Process, &code); e != nil {
		return 0, fmt.Errorf("child exit status: %w", e)
	}
	return int(code), nil
}
