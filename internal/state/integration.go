package state

import "encoding/json"

type PathState struct {
	Mode           string   `json:"mode"`
	Added          []string `json:"added"`
	Backup         string   `json:"backup"`
	BackedUp       bool     `json:"backed_up"`
	OriginalExists bool     `json:"original_exists"`
	OriginalKind   uint32   `json:"original_kind"`
}

func (s *State) Path() PathState { var p PathState; _ = json.Unmarshal(s.Raw["path"], &p); return p }
func (s *State) SetPath(p PathState) {
	raw := map[string]json.RawMessage{}
	_ = json.Unmarshal(s.Raw["path"], &raw)
	if raw == nil {
		raw = map[string]json.RawMessage{}
	}
	b, _ := json.Marshal(p)
	fresh := map[string]json.RawMessage{}
	_ = json.Unmarshal(b, &fresh)
	for k, v := range fresh {
		raw[k] = v
	}
	s.Raw["path"], _ = json.Marshal(raw)
}
func (s *State) Portable() bool     { var v bool; _ = json.Unmarshal(s.Raw["portable"], &v); return v }
func (s *State) SetPortable(v bool) { s.Raw["portable"], _ = json.Marshal(v) }
