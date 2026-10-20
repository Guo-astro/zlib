package cli

import (
	"errors"
	"strings"
	"testing"

	zlib "github.com/heartleo/zlib"
)

// doctor used to exit 0 with "No available EAPI domains found." (or `[]`),
// so a script could not tell an unreachable network from success.
func TestNoDomainsErrorFailsOnlyWhenNothingIsUsable(t *testing.T) {
	err := noDomainsError(nil)
	if !errors.Is(err, zlib.ErrNoDomain) || !strings.Contains(err.Error(), "--proxy") {
		t.Fatalf("noDomainsError(nil) = %v, want ErrNoDomain with a --proxy hint", err)
	}
	if err := noDomainsError([]zlib.DomainCheck{{Domain: "https://z-lib.gd"}}); err != nil {
		t.Fatalf("noDomainsError with a usable domain = %v", err)
	}
}

func TestAvailableDomains(t *testing.T) {
	results := []zlib.DomainCheck{
		{Domain: "https://z-lib.gd", Status: zlib.DomainStatusHealthy, FinalURL: "https://z-lib.gd/eapi/info/domains"},
		{Domain: "https://challenge.example", Status: zlib.DomainStatusChallenged},
		{Domain: "https://blocked.example", Status: zlib.DomainStatusDiamWallBlocked},
		{Domain: "https://zliba.ru", Status: zlib.DomainStatusHealthy, FinalURL: "https://z-lib.by/eapi/info/domains"},
	}
	got := availableDomains(results)
	if len(got) != 1 || got[0].Domain != "https://z-lib.gd" {
		t.Fatalf("availableDomains() = %+v", got)
	}
	if got := availableDomains(nil); got == nil || len(got) != 0 {
		t.Fatalf("availableDomains(nil) = %+v, want empty JSON array", got)
	}
}

func TestValidateDoctorProxy(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "unset", value: "", want: ""},
		{name: "HTTP", value: "http://127.0.0.1:7890", want: "http://127.0.0.1:7890"},
		{name: "SOCKS5", value: "socks5://127.0.0.1:9050", want: "socks5://127.0.0.1:9050"},
		{name: "SOCKS5 hostname", value: "socks5h://127.0.0.1:9050", want: "socks5h://127.0.0.1:9050"},
		{name: "missing scheme", value: "127.0.0.1:7890", wantErr: true},
		{name: "unsupported scheme", value: "ftp://127.0.0.1:21", wantErr: true},
		{name: "path", value: "http://127.0.0.1:7890/proxy", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateDoctorProxy(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateDoctorProxy(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("validateDoctorProxy(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
