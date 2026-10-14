package zlib

import "sync"

const detailFetchConcurrency = 4

// FetchBookDetails resolves book references concurrently. Failed references
// retain a zero Book entry so callers can match results to their inputs.
func (c *Client) FetchBookDetails(refs []string) map[string]Book {
	out := make(map[string]Book, len(refs))
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, detailFetchConcurrency)
	for _, ref := range refs {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			book, _ := c.FetchBook(ref)
			mu.Lock()
			out[ref] = book
			mu.Unlock()
		})
	}
	wg.Wait()
	return out
}

func (c *Client) FetchBook(ref string) (Book, error) {
	if !c.loggedIn {
		return Book{}, ErrNotLoggedIn
	}
	if ref == "" {
		return Book{}, ErrNoID
	}
	return c.fetchBookEAPI(ref)
}
