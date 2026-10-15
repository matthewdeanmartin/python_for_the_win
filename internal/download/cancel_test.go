package download

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDownloadCancellation(t *testing.T) {
	for _, before := range []bool{false, true} {
		root := t.TempDir()
		source := filepath.Join(root, "input")
		data := make([]byte, 128*1024)
		if e := os.WriteFile(source, data, 0600); e != nil {
			t.Fatal(e)
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		cache := filepath.Join(root, "cache")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		client, e := New("", false)
		if e != nil {
			t.Fatal(e)
		}
		client.Progress = func(int64, int64) { cancel() }
		if before {
			cancel()
		}
		if _, e = client.Fetch(ctx, source, hash, cache); !errors.Is(e, context.Canceled) {
			t.Fatal(e)
		}
		entries, e := os.ReadDir(cache)
		if before && !os.IsNotExist(e) {
			t.Fatal("cancelled request created cache", e)
		}
		if len(entries) != 0 {
			t.Fatal("cancelled request retained data", entries)
		}
	}
}
