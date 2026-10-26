package zlib

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteDownloadHistory(t *testing.T) {
	for _, body := range []string{`{"success":1}`, `{"success":0,"error":"denied"}`, `invalid`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/eapi/user/book/42/delete" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("remix-userid") != "123" {
					t.Error("missing authentication")
				}
				fmt.Fprint(w, body)
			}))
			defer s.Close()
			c := NewClient(WithDomain(s.URL))
			if c.DeleteDownloadHistory("42") != ErrNotLoggedIn {
				t.Fatal("expected login requirement")
			}
			c.SetCookies(map[string]string{"remix_userid": "123", "remix_userkey": "key"})
			if c.DeleteDownloadHistory("../42") == nil || calls != 0 {
				t.Fatal("invalid ID sent to server")
			}
			err := c.DeleteDownloadHistory("42:abcdef")
			if (err == nil) != (body == `{"success":1}`) {
				t.Fatalf("error = %v", err)
			}
			if calls != 1 {
				t.Fatalf("calls = %d", calls)
			}
		})
	}
}
