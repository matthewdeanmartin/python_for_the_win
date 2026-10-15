package platform

import (
	"golang.org/x/sys/windows"
	"os"
	"regexp"
	"unsafe"
)

func NativeArch() string {
	k := windows.NewLazySystemDLL("kernel32.dll")
	p := k.NewProc("IsWow64Process2")
	var process, native uint16
	if p.Find() == nil {
		r, _, _ := p.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&process)), uintptr(unsafe.Pointer(&native)))
		if r != 0 {
			switch native {
			case 0x8664:
				return "amd64"
			case 0xaa64:
				return "arm64"
			case 0x14c:
				return "x86"
			}
		}
	}
	var info struct {
		Arch, Reserved           uint16
		Page                     uint32
		Min, Max, Mask           uintptr
		Count, Type, Granularity uint32
		Level, Revision          uint16
	}
	k.NewProc("GetNativeSystemInfo").Call(uintptr(unsafe.Pointer(&info)))
	switch info.Arch {
	case 9:
		return "amd64"
	case 12:
		return "arm64"
	case 0:
		return "x86"
	}
	return "unknown"
}

var msysPTY = regexp.MustCompile(`^(msys|cygwin)-[0-9a-f]+-pty[0-9]+-(from|to)-master$`)

func isMsysPTY(name string) bool {
	return len(name) > 1 && name[0] == 92 && msysPTY.MatchString(name[1:])
}

func IsTerminal(f *os.File) bool {
	h := windows.Handle(f.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) == nil {
		return true
	}
	kind, e := windows.GetFileType(h)
	if e != nil || kind != windows.FILE_TYPE_PIPE {
		return false
	}
	// Git Bash/mintty exposes its terminal as a named pipe rather than a console handle.
	var info struct {
		Length uint32
		Name   [1024]uint16
	}
	if e = windows.GetFileInformationByHandleEx(h, windows.FileNameInfo, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); e != nil {
		return false
	}
	if info.Length == 0 || info.Length > uint32(len(info.Name)*2) {
		return false
	}
	return isMsysPTY(windows.UTF16ToString(info.Name[:info.Length/2]))
}
