package feed

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
)

func TestIndexSizeLimit(t *testing.T) {
	for _, size := range []int{maxIndexBytes, maxIndexBytes + 1} {
		_, err := readIndex(strings.NewReader(strings.Repeat(" ", size)))
		if (err != nil) != (size > maxIndexBytes) {
			t.Fatalf("size=%d err=%v", size, err)
		}
	}
}

func TestOversizedCachedIndexRefetched(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.json")
	if err := os.WriteFile(source, []byte(`{"versions":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(source)))
	cache := filepath.Join(dir, "index-"+hash[:12]+".json")
	if err := os.WriteFile(cache, []byte(strings.Repeat(" ", maxIndexBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := download.New("", false)
	if err != nil {
		t.Fatal(err)
	}
	i, err := Load(c, source, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = i.Next(context.Background()); err != nil {
		t.Fatalf("oversized cache did not fall back to source: %v", err)
	}
}
