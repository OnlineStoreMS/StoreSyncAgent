package kdzs

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

func (c *Client) GetRedirectURL(ctx context.Context, platform, path string) (string, error) {
	q := url.Values{
		"token":    {c.token},
		"platform": {platform},
		"path":     {path},
	}
	endpoint := "/factory/login/getRedirectUrl?" + q.Encode()
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * 800 * time.Millisecond
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}
		var resp APIResponse[string]
		if err := c.get(ctx, endpoint, &resp); err != nil {
			lastErr = err
			if !isTransientHTTPErr(err) {
				return "", err
			}
			continue
		}
		out, err := checkResult(&resp)
		if err != nil {
			return "", err
		}
		return out, nil
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", errors.New("getRedirectUrl failed")
}

func isTransientHTTPErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "temporary failure") ||
		strings.Contains(msg, "i/o timeout")
}
