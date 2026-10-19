package zlib

import "testing"

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
