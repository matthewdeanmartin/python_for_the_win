package platform

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

func SetTitle() {
	p, _ := windows.UTF16PtrFromString("Python for the Win")
	_, _, _ = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleTitleW").Call(uintptr(unsafe.Pointer(p)))
}
