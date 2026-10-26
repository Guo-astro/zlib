package zlib

import "runtime/debug"

// Version is always reported as vX.Y.Z (or a vX.Y.Z-based pre-release or
// pseudo-version), matching the git tag; "dev" marks a build without version
// metadata. See docs/versioning.md.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// init fills version metadata from the embedded build info when it was not
// injected via ldflags (e.g. when installed with `go install ...@latest`).
func init() {
	defer func() { Version = canonicalVersion(Version) }()
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if s.Value != "" {
				Commit = s.Value
			}
		case "vcs.time":
			if s.Value != "" {
				Date = s.Value
			}
		}
	}
}

// canonicalVersion adds the v prefix to a numeric version such as 0.0.10, so
// a build that injects the version without it still reports v0.0.10.
func canonicalVersion(v string) string {
	if v != "" && v[0] >= '0' && v[0] <= '9' {
		return "v" + v
	}
	return v
}
