package state

import (
	"encoding/json"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"os"
	"path/filepath"
)

type Install struct {
	ID          string   `json:"id"`
	Version     string   `json:"version"`
	Arch        string   `json:"arch"`
	Threaded    bool     `json:"freethreaded"`
	Dir         string   `json:"dir"`
	URL         string   `json:"url"`
	SHA256      string   `json:"sha256"`
	Aliases     []string `json:"aliases"`
	InstalledAt string   `json:"installed_at"`
	Pip         bool     `json:"pip"`
	Source      string   `json:"source"`
	Transaction string   `json:"transaction,omitempty"`
}
type State struct {
	Raw      map[string]json.RawMessage
	Installs map[string]json.RawMessage
	Config   map[string]json.RawMessage
	Default  string
}

func New() *State {
	return &State{Raw: map[string]json.RawMessage{}, Installs: map[string]json.RawMessage{}, Config: map[string]json.RawMessage{}}
}
func Load(root string) (*State, error) {
	p, e := layout.Safe(root, "pftw.json")
	if e != nil {
		return nil, e
	}
	data, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		return New(), nil
	}
	if e != nil {
		return nil, e
	}
	s := New()
	if e = json.Unmarshal(data, &s.Raw); e != nil {
		return nil, fmt.Errorf("invalid pftw state: %w", e)
	}
	if s.Raw == nil {
		return nil, fmt.Errorf("invalid null state")
	}
	var schema int
	if e = json.Unmarshal(s.Raw["schema"], &schema); e != nil || schema != 1 {
		return nil, fmt.Errorf("unsupported state schema")
	}
	for k, target := range map[string]any{"installs": &s.Installs, "config": &s.Config, "default": &s.Default} {
		if b, ok := s.Raw[k]; ok {
			if e = json.Unmarshal(b, target); e != nil {
				return nil, e
			}
		}
	}
	if s.Installs == nil {
		s.Installs = map[string]json.RawMessage{}
	}
	if s.Config == nil {
		s.Config = map[string]json.RawMessage{}
	}
	for k := range s.Installs {
		if !layout.ValidKey(k) {
			return nil, fmt.Errorf("unsafe install key in state")
		}
		v, e := s.Get(k)
		if e != nil {
			return nil, e
		}
		expected := filepath.Join("versions", k)
		if filepath.Clean(filepath.FromSlash(v.Dir)) != expected {
			return nil, fmt.Errorf("unexpected directory for %s", k)
		}
	}
	return s, nil
}
func (s *State) Get(k string) (Install, error) {
	var v Install
	b, ok := s.Installs[k]
	if !ok {
		return v, fmt.Errorf("%s is not installed", k)
	}
	e := json.Unmarshal(b, &v)
	return v, e
}
func (s *State) Set(k string, v Install) error {
	if !layout.ValidKey(k) {
		return fmt.Errorf("invalid install key")
	}
	raw := map[string]json.RawMessage{}
	if old, ok := s.Installs[k]; ok {
		if e := json.Unmarshal(old, &raw); e != nil {
			return e
		}
	}
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	fresh := map[string]json.RawMessage{}
	_ = json.Unmarshal(b, &fresh)
	for k, v := range fresh {
		raw[k] = v
	}
	s.Installs[k], e = json.Marshal(raw)
	return e
}
func (s *State) Setting(k string) string { var v string; _ = json.Unmarshal(s.Config[k], &v); return v }
func (s *State) Save(root string) error {
	raw := map[string]json.RawMessage{}
	for k, v := range s.Raw {
		raw[k] = v
	}
	for k, v := range map[string]any{"schema": 1, "installs": s.Installs, "config": s.Config, "default": s.Default} {
		b, e := json.Marshal(v)
		if e != nil {
			return e
		}
		raw[k] = b
	}
	return Write(root, "pftw.json", raw)
}
func Write(root, name string, v any) error {
	p, e := layout.Safe(root, name)
	if e != nil {
		return e
	}
	if _, e := layout.Safe(root, name+".tmp"); e != nil {
		return e
	}
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.OpenFile(p+".tmp", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(append(b, '\n'))
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return replace(p+".tmp", p)
}
