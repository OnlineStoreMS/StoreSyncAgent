package kdzs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const ResultRateLimit = 811

type ShopItemSku struct {
	SkuID          string `json:"skuId"`
	PropertiesName string `json:"propertiesName"`
	Price          string `json:"price"`
	Quantity       int    `json:"quantity"`
	OuterID        string `json:"outerId"`
	PicURL         string `json:"picUrl"`
	ShortTitle     string `json:"shortTitle"`
	Status         string `json:"status"`
	ProductNum     string `json:"productNum"`
}

type ShopItem struct {
	NumIid        string        `json:"numIid"`
	Title         string        `json:"title"`
	ShortTitle    string        `json:"shortTitle"`
	OuterID       string        `json:"outerId"`
	PicURL        string        `json:"picUrl"`
	Price         string        `json:"price"`
	Num           int           `json:"num"`
	Platform      string        `json:"platform"`
	ShopID        string        `json:"shopId"`
	ShopName      string        `json:"shopName"`
	ApproveStatus string        `json:"approveStatus"`
	ProductNum    string        `json:"productNum"`
	BindTime      string        `json:"bindTime"`
	Skus          []ShopItemSku `json:"skus"`
}

type ItemListQuery struct {
	PageNo     int
	PageSize   int
	ShopIDList []string
	Title      string
	ItemIDs    string
	OuterID    string
}

type ItemListResult struct {
	Count    int        `json:"count"`
	List     []ShopItem `json:"list"`
	PageNo   int        `json:"pageNo"`
	PageSize int        `json:"pageSize"`
	Success  bool       `json:"success"`
}

type SyncProgress struct {
	Finish         bool           `json:"finish"`
	Process        int            `json:"process"`
	SyncItemCount  map[string]int `json:"syncItemCount"`
	ErrorMessage   string         `json:"errorMessage"`
	FinishSyncDate string         `json:"finishSyncDate"`
}

type itemAPIResponse struct {
	Result       json.RawMessage `json:"result"`
	Message      string          `json:"message"`
	ErrorMessage string          `json:"errorMessage"`
	Data         json.RawMessage `json:"data"`
}

type itemListRequest struct {
	PageNo     int      `json:"pageNo"`
	PageSize   int      `json:"pageSize"`
	ShopIDList []string `json:"shopIdList"`
	Title      string   `json:"title"`
	ItemIDs    string   `json:"itemIds"`
	OuterID    string   `json:"outerId"`
}

type syncItemsRequest struct {
	ShopIDs []string `json:"shopIds"`
}

func parseFlexibleResult(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("empty result")
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		code, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return 0, fmt.Errorf("unexpected result: %s", s)
		}
		return code, nil
	}
	return 0, fmt.Errorf("unexpected result type: %s", string(raw))
}

func checkItemListResult(resp *itemAPIResponse) (*ItemListResult, error) {
	code, err := parseFlexibleResult(resp.Result)
	if err != nil {
		return nil, err
	}
	switch code {
	case ResultSuccess:
		var data ItemListResult
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return nil, fmt.Errorf("decode item list data: %w", err)
		}
		return &data, nil
	case ResultRateLimit:
		return nil, fmt.Errorf("商品列表查询过于频繁，请稍后再试")
	default:
		msg := firstNonEmpty(resp.Message, resp.ErrorMessage, fmt.Sprintf("api error result=%d", code))
		return nil, fmt.Errorf("%s", msg)
	}
}

func checkItemSyncResult(resp *itemAPIResponse) error {
	code, err := parseFlexibleResult(resp.Result)
	if err != nil {
		return err
	}
	if code == ResultSuccess {
		return nil
	}
	msg := firstNonEmpty(resp.Message, resp.ErrorMessage)
	if code == 300 && (strings.Contains(msg, "正在同步") || strings.Contains(msg, "耐心等待")) {
		return nil
	}
	if msg == "" {
		msg = fmt.Sprintf("api error result=%d", code)
	}
	return fmt.Errorf("%s", msg)
}

func checkItemProgressResult(resp *itemAPIResponse) (*SyncProgress, error) {
	code, err := parseFlexibleResult(resp.Result)
	if err != nil {
		return nil, err
	}
	if code != ResultSuccess {
		msg := firstNonEmpty(resp.Message, resp.ErrorMessage, fmt.Sprintf("api error result=%d", code))
		return nil, fmt.Errorf("%s", msg)
	}
	var data SyncProgress
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("decode sync progress: %w", err)
	}
	return &data, nil
}

func ItemApproveStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "onsale", "on_sale":
		return "在售"
	case "instock", "in_stock":
		return "仓库中"
	case "soldout", "sold_out":
		return "售罄"
	case "delete", "deleted":
		return "已删除"
	default:
		if looksLikeChineseLabel(status) {
			return status
		}
		return status
	}
}

func (s *Session) ListShopItems(ctx context.Context, platform string, q ItemListQuery) (*ItemListResult, error) {
	if len(q.ShopIDList) == 0 {
		return nil, fmt.Errorf("shopIdList is required")
	}
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}
	pageNo := q.PageNo
	if pageNo <= 0 {
		pageNo = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	body := itemListRequest{
		PageNo:     pageNo,
		PageSize:   pageSize,
		ShopIDList: q.ShopIDList,
		Title:      q.Title,
		ItemIDs:    q.ItemIDs,
		OuterID:    q.OuterID,
	}
	var resp itemAPIResponse
	if err := s.client.PostPlatform(ctx, ps, "/item/v2/list", body, &resp); err != nil {
		return nil, err
	}
	return checkItemListResult(&resp)
}

func (s *Session) SyncShopItems(ctx context.Context, platform string, shopIDs []string) error {
	if len(shopIDs) == 0 {
		return fmt.Errorf("shopIds is required")
	}
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return err
	}
	body := syncItemsRequest{ShopIDs: shopIDs}
	var resp itemAPIResponse
	if err := s.client.PostPlatform(ctx, ps, "/item/v2/asyncRefreshItemV2", body, &resp); err != nil {
		return err
	}
	return checkItemSyncResult(&resp)
}

func (s *Session) GetItemSyncProgress(ctx context.Context, platform string) (*SyncProgress, error) {
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}
	var resp itemAPIResponse
	if err := s.client.GetPlatform(ctx, ps, "/item/v2/getAsyncProgress", &resp); err != nil {
		return nil, err
	}
	return checkItemProgressResult(&resp)
}
