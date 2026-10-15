package envpath

import "strings"

type Value struct {
	Text   string `json:"text"`
	Kind   uint32 `json:"kind"`
	Exists bool   `json:"exists"`
}
type Store interface {
	Get() (Value, error)
	Set(Value) error
}
type MemStore struct {
	Value Value
	Err   error
}

func (m *MemStore) Get() (Value, error) { return m.Value, m.Err }
func (m *MemStore) Set(v Value) error {
	if m.Err != nil {
		return m.Err
	}
	m.Value = v
	return nil
}
func Split(s string) []string { return strings.Split(s, ";") }
func Join(s []string) string  { return strings.Join(s, ";") }
func canonical(s string) string {
	return strings.ToLower(strings.TrimRight(strings.ReplaceAll(s, "/", `\`), `\`))
}
func Equal(a, b string) bool { return canonical(a) == canonical(b) }
func Contains(list []string, entry string) bool {
	for _, s := range list {
		if Equal(s, entry) {
			return true
		}
	}
	return false
}
func Remove(value string, entries []string) string {
	out := []string{}
	for _, s := range Split(value) {
		if !Contains(entries, s) {
			out = append(out, s)
		}
	}
	return Join(out)
}
func AddFront(value string, entries []string) string {
	add := []string{}
	for _, s := range entries {
		if !Contains(Split(value), s) && !Contains(add, s) {
			add = append(add, s)
		}
	}
	if len(add) == 0 {
		return value
	}
	if value == "" {
		return Join(add)
	}
	return Join(add) + ";" + value
}
func Replace(value string, old, newEntries []string) string {
	parts := Split(value)
	out := []string{}
	inserted := false
	filtered := Remove(value, old)
	newList := []string{}
	for _, v := range newEntries {
		if !Contains(Split(filtered), v) && !Contains(newList, v) {
			newList = append(newList, v)
		}
	}
	for _, s := range parts {
		if Contains(old, s) {
			if !inserted {
				out = append(out, newList...)
				inserted = true
			}
			continue
		}
		out = append(out, s)
	}
	if !inserted {
		return AddFront(value, newList)
	}
	return Join(out)
}

// Notify is optional: fake/scratch stores do not send desktop-wide notifications.
func Notify(store Store) {
	if notifier, ok := store.(interface{ Notify() }); ok {
		notifier.Notify()
	}
}
