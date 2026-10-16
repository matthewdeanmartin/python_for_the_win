package cli

import (
	"runtime/debug"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		Version, Commit = fromBuildInfo(Version, Commit, info)
	}
}

// fromBuildInfo fills in builds that goreleaser didn't stamp, such as go install ...@v0.1.0, from
// what Go embeds in every binary. Only a clean release tag is adopted: local builds carry a
// pseudo-version (v0.0.0-<time>-<hash>, often +dirty) and must stay "dev" so self-update won't
// silently replace them.
func fromBuildInfo(version, commit string, info *debug.BuildInfo) (string, string) {
	tag := info.Main.Version
	if version == "dev" && semver.IsValid(tag) && semver.Build(tag) == "" && !module.IsPseudoVersion(tag) {
		version = strings.TrimPrefix(tag, "v")
	}
	if commit == "unknown" {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				commit = s.Value[:7]
			}
		}
	}
	return version, commit
}
