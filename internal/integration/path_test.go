package integration

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type truncatingStore struct{ envpath.MemStore }

func (s *truncatingStore) Set(v envpath.Value) error {
	v.Text = v.Text[:1024]
	s.Value = v
	return nil
}

func TestTruncatedWriteNotCommitted(t *testing.T) {
	r := t.TempDir()
	s := state.New()
	s.Default = "3.15-64"
	original := envpath.Value{Text: strings.Repeat(`C:\Tools;`, 2000) + "TAIL", Kind: 2, Exists: true}
	store := &truncatingStore{envpath.MemStore{Value: original}}
	err := Apply(r, s, store, true)
	if err == nil || !strings.Contains(err.Error(), "read-back") {
		t.Fatalf("truncated write not detected: %v", err)
	}
	if s.Path().Mode == "first" {
		t.Fatal("failed write committed to state")
	}
	if _, err := os.Stat(filepath.Join(r, "path-transaction.json")); err != nil {
		t.Fatal("missing recovery journal", err)
	}
	backups, err := filepath.Glob(filepath.Join(r, "logs", "path-backup-*.txt"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("missing backup: %v %v", backups, err)
	}
	b, err := os.ReadFile(backups[0])
	if err != nil || string(b) != original.Text {
		t.Fatal("original PATH not preserved in backup", err)
	}
}

func TestLifecyclePreservesExternalEdits(t *testing.T) {
	r := t.TempDir()
	s := state.New()
	s.Default = "3.15-64"
	store := &envpath.MemStore{Value: envpath.Value{Text: `%USERPROFILE%\bin;;C:\Tools;`, Kind: 2, Exists: true}}
	original := store.Value
	if e := Apply(r, s, store, true); e != nil {
		t.Fatal(e)
	}
	store.Value.Text += "Extra;"
	s.Default = "3.16-64"
	if e := Apply(r, s, store, true); e != nil {
		t.Fatal(e)
	}
	if !envpath.Contains(envpath.Split(store.Value.Text), filepath.Join(r, "versions", "3.16-64")) {
		t.Fatal(store.Value)
	}
	s.Default = ""
	if e := Apply(r, s, store, false); e != nil {
		t.Fatal(e)
	}
	if store.Value.Text != original.Text+"Extra;" {
		t.Fatal(store.Value, original)
	}
	if !s.Path().BackedUp {
		t.Fatal("no backup")
	}
}
func TestRecoverAfterRegistryWrite(t *testing.T) {
	r := t.TempDir()
	s := state.New()
	if e := s.Save(r); e != nil {
		t.Fatal(e)
	}
	j := journal{Before: envpath.Value{Text: "before", Kind: 2, Exists: true}, After: envpath.Value{Text: "after", Kind: 2, Exists: true}, Path: state.PathState{Mode: "first"}}
	if e := state.Write(r, "path-transaction.json", j); e != nil {
		t.Fatal(e)
	}
	store := &envpath.MemStore{Value: j.After}
	if e := Recover(r, store); e != nil {
		t.Fatal(e)
	}
	s, e := state.Load(r)
	if e != nil || s.Path().Mode != "first" {
		t.Fatal(s, e)
	}
	if _, e = os.Stat(filepath.Join(r, "path-transaction.json")); !os.IsNotExist(e) {
		t.Fatal(e)
	}
}
func TestRecoveryDoesNotClobberConcurrentEdit(t *testing.T) {
	r := t.TempDir()
	j := journal{Before: envpath.Value{Text: "before"}, After: envpath.Value{Text: "after"}}
	_ = state.Write(r, "path-transaction.json", j)
	store := &envpath.MemStore{Value: envpath.Value{Text: "external"}}
	if e := Recover(r, store); e == nil || store.Value.Text != "external" {
		t.Fatal(e)
	}
}
func TestAbsentPathRestored(t *testing.T) {
	r := t.TempDir()
	s := state.New()
	s.Default = "3.15-64"
	store := &envpath.MemStore{Value: envpath.Value{Kind: 2}}
	original := store.Value
	if e := Apply(r, s, store, true); e != nil {
		t.Fatal(e)
	}
	s.Default = ""
	if e := Apply(r, s, store, false); e != nil {
		t.Fatal(e)
	}
	if store.Value != original {
		t.Fatal(store.Value, original)
	}
}

func TestPreexistingEntriesNotClaimed(t *testing.T) {
	r := t.TempDir()
	s := state.New()
	s.Default = "3.15-64"
	original := envpath.Value{Text: filepath.Join(r, "shims") + ";Foreign;", Kind: 1, Exists: true}
	store := &envpath.MemStore{Value: original}
	if e := Apply(r, s, store, true); e != nil {
		t.Fatal(e)
	}
	for _, v := range s.Path().Added {
		if envpath.Equal(v, filepath.Join(r, "shims")) {
			t.Fatal("claimed preexisting entry")
		}
	}
	s.Default = ""
	if e := Apply(r, s, store, false); e != nil {
		t.Fatal(e)
	}
	if store.Value != original {
		t.Fatal(store.Value, original)
	}
}
