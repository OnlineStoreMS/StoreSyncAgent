package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"
)

type tokenCache struct {
	mu      sync.Mutex
	appID   string
	token   string
	expires time.Time
}

func (c *Client) UploadBarcodeImage(ctx context.Context, appID, appSecret, sid string) (string, error) {
	appID = trimSpace(appID)
	appSecret = trimSpace(appSecret)
	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("feishu app id/secret required for barcode image")
	}
	png, err := GenerateCode128PNG(sid)
	if err != nil {
		return "", err
	}
	return c.uploadMessageImage(ctx, appID, appSecret, png)
}

func (c *Client) uploadMessageImage(ctx context.Context, appID, appSecret string, png []byte) (string, error) {
	token, err := c.tenantAccessToken(ctx, appID, appSecret)
	if err != nil {
		return "", err
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("image_type", "message"); err != nil {
		return "", err
	}
	part, err := w.CreateFormFile("image", "barcode.png")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(png); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://open.feishu.cn/open-apis/im/v1/images", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("feishu upload image http %d: %s", resp.StatusCode, trimBody(raw))
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ImageKey string `json:"image_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", fmt.Errorf("feishu upload image: %s", result.Msg)
	}
	if result.Data.ImageKey == "" {
		return "", fmt.Errorf("feishu upload image: empty image_key")
	}
	return result.Data.ImageKey, nil
}

func (c *Client) tenantAccessToken(ctx context.Context, appID, appSecret string) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.tokenCache.appID == appID && c.tokenCache.token != "" && time.Now().Before(c.tokenCache.expires) {
		return c.tokenCache.token, nil
	}
	body, err := json.Marshal(map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.Code != 0 || result.TenantAccessToken == "" {
		return "", fmt.Errorf("feishu tenant token: %s", result.Msg)
	}
	expireSec := result.Expire
	if expireSec <= 120 {
		expireSec = 7200
	}
	c.tokenCache = tokenCache{
		appID:   appID,
		token:   result.TenantAccessToken,
		expires: time.Now().Add(time.Duration(expireSec-60) * time.Second),
	}
	return result.TenantAccessToken, nil
}
