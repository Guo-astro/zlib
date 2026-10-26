package zlib

import (
	"context"
	"fmt"
	"sync"
)

const detailFetchConcurrency = 4

// FetchBookDetails resolves book references concurrently. Failed references
// retain a zero Book entry so callers can match results to their inputs.
func (c *Client) FetchBookDetails(refs []string) map[string]Book {
	out, _ := c.fetchBookDetails(context.Background(), refs, false)
	return out
}

// FetchBookDetailsContext resolves references with bounded concurrency. On the
// first failure it cancels outstanding requests and returns a partial map and
// the error. Failed or unprocessed references retain a zero Book entry.
func (c *Client) FetchBookDetailsContext(ctx context.Context, refs []string) (map[string]Book, error) {
	return c.fetchBookDetails(ctx, refs, true)
}

func (c *Client) fetchBookDetails(ctx context.Context, refs []string, failFast bool) (map[string]Book, error) {
	out := make(map[string]Book, len(refs))
	for _, ref := range refs {
		out[ref] = Book{}
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup
	jobs := make(chan string)
	for range min(detailFetchConcurrency, len(refs)) {
		wg.Go(func() {
			for ref := range jobs {
				if workCtx.Err() != nil {
					return
				}
				book, err := c.FetchBookContext(workCtx, ref)
				mu.Lock()
				out[ref] = book
				if err != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("zlibrary: fetch book %q: %w", ref, err)
					}
					if failFast {
						cancel()
					}
				}
				mu.Unlock()
			}
		})
	}
queue:
	for _, ref := range refs {
		select {
		case <-workCtx.Done():
			break queue
		case jobs <- ref:
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return out, firstErr
	}
	return out, ctx.Err()
}

func (c *Client) FetchBook(ref string) (Book, error) {
	return c.FetchBookContext(context.Background(), ref)
}

// FetchBookContext is FetchBook with cancellation support.
func (c *Client) FetchBookContext(ctx context.Context, ref string) (Book, error) {
	if !c.loggedIn {
		return Book{}, ErrNotLoggedIn
	}
	if ref == "" {
		return Book{}, ErrNoID
	}
	return c.fetchBookEAPIContext(ctx, ref)
}
