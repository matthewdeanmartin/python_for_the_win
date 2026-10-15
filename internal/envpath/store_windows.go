package envpath

import (
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"syscall"
	"unsafe"
)

type RegistryStore struct {
	Key             registry.Key
	Path, ValueName string
}

func UserStore() Store { return RegistryStore{registry.CURRENT_USER, `Environment`, "Path"} }
func (s RegistryStore) Get() (Value, error) {
	// Registry is the only source for unexpanded user/system PATH and its original value type.
	k, e := registry.OpenKey(s.Key, s.Path, registry.QUERY_VALUE)
	if e == syscall.ERROR_FILE_NOT_FOUND {
		return Value{Kind: registry.EXPAND_SZ}, nil
	}
	if e != nil {
		return Value{}, e
	}
	defer k.Close()
	text, kind, e := k.GetStringValue(s.ValueName)
	if e == syscall.ERROR_FILE_NOT_FOUND {
		return Value{Kind: registry.EXPAND_SZ}, nil
	}
	if e != nil {
		return Value{}, e
	}
	return Value{text, kind, true}, nil
}
func (s RegistryStore) Set(v Value) error {
	// Persistent per-user PATH has no equivalent file/API storage; never write system PATH.
	if s.Key != registry.CURRENT_USER {
		return fmt.Errorf("only per-user PATH writes are permitted")
	}
	k, e := registry.OpenKey(s.Key, s.Path, registry.SET_VALUE)
	if e != nil {
		return e
	}
	defer k.Close()
	if !v.Exists {
		e = k.DeleteValue(s.ValueName)
		if e == syscall.ERROR_FILE_NOT_FOUND {
			return nil
		}
		return e
	}
	if v.Kind == registry.SZ {
		return k.SetStringValue(s.ValueName, v.Text)
	}
	if v.Kind == registry.EXPAND_SZ {
		return k.SetExpandStringValue(s.ValueName, v.Text)
	}
	return fmt.Errorf("unsupported PATH value type")
}
func SystemPath() (Value, error) {
	return (RegistryStore{registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, "Path"}).Get()
}
func Broadcast() {
	p, _ := windows.UTF16PtrFromString("Environment")
	var result uintptr
	windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW").Call(0xffff, 0x1a, 0, uintptr(unsafe.Pointer(p)), 2, 5000, uintptr(unsafe.Pointer(&result)))
}
func Expand(s string) string {
	p, e := windows.UTF16PtrFromString(s)
	if e != nil {
		return s
	}
	n, e := windows.ExpandEnvironmentStrings(p, nil, 0)
	if e != nil || n == 0 {
		return s
	}
	b := make([]uint16, n)
	if _, e = windows.ExpandEnvironmentStrings(p, &b[0], n); e != nil {
		return s
	}
	return windows.UTF16ToString(b)
}

func (s RegistryStore) Notify() {
	if s.Key == registry.CURRENT_USER && s.Path == `Environment` && s.ValueName == "Path" {
		Broadcast()
	}
}
