package zlib

import (
	"fmt"
	"net/url"
	"strconv"
)

// SavedBooks returns one page of the account's saved books.
func (c *Client) SavedBooks(page int) (SearchResult, error) {
	if !c.loggedIn {
		return SearchResult{}, ErrNotLoggedIn
	}
	if page < 1 {
		page = 1
	}
	body, err := c.eapiGet("/eapi/user/book/saved", url.Values{
		"page":  {strconv.Itoa(page)},
		"limit": {"50"},
	})
	if err != nil {
		return SearchResult{}, err
	}
	var result struct {
		Success    int            `json:"success"`
		Error      string         `json:"error"`
		Books      []eapiBook     `json:"books"`
		Pagination eapiPagination `json:"pagination"`
	}
	if err := decodeEAPIJSON(body, &result); err != nil {
		return SearchResult{}, fmt.Errorf("zlibrary: invalid EAPI saved books response: %w", err)
	}
	if result.Success != 1 {
		if result.Error == "" {
			result.Error = "saved books request was rejected"
		}
		return SearchResult{}, fmt.Errorf("zlibrary: EAPI saved books failed: %s", result.Error)
	}
	books := make([]Book, 0, len(result.Books))
	for _, item := range result.Books {
		books = append(books, c.mapEAPIBook(item))
	}
	current := result.Pagination.Current
	if current == 0 {
		current = page
	}
	return SearchResult{Books: books, Page: current, TotalPages: result.Pagination.TotalPages}, nil
}
