package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	httpClient *http.Client
	tokenMu    sync.Mutex
	tokenCache tokenCache
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func Sign(secret string, timestamp int64) (string, error) {
	if secret == "" {
		return "", nil
	}
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	mac := hmac.New(sha256.New, []byte(stringToSign))
	if _, err := mac.Write(nil); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (c *Client) SendText(ctx context.Context, webhookURL, secret, text string) error {
	return c.postSignedJSON(ctx, webhookURL, secret, map[string]any{
		"msg_type": "text",
		"content": map[string]any{
			"text": text,
		},
	})
}

func signPayload(secret string) (timestamp, sign string, err error) {
	ts := time.Now().Unix()
	if secret = trimSpace(secret); secret == "" {
		return strconv.FormatInt(ts, 10), "", nil
	}
	s, err := Sign(secret, ts)
	if err != nil {
		return "", "", err
	}
	return strconv.FormatInt(ts, 10), s, nil
}

func (c *Client) postSignedJSON(ctx context.Context, webhookURL, secret string, payload map[string]any) error {
	webhookURL = trimSpace(webhookURL)
	if webhookURL == "" {
		return fmt.Errorf("webhook url is required")
	}
	ts, sign, err := signPayload(secret)
	if err != nil {
		return err
	}
	payload["timestamp"] = ts
	if sign != "" {
		payload["sign"] = sign
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.doPost(ctx, webhookURL, body)
}

func (c *Client) doPost(ctx context.Context, webhookURL string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("feishu webhook http %d: %s", resp.StatusCode, trimBody(raw))
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &result)
	}
	if result.Code != 0 && result.Msg != "" {
		return fmt.Errorf("feishu webhook: %s", result.Msg)
	}
	return nil
}

func trimSpace(s string) string {
	return string(bytesTrimSpace([]byte(s)))
}

func bytesTrimSpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t' || b[0] == '\n' || b[0] == '\r') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t' || b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

func trimBody(raw []byte) string {
	if len(raw) > 200 {
		return string(raw[:200]) + "..."
	}
	return string(raw)
}
