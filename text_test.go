package zlib

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A mirror controls every string in an EAPI response. Titles, authors, years
// and error messages reach the terminal, so none may carry ESC, BEL or other
// control characters (OSC 52 would set the user's clipboard).
func TestEAPIResponsesLoseTerminalControlCharacters(t *testing.T) {
	const esc = `\u001b]52;c;ZXZpbA==\u0007\u001b[2J\u009b31m`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case eapiSearchPath:
			fmt.Fprintf(w, `{"success":1,"books":[{"id":1,"hash":"h","title":"Dune%[1]s","author":"Frank%[1]s","year":"1965%[1]s","description":"line one\nline two\ttabbed%[1]s"}]}`, esc)
		case "/eapi/user/profile":
			fmt.Fprintf(w, `{"success":1,"user":{"id":1,"name":"Reader%[1]s","email":"r@example.com","personalDomains":["x%[1]s"]}}`, esc)
		default:
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"success":0,"error":"Daily limit%s"}`, esc)
		}
	}))
	defer server.Close()
	client := NewClient(WithDomain(server.URL))
	client.SetCookies(map[string]string{"remix_userid": "1", "remix_userkey": "key"})

	result, err := client.Search("dune", 1, 10, nil)
	if err != nil || len(result.Books) != 1 {
		t.Fatalf("search = %+v, %v", result, err)
	}
	book := result.Books[0]
	profile, err := client.GetProfile()
	if err != nil {
		t.Fatal(err)
	}
	_, savedErr := client.SavedBooks(1)
	if savedErr == nil {
		t.Fatal("SavedBooks: want the server error")
	}
	for name, got := range map[string]string{
		"title": book.Name, "author": strings.Join(book.Authors, ""), "year": book.Year,
		"description": book.Description, "profile name": profile.Name,
		"personal domain": strings.Join(profile.PersonalDomains, ""), "error": savedErr.Error(),
	} {
		if strings.ContainsAny(got, "\x1b\x07\u009b") {
			t.Errorf("%s still has control characters: %q", name, got)
		}
	}
	if book.Name != "Dune]52;c;ZXZpbA==[2J31m" {
		t.Errorf("title = %q", book.Name)
	}
	if book.Description != "line one\nline two\ttabbed]52;c;ZXZpbA==[2J31m" {
		t.Errorf("description lost its newline or tab: %q", book.Description)
	}
}
