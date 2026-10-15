package platform

import (
	"encoding/binary"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

func (Native) IsElevated() (bool, error) {
	var elevated uint32
	var n uint32
	e := windows.GetTokenInformation(windows.GetCurrentProcessToken(), windows.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), 4, &n)
	return elevated != 0, e
}
func (Native) FreeBytes(path string) (uint64, error) {
	for {
		if _, e := os.Stat(path); e == nil {
			break
		}
		p := filepath.Dir(path)
		if p == path {
			break
		}
		path = p
	}
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return 0, e
	}
	var free uint64
	e = windows.GetDiskFreeSpaceEx(p, &free, nil, nil)
	return free, e
}
func (Native) OwnsConsole() (bool, error) {
	var ids [16]uint32
	r, _, e := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleProcessList").Call(uintptr(unsafe.Pointer(&ids[0])), 16)
	if r == 0 {
		return false, e
	}
	return r == 1, nil
}
func (Native) IsNetworkPath(path string) (bool, error) {
	if strings.HasPrefix(path, `\`) {
		return true, nil
	}
	p, e := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	if e != nil {
		return false, e
	}
	return windows.GetDriveType(p) == windows.DRIVE_REMOTE, nil
}
func (Native) IsAppExecLink(path string) (bool, error) {
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return false, e
	}
	h, e := windows.CreateFile(p, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if e != nil {
		return false, e
	}
	defer func(h windows.Handle) { _ = windows.CloseHandle(h) }(h) // Release owned OS handle.
	var b [16384]byte
	var n uint32
	e = windows.DeviceIoControl(h, windows.FSCTL_GET_REPARSE_POINT, nil, 0, &b[0], uint32(len(b)), &n, nil)
	if e == windows.ERROR_NOT_A_REPARSE_POINT {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	return n >= 4 && binary.LittleEndian.Uint32(b[:4]) == 0x8000001b, nil
}
