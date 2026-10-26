package zlib

import "context"

func (c *Client) GetProfile() (UserProfile, error) {
	return c.GetProfileContext(context.Background())
}

// GetProfileContext is GetProfile with cancellation support.
func (c *Client) GetProfileContext(ctx context.Context) (UserProfile, error) {
	if !c.loggedIn {
		return UserProfile{}, ErrNotLoggedIn
	}
	return c.getProfileEAPIContext(ctx)
}

func (c *Client) GetLimits() (DownloadLimit, error) {
	return c.GetLimitsContext(context.Background())
}

// GetLimitsContext is GetLimits with cancellation support.
func (c *Client) GetLimitsContext(ctx context.Context) (DownloadLimit, error) {
	profile, err := c.GetProfileContext(ctx)
	return profile.DownloadLimit, err
}

func (c *Client) DownloadHistory(page int) (DownloadHistoryResult, error) {
	return c.DownloadHistoryContext(context.Background(), page)
}

// DownloadHistoryContext is DownloadHistory with cancellation support.
func (c *Client) DownloadHistoryContext(ctx context.Context, page int) (DownloadHistoryResult, error) {
	if !c.loggedIn {
		return DownloadHistoryResult{}, ErrNotLoggedIn
	}
	if page <= 0 {
		page = 1
	}
	return c.downloadHistoryEAPIContext(ctx, page)
}
