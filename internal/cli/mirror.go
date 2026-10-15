package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/spf13/cobra"
)

func (o *options) mirrorCommand() *cobra.Command {
	var out string
	c := &cobra.Command{Use: "mirror SPEC [SPEC...] --out DIRECTORY", Short: "Create a self-hostable index and verified runtime ZIPs", Args: cobra.MinimumNArgs(1)}
	c.Flags().StringVar(&out, "out", "", "New directory; existing directories are never overwritten")
	c.RunE = func(c *cobra.Command, args []string) error {
		if out == "" {
			return errs.New(2, "mirror requires --out DIRECTORY")
		}
		target, e := filepath.Abs(out)
		if e != nil {
			return e
		}
		if _, e = os.Lstat(target); !os.IsNotExist(e) {
			return errs.New(2, "mirror output already exists or cannot be inspected: %s", target)
		}
		s, e := o.load()
		if e != nil {
			return e
		}
		readonly := *o
		readonly.dryRun = true
		entries, d, _, e := readonly.entries(c, s)
		if e != nil {
			return e
		}
		selected := []feed.Entry{}
		seen := map[string]bool{}
		for _, spec := range args {
			v, e := resolve.Pick(entries, resolve.Options{Spec: spec, Arch: o.arch})
			if e != nil {
				return e
			}
			id := v.Key() + "/" + v.Version
			if seen[id] {
				continue
			}
			seen[id] = true
			selected = append(selected, v)
		}
		sort.Slice(selected, func(i, j int) bool {
			return selected[i].Key()+selected[i].Version < selected[j].Key()+selected[j].Version
		})
		changes := []string{"Create " + target, "Write " + filepath.Join(target, "index.json"), "Write " + filepath.Join(target, "mirror-manifest.json")}
		for _, v := range selected {
			changes = append(changes, fmt.Sprintf("Verify Python %s and write %s", v.Version, filepath.Join(target, strings.ToLower(v.Hash["sha256"])+".zip")))
		}
		if o.dryRun {
			return o.output(c, map[string]any{"dry_run": true, "changes": changes}, "Dry run: no files or settings changed.\n"+strings.Join(changes, "\n"))
		}
		work, e := os.MkdirTemp(filepath.Dir(target), ".pftw-mirror-")
		if e != nil {
			return e
		}
		defer os.RemoveAll(work)
		payloads := map[string]string{}
		for i, v := range selected {
			p, e := d.Fetch(c.Context(), v.URL, v.Hash["sha256"], work)
			if e != nil {
				return e
			}
			name := strings.ToLower(v.Hash["sha256"]) + ".zip"
			payloads[name] = p
			selected[i].URL = name
		}
		if e = c.Context().Err(); e != nil {
			return e
		}
		if e = os.Mkdir(target, 0700); e != nil {
			return e
		}
		created := []string{}
		complete := false
		defer func() {
			if !complete {
				for _, p := range created {
					_ = os.Remove(p)
				}
				_ = os.Remove(target)
			}
		}()
		write := func(name string, r io.Reader) error {
			p := filepath.Join(target, name)
			f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				return e
			}
			created = append(created, p)
			_, e = io.Copy(f, r)
			if e == nil {
				e = f.Sync()
			}
			ce := f.Close()
			if e == nil {
				e = ce
			}
			return e
		}
		for name, p := range payloads {
			f, e := os.Open(p)
			if e != nil {
				return e
			}
			e = write(name, f)
			f.Close()
			if e != nil {
				return e
			}
		}
		manifest, e := json.MarshalIndent(map[string]any{"format": 1, "versions": selected, "note": "Original payload SHA-256 values preserved. Serve this directory over HTTPS; no server is started."}, "", "  ")
		if e != nil {
			return e
		}
		if e = write("mirror-manifest.json", strings.NewReader(string(manifest)+"\n")); e != nil {
			return e
		}
		index, e := json.MarshalIndent(feed.Index{Versions: selected}, "", "  ")
		if e != nil {
			return e
		}
		// Publish index last: a reader never sees an index before all verified payloads exist.
		if e = write("index.json", strings.NewReader(string(index)+"\n")); e != nil {
			return e
		}
		complete = true
		return o.output(c, map[string]any{"mirror": target, "versions": len(selected)}, "Created "+target+". Use --source "+target+", or serve its contents over HTTPS.")
	}
	return c
}
