package install

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type rmProcess struct {
	PID                   uint32
	Start                 windows.Filetime
	Name                  [256]uint16
	Service               [64]uint16
	Kind, Status, Session uint32
	Restartable           int32
}

// Restart Manager is used for diagnosis only. Never call RmShutdown or RmRestart.
func lockingProcesses(err error) []string {
	path := ""
	var link *os.LinkError
	var pe *os.PathError
	if errors.As(err, &link) {
		path = link.Old
	} else if errors.As(err, &pe) {
		path = pe.Path
	}
	if path == "" {
		return nil
	}
	files := []*uint16{}
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return filepath.SkipDir
		}
		if len(files) >= 2048 {
			return fs.SkipAll
		}
		info, e := d.Info()
		if e != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if stat, ok := info.Sys().(*windows.Win32FileAttributeData); ok && stat.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if p == path || ext == ".exe" || ext == ".dll" || ext == ".pyd" {
			if u, e := windows.UTF16PtrFromString(p); e == nil {
				files = append(files, u)
			}
		}
		return nil
	})
	if len(files) == 0 {
		return nil
	}
	dll := windows.NewLazySystemDLL("rstrtmgr.dll")
	start, end, register, get := dll.NewProc("RmStartSession"), dll.NewProc("RmEndSession"), dll.NewProc("RmRegisterResources"), dll.NewProc("RmGetList")
	if start.Find() != nil || end.Find() != nil || register.Find() != nil || get.Find() != nil {
		return nil
	}
	var session uint32
	var key [33]uint16
	code, _, _ := start.Call(uintptr(unsafe.Pointer(&session)), 0, uintptr(unsafe.Pointer(&key[0])))
	if code != 0 {
		return nil
	}
	defer func() { _, _, _ = end.Call(uintptr(session)) }() // Best-effort OS resource release.
	code, _, _ = register.Call(uintptr(session), uintptr(len(files)), uintptr(unsafe.Pointer(&files[0])), 0, 0, 0, 0)
	runtime.KeepAlive(files)
	if code != 0 {
		return nil
	}
	var needed, count, reasons uint32
	code, _, _ = get.Call(uintptr(session), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)), 0, uintptr(unsafe.Pointer(&reasons)))
	for attempts := 0; code == 234 && needed > 0 && needed < 4096 && attempts < 3; attempts++ {
		processes := make([]rmProcess, needed)
		count = needed
		code, _, _ = get.Call(uintptr(session), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&processes[0])), uintptr(unsafe.Pointer(&reasons)))
		if code == 0 {
			names := []string{}
			for _, p := range processes[:count] {
				names = append(names, fmt.Sprintf("%s (PID %d)", windows.UTF16ToString(p.Name[:]), p.PID))
			}
			return names
		}
	}
	return nil
}
