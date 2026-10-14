package zlib

import (
	"fmt"
	"strings"
)

func (c *Client) Search(query string, page, count int, opts *SearchOptions) (SearchResult, error) {
	if !c.loggedIn {
		return SearchResult{}, ErrNotLoggedIn
	}
	if query == "" {
		return SearchResult{}, ErrEmptyQuery
	}
	if count <= 0 {
		count = 10
	}
	if count > 50 {
		count = 50
	}
	if page <= 0 {
		page = 1
	}
	return c.searchEAPI(query, page, count, opts)
}

// FullTextSearch is unavailable because the mobile EAPI has no verified
// equivalent to the website's full-text search.
func (c *Client) FullTextSearch(query string, page, count int, opts *FullTextSearchOptions) (SearchResult, error) {
	if !c.loggedIn {
		return SearchResult{}, ErrNotLoggedIn
	}
	if query == "" {
		return SearchResult{}, ErrEmptyQuery
	}
	if opts != nil && opts.Phrase && len(strings.Fields(query)) < 2 {
		return SearchResult{}, ErrPhraseMinWords
	}
	return SearchResult{}, fmt.Errorf("%w: full-text search has no verified EAPI endpoint", ErrUnsupported)
}
