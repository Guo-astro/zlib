package zlib

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	credMirror = "z-lib.gd"
	credCDN    = "files.example.test"
)

// fakeRoundTripper serves every request in memory, so these tests never
// resolve or contact the hosts they name.
type fakeRoundTripper func(*http.Request) (*http.Response, error)

func (f fakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func fakeResponse(r *http.Request, status int, header http.Header, body string) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode:    status,
		Header:        header,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       r,
	}
}

func fileResponse(r *http.Request) *http.Response {
	return fakeResponse(r, http.StatusOK, http.Header{
		"Content-Type":        {"application/octet-stream"},
		"Content-Disposition": {`attachment; filename="book.epub"`},
	}, "book body")
}

func redirectResponse(r *http.Request, location string) *http.Response {
	return fakeResponse(r, http.StatusFound, http.Header{"Location": {location}}, "")
}

// credentialRecorder records, per request URL, whether the request carried
// any of the account credentials.
type credentialRecorder struct {
	mu   sync.Mutex
	seen map[string][]bool
}

func (rec *credentialRecorder) record(r *http.Request) {
	cookie := r.Header.Get("Cookie")
	leaked := strings.Contains(cookie, "remix_userid=") || strings.Contains(cookie, "remix_userkey=") ||
		r.Header.Get("remix-userid") != "" || r.Header.Get("remix-userkey") != ""
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.seen == nil {
		rec.seen = map[string][]bool{}
	}
	rec.seen[r.URL.String()] = append(rec.seen[r.URL.String()], leaked)
}

func (rec *credentialRecorder) requests(rawURL string) []bool {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return rec.seen[rawURL]
}

func newCredentialTestClient(t *testing.T, handle func(*http.Request) *http.Response) (*Client, *credentialRecorder) {
	t.Helper()
	rec := &credentialRecorder{}
	c := NewClient(WithDomain("https://" + credMirror))
	c.SetCookies(map[string]string{"remix_userid": "fake-user", "remix_userkey": "FAKE-KEY"})
	c.httpClient.Transport = fakeRoundTripper(func(r *http.Request) (*http.Response, error) {
		rec.record(r)
		return handle(r), nil
	})
	return c, rec
}

func assertCredentials(t *testing.T, rec *credentialRecorder, rawURL string, want bool) {
	t.Helper()
	got := rec.requests(rawURL)
	if len(got) == 0 {
		t.Fatalf("%s was never requested", rawURL)
	}
	for i, leaked := range got {
		if leaked != want {
			t.Errorf("request %d to %s carried credentials = %v, want %v", i+1, rawURL, leaked, want)
		}
	}
}

func TestDownloadSendsCredentialsToMirror(t *testing.T) {
	fileURL := "https://" + credMirror + "/dl/42"
	c, rec := newCredentialTestClient(t, fileResponse)

	if _, err := c.Download(fileURL, t.TempDir(), nil); err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	assertCredentials(t, rec, fileURL, true)
}

func TestDownloadKeepsCredentialsOffExternalFileLink(t *testing.T) {
	for _, tc := range []struct {
		name string
		link string
	}{
		{"external https host", "https://" + credCDN + "/book.epub?md5=x&expires=1"},
		{"mirror host over http", "http://" + credMirror + "/book.epub"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			eapiURL := "https://" + credMirror + "/eapi/book/42/abc/file"
			c, rec := newCredentialTestClient(t, func(r *http.Request) *http.Response {
				if r.URL.String() == eapiURL {
					return fakeResponse(r, http.StatusOK, http.Header{"Content-Type": {"application/json"}},
						`{"success":1,"file":{"downloadLink":"`+tc.link+`"}}`)
				}
				return fileResponse(r)
			})

			if _, err := c.Download(eapiURL, t.TempDir(), nil); err != nil {
				t.Fatalf("Download() error = %v", err)
			}
			assertCredentials(t, rec, eapiURL, true)
			assertCredentials(t, rec, tc.link, false)
		})
	}
}

func TestDownloadRedirectDropsCredentials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target string
	}{
		{"cross-origin", "https://" + credCDN + "/book.epub"},
		{"https to http downgrade", "http://" + credMirror + "/book.epub"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fileURL := "https://" + credMirror + "/dl/42"
			c, rec := newCredentialTestClient(t, func(r *http.Request) *http.Response {
				if r.URL.String() == fileURL {
					return redirectResponse(r, tc.target)
				}
				return fileResponse(r)
			})

			if _, err := c.Download(fileURL, t.TempDir(), nil); err != nil {
				t.Fatalf("Download() error = %v", err)
			}
			assertCredentials(t, rec, fileURL, true)
			assertCredentials(t, rec, tc.target, false)
		})
	}
}

func TestDownloadRetriesKeepCredentialsOffExternalHost(t *testing.T) {
	setRetryBackoff(t, time.Millisecond)
	link := "https://" + credCDN + "/book.epub"
	var attempts int
	c, rec := newCredentialTestClient(t, func(r *http.Request) *http.Response {
		attempts++
		if attempts <= 2 {
			return fakeResponse(r, http.StatusBadGateway, nil, "")
		}
		return fileResponse(r)
	})

	if _, err := c.Download(link, t.TempDir(), nil); err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if got := len(rec.requests(link)); got != 3 {
		t.Fatalf("requests = %d, want 3", got)
	}
	assertCredentials(t, rec, link, false)
}
