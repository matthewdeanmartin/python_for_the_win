// Package scoopmeta adds update metadata omitted by GoReleaser's Scoop generator.
package scoopmeta

import (
	"encoding/json"
	"os"
)

func Enrich(source, target string) error {
	b, e := os.ReadFile(source)
	if e != nil {
		return e
	}
	var doc map[string]any
	if e = json.Unmarshal(b, &doc); e != nil {
		return e
	}
	doc["checkver"] = "github"
	base := "https://github.com/matthewdeanmartin/python_for_the_win/releases/download/v$version/"
	arches := map[string]any{}
	for arch, goarch := range map[string]string{"64bit": "amd64", "arm64": "arm64"} {
		arches[arch] = map[string]any{"url": base + "pftw_$version_windows_" + goarch + ".zip", "hash": map[string]string{"url": base + "checksums.txt"}}
	}
	doc["autoupdate"] = map[string]any{"architecture": arches}
	b, e = json.MarshalIndent(doc, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(target, append(b, 10), 0600)
}
