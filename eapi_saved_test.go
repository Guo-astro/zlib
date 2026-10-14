package zlib

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSavedBooksPagesAndMapsEAPIResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/eapi/user/book/saved" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("limit") != "50" {
			t.Errorf("query = %v", r.URL.Query())
		}
		if r.Header.Get("remix-userid") != "123" || r.Header.Get("remix-userkey") != "key123" {
			t.Error("missing EAPI credentials")
		}
		fmt.Fprint(w, `{"success":1,"books":[{"id":42,"hash":"abc","title":"Saved Book","author":"A Writer","extension":"epub","filesizeString":"1 MB"}],"pagination":{"current":2,"total_pages":4}}`)
	}))
	defer server.Close()
	client := NewClient(WithDomain(server.URL))
	if _, err := client.SavedBooks(2); err != ErrNotLoggedIn {
		t.Fatalf("unauthenticated error = %v", err)
	}
	client.SetCookies(map[string]string{"remix_userid": "123", "remix_userkey": "key123"})
	result, err := client.SavedBooks(2)
	if err != nil || result.Page != 2 || result.TotalPages != 4 || len(result.Books) != 1 {
		t.Fatalf("result = %+v, error = %v", result, err)
	}
	book := result.Books[0]
	if book.ID != "42" || book.Hash != "abc" || book.Name != "Saved Book" || book.Extension != "epub" || book.Size != "1 MB" {
		t.Fatalf("book = %+v", book)
	}
}

func TestSavedBooksReportsEAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"success":0,"error":"Not authorized"}`)
	}))
	defer server.Close()
	client := NewClient(WithDomain(server.URL))
	client.SetCookies(map[string]string{"remix_userid": "123", "remix_userkey": "key123"})
	_, err := client.SavedBooks(1)
	if err == nil || !strings.Contains(err.Error(), "Not authorized") {
		t.Fatalf("error = %v", err)
	}
}
