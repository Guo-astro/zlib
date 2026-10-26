package zlib

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchBookDetailsUsesEAPIAndKeepsFailedReferences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/eapi/book/2/bad" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"success":0,"error":"missing"}`)
			return
		}
		if r.URL.Path != "/eapi/book/1/hash" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"success":1,"book":{"id":1,"hash":"hash","title":"Book 1"}}`)
	}))
	defer server.Close()

	c := NewClient(WithDomain(server.URL))
	c.SetCookies(map[string]string{"remix_userid": "1", "remix_userkey": "k"})
	got := c.FetchBookDetails([]string{"1:hash", "2:bad"})
	if len(got) != 2 || got["1:hash"].Name != "Book 1" || got["2:bad"].Name != "" {
		t.Fatalf("FetchBookDetails() = %#v", got)
	}
}

func TestCookiesReturnsCopy(t *testing.T) {
	c := NewClient(WithDomain("https://example.com"))
	c.SetCookies(map[string]string{"remix_userid": "1"})
	c.Cookies()["remix_userid"] = "tampered"
	if got := c.Cookies()["remix_userid"]; got != "1" {
		t.Errorf("Cookies() leaked the internal map: %q", got)
	}
}
