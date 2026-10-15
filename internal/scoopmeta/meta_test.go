package scoopmeta

import (
	"encoding/json"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestSchema(t *testing.T) {
	read := func(p string) any {
		t.Helper()
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		var d any
		if e = json.Unmarshal(b, &d); e != nil {
			t.Fatal(e)
		}
		return d
	}
	c := jsonschema.NewCompiler()
	schema := read(filepath.Join("..", "..", "testdata", "scoop-schema.json"))
	if e := c.AddResource("https://pftw.test/scoop.json", schema); e != nil {
		t.Fatal(e)
	}
	compiled, e := c.Compile("https://pftw.test/scoop.json")
	if e != nil {
		t.Fatal(e)
	}
	manifest := filepath.Join("..", "..", "bucket", "pftw.json")
	if e = compiled.Validate(read(manifest)); e != nil {
		t.Fatal(e)
	}
	dest := filepath.Join(t.TempDir(), "manifest.json")
	if e = Enrich(manifest, dest); e != nil {
		t.Fatal(e)
	}
	if e = compiled.Validate(read(dest)); e != nil {
		t.Fatal(e)
	}
}
