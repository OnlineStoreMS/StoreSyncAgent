package kdzs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const ResultRateLimit = 811

// ErrItemListRateLimit is returned when KDZS rejects /item/v2/list with result 811.
var ErrItemListRateLimit = errors.New("商品列表查询过于频繁，请稍后再试")

const (
	itemListMinInterval = 800 * time.Millisecond
	itemListMaxRetries  = 5
)

type ShopItemSku struct {
	SkuID          string `json:"skuId"`
	PropertiesName string `json:"propertiesName"`
	Price          string `json:"price"`
	Quantity       int    `json:"quantity"`
	OuterID        string `json:"outerId"`
	PicURL         string `json:"picUrl"`
	ScmPicURL      string `json:"scmPicUrl"`
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
	PageNo             int
	PageSize           int
	ShopIDList         []string
	Type               string // onsale / instock
	Title              string
	ShortTitle         string
	ItemIDs            string // numIids
	OuterID            string
	ProductNumLike     string
	SpuPropertiesName  string
	SkuIDs             string
	SkuOuterID         string
	SkuShortTitle      string
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
	PageNo            int      `json:"pageNo"`
	PageSize          int      `json:"pageSize"`
	ShopIDList        []string `json:"shopIdList"`
	Type              string   `json:"type"`
	Title             string   `json:"title"`
	ShortTitle        string   `json:"shortTitle"`
	NumIids           string   `json:"numIids"`
	OuterID           string   `json:"outerId"`
	ProductNumLike    string   `json:"productNumLike"`
	SpuPropertiesName string   `json:"spuPropertiesName"`
	SkuIDs            string   `json:"skuIds"`
	SkuOuterID        string   `json:"skuOuterId"`
	SkuShortTitle     string   `json:"skuShortTitle"`
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
		return nil, ErrItemListRateLimit
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
		PageNo:            pageNo,
		PageSize:          pageSize,
		ShopIDList:        q.ShopIDList,
		Type:              q.Type,
		Title:             q.Title,
		ShortTitle:        q.ShortTitle,
		NumIids:           q.ItemIDs,
		OuterID:           q.OuterID,
		ProductNumLike:    q.ProductNumLike,
		SpuPropertiesName: q.SpuPropertiesName,
		SkuIDs:            q.SkuIDs,
		SkuOuterID:        q.SkuOuterID,
		SkuShortTitle:     q.SkuShortTitle,
	}

	// Serialize item list calls per session to avoid KDZS 811 rate limits.
	s.itemListMu.Lock()
	defer s.itemListMu.Unlock()

	var lastErr error
	for attempt := 0; attempt < itemListMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * time.Second // 2s, 4s, 8s, 16s
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		} else if !s.itemListLastCall.IsZero() {
			wait := itemListMinInterval - time.Since(s.itemListLastCall)
			if wait > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(wait):
				}
			}
		}
		s.itemListLastCall = time.Now()
		var resp itemAPIResponse
		if err := s.client.PostPlatform(ctx, ps, "/item/v2/list", body, &resp); err != nil {
			return nil, err
		}
		result, err := checkItemListResult(&resp)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !errors.Is(err, ErrItemListRateLimit) {
			return nil, err
		}
	}
	return nil, lastErr
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
