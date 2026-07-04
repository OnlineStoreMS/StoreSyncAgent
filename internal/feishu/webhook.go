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
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

type textPayload struct {
	Timestamp string    `json:"timestamp"`
	Sign      string    `json:"sign,omitempty"`
	MsgType   string    `json:"msg_type"`
	Content   textBody  `json:"content"`
}

type textBody struct {
	Text string `json:"text"`
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
	webhookURL = trimSpace(webhookURL)
	if webhookURL == "" {
		return fmt.Errorf("webhook url is required")
	}
	ts := time.Now().Unix()
	payload := textPayload{
		Timestamp: strconv.FormatInt(ts, 10),
		MsgType:   "text",
		Content:   textBody{Text: text},
	}
	if secret = trimSpace(secret); secret != "" {
		sign, err := Sign(secret, ts)
		if err != nil {
			return err
		}
		payload.Sign = sign
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
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
