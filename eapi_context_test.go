package zlib

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEAPIPublicContextCancellation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		path   string
		call   func(context.Context, *Client) error
	}{
		{"search", http.MethodPost, eapiSearchPath, func(ctx context.Context, c *Client) error {
			_, err := c.SearchContext(ctx, "book", 1, 10, nil)
			return err
		}},
		{"book", http.MethodGet, "/eapi/book/1/hash", func(ctx context.Context, c *Client) error {
			_, err := c.FetchBookContext(ctx, "1:hash")
			return err
		}},
		{"profile", http.MethodGet, "/eapi/user/profile", func(ctx context.Context, c *Client) error {
			_, err := c.GetProfileContext(ctx)
			return err
		}},
		{"limits", http.MethodGet, "/eapi/user/profile", func(ctx context.Context, c *Client) error {
			_, err := c.GetLimitsContext(ctx)
			return err
		}},
		{"history", http.MethodGet, "/eapi/user/book/downloaded", func(ctx context.Context, c *Client) error {
			_, err := c.DownloadHistoryContext(ctx, 1)
			return err
		}},
		{"recommended", http.MethodGet, recommendedBooksPath, func(ctx context.Context, c *Client) error {
			_, err := c.RecommendedBooksContext(ctx)
			return err
		}},
		{"most-popular", http.MethodGet, mostPopularBooksPath, func(ctx context.Context, c *Client) error {
			_, err := c.MostPopularBooksContext(ctx)
			return err
		}},
		{"saved", http.MethodGet, "/eapi/user/book/saved", func(ctx context.Context, c *Client) error {
			_, err := c.SavedBooksContext(ctx, 1)
			return err
		}},
		{"delete-history", http.MethodGet, "/eapi/user/book/1/delete", func(ctx context.Context, c *Client) error {
			return c.DeleteDownloadHistoryContext(ctx, "1:hash")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			started := make(chan struct{})
			cancelled := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, tc.method, tc.path)
				}
				// Leave the body open to prove cancellation also interrupts response reads.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				close(started)
				select {
				case <-r.Context().Done():
					close(cancelled)
				case <-release:
				}
			}))
			defer server.Close()
			defer close(release)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- tc.call(ctx, newLoggedInEAPITestClient(server.URL)) }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("request did not reach upstream")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want context.Canceled", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("request did not stop after cancellation")
			}
			select {
			case <-cancelled:
			case <-time.After(5 * time.Second):
				t.Fatal("upstream did not observe cancellation")
			}
		})
	}
}

func TestDeleteDownloadHistoryContextPreCanceledNeverCallsTransport(t *testing.T) {
	c := newLoggedInEAPITestClient("https://example.com")
	calls := 0
	c.httpClient.Transport = fakeRoundTripper(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("unexpected upstream request")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.DeleteDownloadHistoryContext(ctx, "42:hash"); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if calls != 0 {
		t.Fatalf("transport calls = %d, want 0", calls)
	}
}

func TestFetchBookDetailsContextStopsQueuedRequests(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprintf("failure=%t", failure), func(t *testing.T) {
			var calls atomic.Int32
			started := make(chan struct{}, detailFetchConcurrency)
			cancelled := make(chan struct{}, detailFetchConcurrency)
			fail := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				select {
				case started <- struct{}{}:
				default:
				}
				if failure && r.URL.Path == "/eapi/book/1/hash" {
					select {
					case <-fail:
						fmt.Fprint(w, `{"success":0,"error":"missing detail"}`)
					case <-release:
					}
					return
				}
				select {
				case <-r.Context().Done():
					select {
					case cancelled <- struct{}{}:
					default:
					}
				case <-release:
				}
			}))
			defer server.Close()
			defer close(release)
			refs := make([]string, 100)
			for i := range refs {
				refs[i] = fmt.Sprintf("%d:hash", i+1)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			type result struct {
				books map[string]Book
				err   error
			}
			done := make(chan result, 1)
			go func() {
				books, err := newLoggedInEAPITestClient(server.URL).FetchBookDetailsContext(ctx, refs)
				done <- result{books, err}
			}()
			for range detailFetchConcurrency {
				select {
				case <-started:
				case <-time.After(5 * time.Second):
					t.Fatal("detail worker did not reach upstream")
				}
			}
			if failure {
				close(fail)
			} else {
				cancel()
			}
			select {
			case got := <-done:
				if failure {
					if got.err == nil || !strings.Contains(got.err.Error(), "missing detail") {
						t.Fatalf("error = %v, want detail failure", got.err)
					}
				} else if !errors.Is(got.err, context.Canceled) {
					t.Fatalf("error = %v, want context.Canceled", got.err)
				}
				if len(got.books) != len(refs) {
					t.Fatalf("map length = %d, want %d", len(got.books), len(refs))
				}
				for _, ref := range refs {
					if book, ok := got.books[ref]; !ok || !reflect.DeepEqual(book, Book{}) {
						t.Errorf("map[%q] = %+v, present = %t, want zero Book", ref, book, ok)
					}
				}
			case <-time.After(5 * time.Second):
				t.Fatal("detail fetching did not stop")
			}
			wantCancelled := detailFetchConcurrency
			if failure {
				wantCancelled--
			}
			for range wantCancelled {
				select {
				case <-cancelled:
				case <-time.After(5 * time.Second):
					t.Fatal("detail request was not cancelled upstream")
				}
			}
			if got := calls.Load(); got != detailFetchConcurrency {
				t.Fatalf("upstream requests = %d, want %d; queued work escaped", got, detailFetchConcurrency)
			}
		})
	}
}

func TestFetchBookDetailsLegacyPoolContinuesAfterFailure(t *testing.T) {
	var mu sync.Mutex
	active, peak, calls := 0, 0, 0
	started := make(chan struct{}, detailFetchConcurrency)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		active++
		calls++
		if active > peak {
			peak = active
		}
		firstWave := calls <= detailFetchConcurrency
		mu.Unlock()
		defer func() {
			mu.Lock()
			active--
			mu.Unlock()
		}()
		if firstWave {
			started <- struct{}{}
			<-release
		}
		if r.URL.Path == "/eapi/book/1/hash" {
			fmt.Fprint(w, `{"success":0,"error":"missing detail"}`)
			return
		}
		fmt.Fprint(w, `{"success":1,"book":{"title":"Found"}}`)
	}))
	defer server.Close()
	refs := make([]string, 20)
	for i := range refs {
		refs[i] = fmt.Sprintf("%d:hash", i+1)
	}
	done := make(chan map[string]Book, 1)
	go func() { done <- newLoggedInEAPITestClient(server.URL).FetchBookDetails(refs) }()
	for range detailFetchConcurrency {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			close(release)
			t.Fatal("legacy detail worker did not reach upstream")
		}
	}
	close(release)
	select {
	case books := <-done:
		if len(books) != len(refs) {
			t.Fatalf("map length = %d, want %d", len(books), len(refs))
		}
		for _, ref := range refs {
			want := "Found"
			if ref == "1:hash" {
				want = ""
			}
			if got := books[ref].Name; got != want {
				t.Errorf("map[%q].Name = %q, want %q", ref, got, want)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("legacy detail fetching did not finish")
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != len(refs) || peak > detailFetchConcurrency {
		t.Errorf("upstream calls = %d, peak = %d; want %d calls, peak <= %d", calls, peak, len(refs), detailFetchConcurrency)
	}
}
