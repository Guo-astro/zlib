package zlib

func (c *Client) GetProfile() (UserProfile, error) {
	if !c.loggedIn {
		return UserProfile{}, ErrNotLoggedIn
	}
	return c.getProfileEAPI()
}

func (c *Client) GetLimits() (DownloadLimit, error) {
	profile, err := c.GetProfile()
	return profile.DownloadLimit, err
}

func (c *Client) DownloadHistory(page int) (DownloadHistoryResult, error) {
	if !c.loggedIn {
		return DownloadHistoryResult{}, ErrNotLoggedIn
	}
	if page <= 0 {
		page = 1
	}
	return c.downloadHistoryEAPI(page)
}
