package zlib

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBookFeeds(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		load func(*Client) ([]Book, error)
	}{
		{"recommended", recommendedBooksPath, (*Client).RecommendedBooks},
		{"most-popular", mostPopularBooksPath, (*Client).MostPopularBooks},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != test.path {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("remix-userid") != "123" || r.Header.Get("remix-userkey") != "key123" {
					t.Error("missing EAPI credentials")
				}
				fmt.Fprint(w, `{"success":1,"books":[{"id":42,"hash":"abc","title":"A Book","author":"A Writer","cover":"/cover.jpg"}]}`)
			}))
			defer server.Close()
			client := NewClient(WithDomain(server.URL))
			if _, err := test.load(client); err != ErrNotLoggedIn {
				t.Fatalf("unauthenticated error = %v", err)
			}
			client.SetCookies(map[string]string{"remix_userid": "123", "remix_userkey": "key123"})
			books, err := test.load(client)
			if err != nil || len(books) != 1 {
				t.Fatalf("books = %+v, error = %v", books, err)
			}
			if books[0].ID != "42" || books[0].Hash != "abc" || books[0].Name != "A Book" || books[0].Authors[0] != "A Writer" {
				t.Fatalf("book = %+v", books[0])
			}
		})
	}
}

func TestBookFeedRejectsServerErrorAndMissingReference(t *testing.T) {
	for _, test := range []struct {
		body string
		want string
	}{
		{`{"success":0,"error":"Not authorized"}`, "Not authorized"},
		{`{"success":1,"books":[{"id":42,"title":"No hash"}]}`, "missing id or hash"},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, test.body)
		}))
		client := NewClient(WithDomain(server.URL))
		client.SetCookies(map[string]string{"remix_userid": "123", "remix_userkey": "key123"})
		_, err := client.RecommendedBooks()
		server.Close()
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("error = %v, want %q", err, test.want)
		}
	}
}
