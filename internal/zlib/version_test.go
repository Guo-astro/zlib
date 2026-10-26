package zlib

import (
	"os"
	"reflect"
	"regexp"
	"testing"
)

func TestCanonicalVersion(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"0.0.10", "v0.0.10"},
		{"v0.0.10", "v0.0.10"},
		{"0.0.11-SNAPSHOT-fb0e248", "v0.0.11-SNAPSHOT-fb0e248"},
		{"v0.0.11-0.20261019110558-9e05d988daa5", "v0.0.11-0.20261019110558-9e05d988daa5"},
		{"dev", "dev"},
		{"", ""},
	} {
		if got := canonicalVersion(tt.in); got != tt.want {
			t.Errorf("canonicalVersion(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// versionPackage anchors this package's import path for the ldflags check.
type versionPackage struct{}

// TestGoReleaserLdflagsTargetThisPackage guards release builds: -X with a
// stale package path is silently ignored, and the binary would report "dev".
func TestGoReleaserLdflagsTargetThisPackage(t *testing.T) {
	data, err := os.ReadFile("../../.goreleaser.yml")
	if err != nil {
		t.Fatal(err)
	}
	pkg := reflect.TypeOf(versionPackage{}).PkgPath()
	matches := regexp.MustCompile(`-X ([^=\s]+)\.(Version|Commit|Date)=`).FindAllStringSubmatch(string(data), -1)
	if len(matches) != 3 {
		t.Fatalf("found %d ldflags -X entries for Version, Commit and Date, want 3", len(matches))
	}
	for _, m := range matches {
		if m[1] != pkg {
			t.Errorf("ldflags set %s.%s, want %s.%s", m[1], m[2], pkg, m[2])
		}
	}
}
