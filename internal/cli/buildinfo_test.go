package cli

import (
	"runtime/debug"
	"testing"
)

func TestFromBuildInfo(t *testing.T) {
	revision := []debug.BuildSetting{{Key: "vcs.revision", Value: "eb93ef416eec7fcca0caf0034d78788de57171d4"}}
	for _, tc := range []struct {
		name, stampedVersion, stampedCommit, moduleVersion string
		settings                                           []debug.BuildSetting
		wantVersion, wantCommit                            string
	}{
		{"go install of a release", "dev", "unknown", "v0.1.0", nil, "0.1.0", "unknown"},
		{"go install of a pre-release", "dev", "unknown", "v0.2.0-rc.1", nil, "0.2.0-rc.1", "unknown"},
		{"goreleaser stamp wins", "0.1.0", "abc1234", "v9.9.9", revision, "0.1.0", "abc1234"},
		{"local checkout build", "dev", "unknown", "v0.0.0-20261015231032-eb93ef416eec+dirty", revision, "dev", "eb93ef4"},
		{"clean pseudo-version", "dev", "unknown", "v0.0.0-20261015231032-eb93ef416eec", nil, "dev", "unknown"},
		{"go run", "dev", "unknown", "(devel)", nil, "dev", "unknown"},
	} {
		info := &debug.BuildInfo{Main: debug.Module{Version: tc.moduleVersion}, Settings: tc.settings}
		v, c := fromBuildInfo(tc.stampedVersion, tc.stampedCommit, info)
		if v != tc.wantVersion || c != tc.wantCommit {
			t.Errorf("%s: got %q %q, want %q %q", tc.name, v, c, tc.wantVersion, tc.wantCommit)
		}
	}
}
