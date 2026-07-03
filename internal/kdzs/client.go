package kdzs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// Client talks to 快递助手分销代发 web APIs (df.kdzs.com).
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) Token() string {
	return c.token
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("qnquerystring", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if len(raw) == 0 {
		return fmt.Errorf("empty response from %s", req.URL.String())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w (body=%s)", err, string(raw))
	}
	return nil
}

func (c *Client) postPlatform(ctx context.Context, ps *PlatformSession, path string, body any, out any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}
	url := "https://" + ps.Host + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Origin", "https://"+ps.Host)
	if ps.Referer != "" {
		req.Header.Set("Referer", ps.Referer)
	}
	if ps.Token != "" {
		req.Header.Set("qnquerystring", ps.Token)
	}
	if ch := ps.CookieHeader(); ch != "" {
		req.Header.Set("Cookie", ch)
	}
	return c.do(req, out)
}

func (c *Client) PostPlatform(ctx context.Context, ps *PlatformSession, path string, body any, out any) error {
	return c.postPlatform(ctx, ps, path, body, out)
}

func (c *Client) postPlatformForm(ctx context.Context, ps *PlatformSession, path string, form url.Values, out any) error {
	urlStr := "https://" + ps.Host + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Origin", "https://"+ps.Host)
	if ps.Referer != "" {
		req.Header.Set("Referer", ps.Referer)
	}
	if ps.Token != "" {
		req.Header.Set("qnquerystring", ps.Token)
	}
	if ch := ps.CookieHeader(); ch != "" {
		req.Header.Set("Cookie", ch)
	}
	return c.do(req, out)
}

func checkResult[T any](resp *APIResponse[T]) (T, error) {
	var zero T
	switch resp.Result {
	case ResultSuccess:
		return resp.Data, nil
	case ResultPasswordWrong:
		return zero, fmt.Errorf("login failed: %s", firstNonEmpty(resp.Message, resp.ErrorMessage, "password wrong"))
	case ResultSessionInvalid:
		return zero, fmt.Errorf("session invalid: please login again")
	case ResultTokenEmpty:
		return zero, fmt.Errorf("token missing or expired")
	default:
		msg := firstNonEmpty(resp.Message, resp.ErrorMessage, fmt.Sprintf("api error result=%d", resp.Result))
		return zero, fmt.Errorf("%s", msg)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
