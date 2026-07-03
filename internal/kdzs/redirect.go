package kdzs

import (
	"context"
	"net/url"
)

func (c *Client) GetRedirectURL(ctx context.Context, platform, path string) (string, error) {
	q := url.Values{
		"token":    {c.token},
		"platform": {platform},
		"path":     {path},
	}
	var resp APIResponse[string]
	if err := c.get(ctx, "/factory/login/getRedirectUrl?"+q.Encode(), &resp); err != nil {
		return "", err
	}
	return checkResult(&resp)
}
