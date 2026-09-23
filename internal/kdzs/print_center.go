package kdzs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
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

// printTemplatePlatforms 对齐打单页 mall 参数。模板在各平台站点的
// /print/center/modeListshow/getTemplateList，不在智能网点接口。
var printTemplatePlatforms = []string{
	PlatformManual,
	PlatformDouyin,
	PlatformTaobao,
	"PDD",
	"KSXD",
	PlatformXHS,
	"SPH",
	"JD",
	"ALI1688",
	"TGC",
}

// ListPrintExpressTemplates 拉取打单页实际使用的快递单模板（ModeListShows）。
// 各平台站点可能只返回本电子面单类型，因此按平台分别请求后按模板 ID 合并。
func (s *Session) ListPrintExpressTemplates(ctx context.Context) (items []ExpressTemplate, complete bool, err error) {
	userID := strings.TrimSpace(s.UserID())
	if userID == "" {
		return nil, false, fmt.Errorf("missing user id after login")
	}

	type platformResult struct {
		platform string
		items    []ExpressTemplate
		err      error
	}
	results := make([]platformResult, len(printTemplatePlatforms))
	var wg sync.WaitGroup
	for i, platform := range printTemplatePlatforms {
		wg.Add(1)
		go func(i int, platform string) {
			defer wg.Done()
			items, err := s.listPlatformKddTemplates(ctx, platform, userID)
			results[i] = platformResult{platform: platform, items: items, err: err}
		}(i, platform)
	}
	wg.Wait()

	seen := map[string]struct{}{}
	merged := make([]ExpressTemplate, 0)
	var errs []string
	success := 0
	for _, res := range results {
		if res.err != nil {
			errs = append(errs, res.platform+": "+res.err.Error())
			log.Printf("[kdzs] list print templates %s: %v", res.platform, res.err)
			continue
		}
		success++
		for _, item := range res.items {
			if item.TemplateID == "" {
				continue
			}
			if _, ok := seen[item.TemplateID]; ok {
				continue
			}
			seen[item.TemplateID] = struct{}{}
			merged = append(merged, item)
		}
	}
	if success == 0 {
		return nil, false, fmt.Errorf("拉取打单模板失败: %s", strings.Join(errs, "; "))
	}
	if len(merged) == 0 && len(errs) > 0 {
		return nil, false, fmt.Errorf("未获取到打单模板: %s", strings.Join(errs, "; "))
	}
	// 有平台请求失败时只增改、不按这份不完整结果删除本地模板。
	return merged, len(errs) == 0, nil
}

func (s *Session) listPlatformKddTemplates(ctx context.Context, platform, userID string) ([]ExpressTemplate, error) {
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}
	form := url.Values{
		"modeId":   {"kdd"},
		"exuserId": {userID},
	}
	var resp flexAPIResponse
	if err := s.client.PostPlatformForm(ctx, ps, "/print/center/modeListshow/getTemplateList", form, &resp); err != nil {
		return nil, err
	}
	items, err := parseKddTemplateList(resp)
	if err != nil {
		return nil, err
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
	item, ok := parseKddModeShow(raw)
	if ok {
		return item
	}
	item = ExpressTemplate{Raw: raw}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return item
	}
	item.TemplateID = firstString(m, "templateId", "id", "templateID", "wpOid", "wpoid", "Mode_ListShowId")
	item.TemplateName = firstString(m, "templateName", "name", "templateTitle", "wpName", "wpname", "ExcodeName")
	item.Platform = firstString(m, "platformName", "platform", "plat", "mall")
	if item.Platform == "" {
		item.Platform = printTemplatePlatform(asInt(m["KddType"], m["kddType"]))
	}
	item.CarrierCode = firstString(m, "carrierCode", "cpCode", "ExCode", "exCode", "expressCode")
	item.CarrierName = firstString(m, "carrierName", "cpName", "exName", "expressName")
	item.ShopID = firstString(m, "shopId", "mallUserId", "ownerShopId")
	item.ShopName = firstString(m, "shopName", "mallUserName", "storeName")
	return item
}

func parseKddTemplateList(resp flexAPIResponse) ([]ExpressTemplate, error) {
	code, err := parseFlexibleResult(resp.Result)
	if err != nil {
		return nil, err
	}
	if code != 0 && code != ResultSuccess && code != 101 {
		msg := firstNonEmpty(resp.Message, resp.ErrorMessage, fmt.Sprintf("api error result=%d", code))
		return nil, fmt.Errorf("%s", msg)
	}
	rawItems, err := extractModeListShows(resp.Data)
	if err != nil {
		return nil, err
	}
	out := make([]ExpressTemplate, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := parseKddModeShow(raw)
		if !ok {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func extractModeListShows(data json.RawMessage) ([]json.RawMessage, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, fmt.Errorf("模板接口无数据")
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("decode template data: %w", err)
	}
	raw, ok := probe["ModeListShows"]
	if !ok {
		return nil, fmt.Errorf("模板接口缺少 ModeListShows")
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode ModeListShows: %w", err)
	}
	return items, nil
}

// parseKddModeShow 解析打单页 ModeListShow。普通面单(1)和网点面单(2)在分销代发打单页不展示。
func parseKddModeShow(raw json.RawMessage) (ExpressTemplate, bool) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ExpressTemplate{}, false
	}
	if modeID := firstString(m, "Modeid", "modeId"); modeID != "" && !strings.EqualFold(modeID, "kdd") {
		return ExpressTemplate{}, false
	}
	kddType := asInt(m["KddType"], m["kddType"])
	if kddType == 1 || kddType == 2 {
		return ExpressTemplate{}, false
	}
	id := firstString(m, "Mode_ListShowId", "templateId", "id")
	if kddType == 11 {
		if alt := firstString(m, "Mode_ListShowId_Sting", "Mode_ListShowId_String"); alt != "" {
			id = alt
		}
	}
	name := firstString(m, "ExcodeName", "templateName", "name")
	if id == "" || id == "0" || name == "" {
		return ExpressTemplate{}, false
	}
	return ExpressTemplate{
		TemplateID:   id,
		TemplateName: name,
		Platform:     printTemplatePlatform(kddType),
		CarrierCode:  firstString(m, "ExCode", "exCode", "carrierCode", "cpCode"),
		CarrierName:  firstString(m, "carrierName", "cpName", "exName", "expressName"),
		ShopID:       firstString(m, "shopId", "mallUserId", "ownerShopId"),
		ShopName:     firstString(m, "shopName", "mallUserName", "storeName"),
		Raw:          raw,
	}, true
}

// printTemplatePlatform 把 KddType 映射成发货中心筛选用的平台名。
// 小红书新旧面单都归到「小红书」，与打单时的模板分组一致。
func printTemplatePlatform(kddType int) string {
	switch kddType {
	case 3:
		return "菜鸟"
	case 5:
		return "京东"
	case 7:
		return "拼多多"
	case 8:
		return "抖店"
	case 9:
		return "快手小店"
	case 13, 16:
		return "小红书"
	case 14:
		return "视频号"
	default:
		return ""
	}
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

