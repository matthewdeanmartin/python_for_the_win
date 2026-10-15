package platform

import (
	"golang.org/x/sys/windows"
	"os"
	"strings"
	"unsafe"
)

// DesktopLaunch requires both an exclusively owned console and Explorer as the
// parent. Unknown launchers deliberately fall back to terminal behavior.
func DesktopLaunch() bool {
	own, err := (Native{}).OwnsConsole()
	if err != nil || !own {
		return false
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer func(h windows.Handle) { _ = windows.CloseHandle(h) }(snapshot) // Release owned OS handle.
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	parents := map[uint32]uint32{}
	names := map[uint32]string{}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		parents[entry.ProcessID] = entry.ParentProcessID
		names[entry.ProcessID] = windows.UTF16ToString(entry.ExeFile[:])
	}
	return strings.EqualFold(names[parents[uint32(os.Getpid())]], "explorer.exe")
}
