package kdzs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const defaultShareBaseURL = "https://dfshare.kdzs.com"

// ElecAuthRecord is a normalized electronic waybill authorization row.
type ElecAuthRecord struct {
	Platform    string          `json:"platform,omitempty"`
	ShopName    string          `json:"shopName,omitempty"`
	AccountName string          `json:"accountName,omitempty"`
	AuthStatus  string          `json:"authStatus,omitempty"`
	Raw         json.RawMessage `json:"raw,omitempty"`
}

// ExpressTemplate is a normalized express template row.
type ExpressTemplate struct {
	TemplateID   string          `json:"templateId,omitempty"`
	TemplateName string          `json:"templateName,omitempty"`
	Platform     string          `json:"platform,omitempty"`
	CarrierCode  string          `json:"carrierCode,omitempty"`
	CarrierName  string          `json:"carrierName,omitempty"`
	ShopID       string          `json:"shopId,omitempty"`
	ShopName     string          `json:"shopName,omitempty"`
	Raw          json.RawMessage `json:"raw,omitempty"`
}

// SharedExpressAccount is a normalized shared express account row.
type SharedExpressAccount struct {
	Platform    string          `json:"platform,omitempty"`
	AccountName string          `json:"accountName,omitempty"`
	ShopName    string          `json:"shopName,omitempty"`
	Status      string          `json:"status,omitempty"`
	Raw         json.RawMessage `json:"raw,omitempty"`
}

type flexAPIResponse struct {
	Result       json.RawMessage `json:"result"`
	Message      string          `json:"message"`
	ErrorMessage string          `json:"errorMessage"`
	Data         json.RawMessage `json:"data"`
}

// ListElecAuth returns electronic waybill authorization rows.
func (c *Client) ListElecAuth(ctx context.Context) ([]ElecAuthRecord, error) {
	var resp flexAPIResponse
	if err := c.get(ctx, "/print/center/authorization/manage/getList", &resp); err != nil {
		return nil, err
	}
	items, err := decodeFlexList(resp, parseElecAuthRecord)
	if err != nil {
		return nil, fmt.Errorf("list elec auth: %w", err)
	}
	return items, nil
}

// ListExpressTemplates returns express templates for the given user.
// 注意：快递助手前端把 /intelligent/branch/* 路由到 dfshare.kdzs.com，不是 df.kdzs.com。
func (c *Client) ListExpressTemplates(ctx context.Context, userID string) ([]ExpressTemplate, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("userId is required")
	}
	path := "/intelligent/branch/getTemplateList?" + url.Values{"userId": {userID}}.Encode()
	var resp flexAPIResponse
	if err := c.getShare(ctx, path, &resp); err != nil {
		return nil, err
	}
	items, err := decodeFlexList(resp, parseExpressTemplate)
	if err != nil {
		return nil, fmt.Errorf("list express templates: %w", err)
	}
	return items, nil
}

// ListSharedAccounts queries all shared express accounts from dfshare.kdzs.com.
func (c *Client) ListSharedAccounts(ctx context.Context, body any) ([]SharedExpressAccount, error) {
	if body == nil {
		body = map[string]any{}
	}
	var resp flexAPIResponse
	if err := c.postShare(ctx, "/share/fx/queryAllSharedAccount", body, &resp); err != nil {
		return nil, err
	}
	items, err := decodeFlexList(resp, parseSharedExpressAccount)
	if err != nil {
		return nil, fmt.Errorf("list shared accounts: %w", err)
	}
	return items, nil
}

func (c *Client) postShare(ctx context.Context, path string, body any, out any) error {
	baseURL := defaultShareBaseURL
	reqURL := strings.TrimRight(baseURL, "/") + path
	return c.postAbsolute(ctx, reqURL, body, out)
}

