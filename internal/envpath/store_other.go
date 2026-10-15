//go:build !windows

package envpath

import "fmt"

type unsupported struct{}

func UserStore() Store                  { return unsupported{} }
func (unsupported) Get() (Value, error) { return Value{}, fmt.Errorf("PATH storage is Windows-only") }
func (unsupported) Set(Value) error     { return fmt.Errorf("PATH storage is Windows-only") }
func SystemPath() (Value, error)        { return Value{}, fmt.Errorf("Windows-only") }
func Broadcast()                        {}
func Expand(s string) string            { return s }
