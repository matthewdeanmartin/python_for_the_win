package platform

import "github.com/matthewdeanmartin/python_for_the_win/internal/envpath"

type Info interface {
	IsElevated() (bool, error)
	FreeBytes(string) (uint64, error)
	OwnsConsole() (bool, error)
	IsNetworkPath(string) (bool, error)
	SystemPath() (envpath.Value, error)
	UserPath() (envpath.Value, error)
	IsAppExecLink(string) (bool, error)
}
type Native struct{}

func (Native) SystemPath() (envpath.Value, error) { return envpath.SystemPath() }
func (Native) UserPath() (envpath.Value, error)   { return envpath.UserStore().Get() }
