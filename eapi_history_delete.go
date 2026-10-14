package zlib

import (
	"fmt"
	"strings"
)

// DeleteDownloadHistory removes a remote history entry, leaving local files intact.
// ref accepts a numeric book ID or id:hash.
func (c *Client) DeleteDownloadHistory(ref string) error {
	if !c.loggedIn {
		return ErrNotLoggedIn
	}
	id, _, _ := strings.Cut(strings.TrimSpace(ref), ":")
	if id == "" || strings.Trim(id, "0123456789") != "" {
		return fmt.Errorf("zlibrary: history deletion requires a numeric book ID")
	}
	body, err := c.eapiGet("/eapi/user/book/"+id+"/delete", nil)
	if err != nil {
		return err
	}
	var result struct {
		Success int    `json:"success"`
		Error   string `json:"error"`
	}
	if err := decodeEAPIJSON(body, &result); err != nil {
		return fmt.Errorf("zlibrary: invalid history deletion response: %w", err)
	}
	if result.Success != 1 {
		return fmt.Errorf("zlibrary: history deletion rejected: %s", result.Error)
	}
	return nil
}