func (c *Client) getShare(ctx context.Context, path string, out any) error {
	reqURL := strings.TrimRight(defaultShareBaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) postAbsolute(ctx context.Context, reqURL string, body any, out any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func decodeFlexList[T any](resp flexAPIResponse, parse func(json.RawMessage) T) ([]T, error) {
	code, err := parseFlexibleResult(resp.Result)
	if err != nil {
		return nil, err
	}
	// 快递助手多数接口成功为 100，部分 print/center 接口为 0
	if code != 0 && code != ResultSuccess && code != 101 {
		msg := firstNonEmpty(resp.Message, resp.ErrorMessage, fmt.Sprintf("api error result=%d", code))
		return nil, fmt.Errorf("%s", msg)
	}
	rawItems, err := unwrapListPayload(resp.Data)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(rawItems))
	for _, raw := range rawItems {
		out = append(out, parse(raw))
	}
	return out, nil
}

func unwrapListPayload(data json.RawMessage) ([]json.RawMessage, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}
	var wrapper struct {
		List  []json.RawMessage `json:"list"`
		Items []json.RawMessage `json:"items"`
		Rows  []json.RawMessage `json:"rows"`
		Data  []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("decode list data: %w", err)
	}
	switch {
	case len(wrapper.List) > 0:
		return wrapper.List, nil
	case len(wrapper.Items) > 0:
		return wrapper.Items, nil
	case len(wrapper.Rows) > 0:
		return wrapper.Rows, nil
	case len(wrapper.Data) > 0:
		return wrapper.Data, nil
	default:
		return nil, nil
	}
}

// elecTypePlatformName 对齐快递助手前端 Ct 表（kddType → platformName）
var elecTypePlatformName = map[int]string{
	3:  "菜鸟",
	5:  "京东",
	7:  "拼多多",
	8:  "抖店",
	9:  "快手小店",
	14: "视频号",
	16: "小红书(新)",
}

func parseElecAuthRecord(raw json.RawMessage) ElecAuthRecord {
	item := ElecAuthRecord{Raw: raw}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return item
	}
	// sygj/bind 页字段：electronicType + platformName（前端用 Ct.kddType 补全）
	item.Platform = firstString(m, "platformName", "platform", "plat", "mall", "mallType")
	if item.Platform == "" {
		if t := asInt(m["electronicType"]); t > 0 {
			item.Platform = elecTypePlatformName[t]
		}
	}
	item.ShopName = firstString(m, "shopName", "mallUserName", "mallShopName", "storeName", "nick", "shopNick")
	item.AccountName = firstString(m, "accountName", "userName", "nickName", "bindAccount", "shareAccount", "account")
	item.AuthStatus = firstString(m, "authStatus", "status", "bindStatus", "authorizationStatus", "authState", "expireDesc")
	if item.AuthStatus == "" {
		if exp := firstString(m, "expireTime", "electAuthExpireTime"); exp != "" {
			item.AuthStatus = exp
		}
	}
	return item
}

func parseExpressTemplate(raw json.RawMessage) ExpressTemplate {
	item := ExpressTemplate{Raw: raw}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return item
	}
	item.TemplateID = firstString(m, "templateId", "id", "templateID", "wpOid", "wpoid")
	item.TemplateName = firstString(m, "templateName", "name", "templateTitle", "wpName", "wpname")
	item.Platform = firstString(m, "platformName", "platform", "plat", "mall")
	item.CarrierCode = firstString(m, "carrierCode", "cpCode", "exCode", "expressCode")
	item.CarrierName = firstString(m, "carrierName", "cpName", "exName", "expressName")
	item.ShopID = firstString(m, "shopId", "mallUserId", "ownerShopId")
	item.ShopName = firstString(m, "shopName", "mallUserName", "storeName")
	return item
}

func parseSharedExpressAccount(raw json.RawMessage) SharedExpressAccount {
	item := SharedExpressAccount{Raw: raw}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return item
	}
	item.Platform = firstString(m, "platform", "plat", "mall")
	item.AccountName = firstString(m, "accountName", "userName", "shareAccountName", "consumerName")
	item.ShopName = firstString(m, "shopName", "mallUserName", "storeName")
	item.Status = firstString(m, "status", "shareStatus", "state")
	return item
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s := stringifyAny(v); s != "" {
				return s
			}
		}
	}
	return ""
}

func stringifyAny(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

