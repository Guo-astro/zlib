package zlib

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// useUserMirrors installs mirrors for one test and restores the previous set.
func useUserMirrors(t *testing.T, mirrors ...string) {
	t.Helper()
	previous := UserMirrors()
	if err := SetUserMirrors(mirrors); err != nil {
		t.Fatalf("SetUserMirrors(%v) error = %v", mirrors, err)
	}
	t.Cleanup(func() { _ = SetUserMirrors(previous) })
}

func TestParseMirrorOrigin(t *testing.T) {
	for _, tt := range []struct {
		value, want string
		wantErr     bool
	}{
		{value: "mirror.example", want: "https://mirror.example"},
		{value: " https://Mirror.Example/ ", want: "https://mirror.example"},
		{value: "https://books.mirror.example.", want: "https://books.mirror.example"},
		{value: "xn--80ak6aa92e.com", want: "https://xn--80ak6aa92e.com"},
		{value: "", wantErr: true},
		{value: "http://mirror.example", wantErr: true},
		{value: "https://mirror.example:8443", wantErr: true},
		{value: "https://mirror.example/eapi", wantErr: true},
		{value: "https://user@mirror.example", wantErr: true},
		{value: "https://mirror.example/?x=1", wantErr: true},
		{value: "https://203.0.113.7", wantErr: true},
		{value: "https://[2001:db8::1]", wantErr: true},
		{value: "localhost", wantErr: true},
		{value: "mirror..example", wantErr: true},
		{value: "-mirror.example", wantErr: true},
		{value: "mirror_.example", wantErr: true},
		{value: "bücher.example", wantErr: true},
	} {
		got, err := ParseMirrorOrigin(tt.value)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("ParseMirrorOrigin(%q) = %q, %v; want %q, error %v", tt.value, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestUserMirrorsTrustExactHostOnly(t *testing.T) {
	if _, err := ParseAllowedDomain("https://mirror.example"); err == nil {
		t.Fatal("an unconfigured mirror was trusted")
	}
	useUserMirrors(t, "mirror.example")

	if got, err := ParseAllowedDomain("https://MIRROR.example/"); err != nil || got != "https://mirror.example" {
		t.Fatalf("trusted mirror rejected: %q, %v", got, err)
	}
	for _, untrusted := range []string{
		"https://evil.mirror.example", // a subdomain is a different host
		"https://mirror.example.evil", // suffix tricks
		"https://xmirror.example",
		"http://mirror.example", // plain HTTP would expose the key
	} {
		if _, err := ParseAllowedDomain(untrusted); err == nil {
			t.Errorf("ParseAllowedDomain(%q) trusted a host that is not the mirror", untrusted)
		}
	}
}

func TestSetUserMirrorsKeepsValidEntriesAndReportsInvalidOnes(t *testing.T) {
	previous := UserMirrors()
	t.Cleanup(func() { _ = SetUserMirrors(previous) })

	err := SetUserMirrors([]string{"b.example", "http://plain.example", "A.example", "b.example"})
	if err == nil || !strings.Contains(err.Error(), "plain.example") {
		t.Fatalf("SetUserMirrors() error = %v, want the invalid entry named", err)
	}
	if got, want := UserMirrors(), []string{"https://b.example", "https://a.example"}; !slices.Equal(got, want) {
		t.Fatalf("UserMirrors() = %v, want %v (order kept, deduplicated, invalid skipped)", got, want)
	}
	if err := SetUserMirrors(nil); err != nil || len(UserMirrors()) != 0 {
		t.Fatalf("clearing mirrors: %v, %v", err, UserMirrors())
	}
}

func TestUserMirrorsReturnsCopy(t *testing.T) {
	useUserMirrors(t, "mirror.example")
	got := UserMirrors()
	got[0] = "https://evil.example"
	if _, err := ParseAllowedDomain("https://evil.example"); err == nil {
		t.Fatal("UserMirrors() exposed mutable internal storage")
	}
}

func TestIsBuiltInDomain(t *testing.T) {
	for raw, want := range map[string]bool{
		"z-lib.gd":                true,
		"https://books.z-lib.gl":  true,
		"mirror.example":          false,
		"https://z-lib.gd.evil.x": false,
		"not a domain":            false,
	} {
		if got := IsBuiltInDomain(raw); got != want {
			t.Errorf("IsBuiltInDomain(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestKnownDomainsAppendsUserMirrors(t *testing.T) {
	useUserMirrors(t, "mirror.example", "z-lib.gd")
	got := KnownDomains()
	if got[len(got)-1] != "https://mirror.example" {
		t.Fatalf("KnownDomains() = %v, want the user mirror last", got)
	}
	count := 0
	for _, d := range got {
		if d == "https://z-lib.gd" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("KnownDomains() listed a mirror that is also built in %d times: %v", count, got)
	}
}

func TestRedirectKeepsCredentialsOnlyOnTheExactMirror(t *testing.T) {
	useUserMirrors(t, "mirror.example")
	for target, keep := range map[string]bool{
		"https://mirror.example/eapi/user/profile":      true,
		"https://evil.mirror.example/eapi/user/profile": false,
		"http://mirror.example/eapi/user/profile":       false,
	} {
		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("remix-userkey", "fixture-key")
		if err := checkRedirect(req, nil); err != nil {
			t.Fatal(err)
		}
		if got := req.Header.Get("remix-userkey") != ""; got != keep {
			t.Errorf("redirect to %s kept credentials = %v, want %v", target, got, keep)
		}
	}
}

func TestProbeRecognizesZLibraryDomainsInfo(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		want       bool
	}{
		{"zlibrary", `{"success":1,"domains":[{"domain":"z-library.sk"}]}`, true},
		{"empty list", `{"success":1,"domains":[]}`, true},
		{"failure", `{"success":0,"domains":[]}`, false},
		{"no domains", `{"success":1}`, false},
		{"other json", `{"ok":true}`, false},
		{"array", `[]`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			check := ProbeDomainWithOptions(context.Background(), server.URL, DomainProbeOptions{Path: eapiDomainsInfoPath})
			if check.Status != DomainStatusHealthy || check.zlibEAPI != tt.want {
				t.Fatalf("status %s, zlibEAPI %v; want healthy, %v", check.Status, check.zlibEAPI, tt.want)
			}
		})
	}
}

func TestMirrorProbeError(t *testing.T) {
	const origin = "https://mirror.example"
	ok := DomainCheck{Domain: origin, Status: DomainStatusHealthy, FinalURL: origin + eapiDomainsInfoPath, zlibEAPI: true}
	if err := mirrorProbeError(origin, ok); err != nil {
		t.Fatalf("a verified mirror was rejected: %v", err)
	}
	for name, tt := range map[string]struct {
		check DomainCheck
		want  string
	}{
		"blocked":      {DomainCheck{Status: DomainStatusDiamWallBlocked, Detail: "DiamWall anti-bot response"}, "diamwall_blocked"},
		"unreachable":  {DomainCheck{Status: DomainStatusNetworkError, Detail: "no such host"}, "no such host"},
		"redirected":   {DomainCheck{Status: DomainStatusHealthy, FinalURL: "https://z-lib.gd" + eapiDomainsInfoPath, zlibEAPI: true}, "redirects to https://z-lib.gd"},
		"to http":      {DomainCheck{Status: DomainStatusHealthy, FinalURL: "http://mirror.example" + eapiDomainsInfoPath, zlibEAPI: true}, "redirects to http://"},
		"not zlibrary": {DomainCheck{Status: DomainStatusHealthy, FinalURL: origin + eapiDomainsInfoPath}, "not with the Z-Library EAPI"},
	} {
		err := mirrorProbeError(origin, tt.check)
		if !errors.Is(err, ErrMirrorUnverified) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error = %v, want ErrMirrorUnverified mentioning %q", name, err, tt.want)
		}
	}
}

func TestVerifyUserMirrorRejectsMalformedInputWithoutNetwork(t *testing.T) {
	if _, err := VerifyUserMirror(context.Background(), "http://mirror.example", DomainProbeOptions{}); err == nil || errors.Is(err, ErrMirrorUnverified) {
		t.Fatalf("VerifyUserMirror(http) error = %v, want a format error before any probe", err)
	}
}
