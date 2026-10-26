package zlib

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestWithDomain(t *testing.T) {
	c := NewClient(WithDomain("https://example.com"))

	if c.Domain() != "https://example.com" {
		t.Fatalf("Domain() = %q, want %q", c.Domain(), "https://example.com")
	}
	if c.Mode() != ClientModeEAPI {
		t.Fatalf("Mode() = %q, want EAPI", c.Mode())
	}
}

func TestNewClientPrefersExplicitProxyOverEnv(t *testing.T) {
	t.Setenv(EnvProxy, "http://127.0.0.1:7890")

	client := NewClient(WithProxy("http://127.0.0.1:1080"))

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatal("expected http transport to be configured")
	}

	reqURL, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatalf("failed to parse request url: %v", err)
	}
	proxyURL, err := transport.Proxy(&http.Request{URL: reqURL})
	if err != nil {
		t.Fatalf("proxy func returned error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:1080" {
		t.Fatalf("proxy url = %v, want %q", proxyURL, "http://127.0.0.1:1080")
	}
}

func TestNewClientIgnoresEmptyExplicitProxy(t *testing.T) {
	t.Setenv(EnvProxy, "http://127.0.0.1:7890")

	client := NewClient(WithProxy(""))

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatal("expected http transport to be configured")
	}

	reqURL, err := url.Parse("https://example.com")
	if err != nil {
		t.Fatalf("failed to parse request url: %v", err)
	}
	proxyURL, err := transport.Proxy(&http.Request{URL: reqURL})
	if err != nil {
		t.Fatalf("proxy func returned error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("proxy url = %v, want %q", proxyURL, "http://127.0.0.1:7890")
	}
}

func TestWithProxyPreservesDefaultTransportSettings(t *testing.T) {
	client := NewClient(WithProxy("http://127.0.0.1:7890"))

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatal("expected http transport to be configured")
	}

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Skip("http.DefaultTransport is not an *http.Transport on this platform")
	}
	if transport.MaxIdleConns != defaultTransport.MaxIdleConns {
		t.Errorf("MaxIdleConns = %d, want %d", transport.MaxIdleConns, defaultTransport.MaxIdleConns)
	}
	if transport.IdleConnTimeout != defaultTransport.IdleConnTimeout {
		t.Errorf("IdleConnTimeout = %v, want %v", transport.IdleConnTimeout, defaultTransport.IdleConnTimeout)
	}
	if transport.ForceAttemptHTTP2 != defaultTransport.ForceAttemptHTTP2 {
		t.Errorf("ForceAttemptHTTP2 = %v, want %v", transport.ForceAttemptHTTP2, defaultTransport.ForceAttemptHTTP2)
	}
}

func TestCheckRedirectKeepsEAPICredentialsOnAllowedHost(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://z-lib.gd/eapi/user/profile", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("remix-userid", "1")
	req.Header.Set("remix-userkey", "secret")

	if err := checkRedirect(req, nil); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	if req.Header.Get("remix-userkey") != "secret" {
		t.Error("checkRedirect() dropped EAPI credentials on an allowed host")
	}
}

func TestCheckRedirectStripsEAPICredentialsOffAllowlist(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://evil.example/eapi/user/profile", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("remix-userid", "1")
	req.Header.Set("remix-userkey", "secret")

	if err := checkRedirect(req, nil); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	if got := req.Header.Get("remix-userkey"); got != "" {
		t.Errorf("checkRedirect() forwarded remix-userkey to a foreign host: %q", got)
	}
	if got := req.Header.Get("remix-userid"); got != "" {
		t.Errorf("checkRedirect() forwarded remix-userid to a foreign host: %q", got)
	}
}

func TestEAPIRedirectKeepsCredentialsOffDowngradedURL(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target string
		want   bool
	}{
		{"https to http downgrade", "http://" + credMirror + "/eapi/user/profile", false},
		{"foreign host", "https://" + credCDN + "/eapi/user/profile", false},
		{"same mirror", "https://" + credMirror + "/eapi/user/profile/v2", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := "https://" + credMirror + "/eapi/user/profile"
			c, rec := newCredentialTestClient(t, func(r *http.Request) *http.Response {
				if r.URL.String() == start {
					return redirectResponse(r, tc.target)
				}
				return fakeResponse(r, http.StatusOK, http.Header{"Content-Type": {"application/json"}}, `{"success":1}`)
			})

			if _, err := c.eapiRequest(context.Background(), http.MethodGet, "/eapi/user/profile", nil); err != nil {
				t.Fatalf("eapiRequest() error = %v", err)
			}
			assertCredentials(t, rec, start, true)
			assertCredentials(t, rec, tc.target, tc.want)
		})
	}
}
