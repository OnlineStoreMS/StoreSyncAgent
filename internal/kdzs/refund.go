package kdzs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type RefundQuery struct {
	Platform            string
	ShopID              string
	PageNo              int
	PageSize            int
	StartDateTime       string
	EndDateTime         string
	DateType            int // 4 = 申请时间
	AfterSaleStatusList []string
	AfterSaleTypeList   []int
	Sid                 string
	RefundID            string
	Tid                 string
	SysTid              string
	TradeDaifaStatus    string // ""=全部, "1"=自营售后, "2"=代发售后
}

type RefundListResult struct {
	Total    int          `json:"total"`
	PageNo   int          `json:"pageNo"`
	PageSize int          `json:"pageSize"`
	Items    []RefundItem `json:"items"`
}

type RefundGoods struct {
	Title        string `json:"title,omitempty"`
	SkuName      string `json:"skuName,omitempty"`
	PicURL       string `json:"picUrl,omitempty"`
	Num          int    `json:"num,omitempty"`
	RefundAmount string `json:"refundAmount,omitempty"`
}

type RefundItem struct {
	Platform          string        `json:"platform"`
	PlatformName      string        `json:"platformName"`
	RefundID          string        `json:"refundId"`
	Tid               string        `json:"tid,omitempty"`
	SysTid            string        `json:"sysTid,omitempty"`
	AfterSaleStatus   string        `json:"afterSaleStatus"`
	AfterSaleStatusText string      `json:"afterSaleStatusText"`
	AfterSaleType     int           `json:"afterSaleType"`
	AfterSaleTypeText string        `json:"afterSaleTypeText"`
	RefundReason      string        `json:"refundReason,omitempty"`
	RefundAmount      string        `json:"refundAmount,omitempty"`
	ConfirmTime       string        `json:"confirmTime,omitempty"`
	Created           string        `json:"created,omitempty"`
	BuyerNick         string        `json:"buyerNick,omitempty"`
	ShopName          string        `json:"shopName,omitempty"`
	ShopID            string        `json:"shopId,omitempty"`
	Sid               string        `json:"sid,omitempty"`
	SidCode           string        `json:"sidCode,omitempty"`
	FactoryName       string        `json:"factoryName,omitempty"`
	DaifaStatus       int           `json:"daifaStatus,omitempty"`
	ReviewStatus      int           `json:"reviewStatus,omitempty"`
	OrderSent         bool          `json:"orderSent,omitempty"`
	Goods             []RefundGoods `json:"goods,omitempty"`
	SLA               *RefundSLA    `json:"sla,omitempty"`
}

type refundListResponse struct {
	Result   int               `json:"result"`
	Message  string            `json:"message"`
	Error    string            `json:"error"`
	Data     []json.RawMessage `json:"data"`
	Total    int               `json:"total"`
	PageNo   int               `json:"pageNo"`
	PageSize int               `json:"pageSize"`
}

func ResolveRefundSearchDateRange(start, end string, idSearch bool) (string, string) {
	if start != "" && end != "" {
		return start, end
	}
	now := time.Now()
	endTime := now.Format("2006-01-02 15:04:05")
	if idSearch {
		// Kdzs sidList/refundIds search requires dateType + range; use 90 days for ID lookup.
		startTime := now.AddDate(0, 0, -89).Truncate(24 * time.Hour).Format("2006-01-02 15:04:05")
		return startTime, endTime
	}
	return DefaultDateRange()
}

func splitRefundIDs(raw string) []string {
	raw = strings.ReplaceAll(raw, "，", ",")
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func DefaultRefundDateType() int {
	return 4
}

func TradeDaifaStatusLabel(status string) string {
	switch status {
	case "1":
		return "自营售后"
	case "2":
		return "代发售后"
	default:
		return "全部售后"
	}
}

func RefundDateTypeLabel(dateType int) int {
	if dateType <= 0 {
		return DefaultRefundDateType()
	}
	return dateType
}

func AfterSaleTypeLabel(t int) string {
	switch t {
	case 1:
		return "仅退款"
	case 2:
		return "退货退款"
	case 3:
		return "换货"
	case 4:
		return "补差价"
	case 5:
		return "补发"
	default:
		return fmt.Sprintf("类型%d", t)
	}
}

func AfterSaleStatusLabel(status string) string {
	switch status {
	case "WAIT_SELLER_AGREE":
		return "等待卖家同意"
	case "WAIT_BUYER_RETURN_ITEM":
		return "等待买家退货"
	case "WAIT_SELLER_CONFIRM_RECEIVE":
		return "待卖家确认收货"
	case "REFUND_SUCCESS":
		return "退款成功"
	case "REFUND_CLOSE":
		return "售后关闭"
	case "SELLER_REFUSE", "SELLER_REFUSAL_REFUND":
		return "卖家拒绝"
	case "WAIT_BUYER_CONFIRM":
		return "等待买家确认"
	case "WAIT_BUYER_MODIFY":
		return "待买家修改"
	case "WAIT_SEND_EXCHANGE_ITEM":
		return "待发出换货商品"
	case "WAIT_RECEIVE_EXCHANGE_ITEM":
		return "换货补寄待收货"
	default:
		if status == "" {
			return "—"
		}
		return status
	}
}

func (s *Session) QueryRefunds(ctx context.Context, q RefundQuery) (*RefundListResult, error) {
	if q.PageNo <= 0 {
		q.PageNo = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	platform := strings.ToUpper(q.Platform)
	if platform == "" {
		platform = "FXG"
	}

	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}

	body, err := s.buildRefundListRequest(ctx, q)
	if err != nil {
		return nil, err
	}

	var resp refundListResponse
	if err := s.client.postPlatform(ctx, ps, "/refund/fxdf/queryRefund", body, &resp); err != nil {
		return nil, err
	}
	if resp.Result != 0 && resp.Result != ResultSuccess {
		return nil, fmt.Errorf("%s", firstNonEmpty(resp.Message, resp.Error, "query refunds failed"))
	}

	items := make([]RefundItem, 0, len(resp.Data))
	for _, raw := range resp.Data {
		item := parseRefundItem(raw, platform)
		if item != nil {
			items = append(items, *item)
		}
	}

	return &RefundListResult{
		Total:    resp.Total,
		PageNo:   resp.PageNo,
		PageSize: resp.PageSize,
		Items:    items,
	}, nil
}

// QueryAllRefunds paginates through the API until all matching items are loaded.
func (s *Session) QueryAllRefunds(ctx context.Context, q RefundQuery) ([]RefundItem, int, error) {
	q.PageNo = 1
	q.PageSize = 100
	var all []RefundItem
	total := 0
	for {
		res, err := s.QueryRefunds(ctx, q)
		if err != nil {
			return nil, 0, err
		}
		total = res.Total
		all = append(all, res.Items...)
		if len(all) >= total || len(res.Items) == 0 {
			break
		}
		q.PageNo++
	}
	return all, total, nil
}

func (s *Session) buildRefundListRequest(ctx context.Context, q RefundQuery) (map[string]any, error) {
	shopIDs, err := s.resolveRefundShopIDs(ctx, q.Platform, q.ShopID)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"platform":         strings.ToUpper(q.Platform),
		"roleSource":       "SELLER",
		"pageNo":           q.PageNo,
		"pageSize":         q.PageSize,
		"shopIds":          shopIDs,
		"tradeDaifaStatus": strings.TrimSpace(q.TradeDaifaStatus),
	}

	hasIDSearch := q.Sid != "" || q.RefundID != "" || q.Tid != "" || q.SysTid != ""
	start, end := ResolveRefundSearchDateRange(q.StartDateTime, q.EndDateTime, hasIDSearch)
	body["dateType"] = RefundDateTypeLabel(q.DateType)
	body["startDateTime"] = start
	body["endDateTime"] = end

	if hasIDSearch {
		if ids := splitRefundIDs(q.Sid); len(ids) > 0 {
			body["sidList"] = ids
		}
		if ids := splitRefundIDs(q.RefundID); len(ids) > 0 {
			body["refundIds"] = ids
		}
		if ids := splitRefundIDs(q.Tid); len(ids) > 0 {
			body["tids"] = ids
		}
		if ids := splitRefundIDs(q.SysTid); len(ids) > 0 {
			body["systemTids"] = ids
		}
		return body, nil
	}

	if len(q.AfterSaleStatusList) > 0 {
		body["afterSaleStatusList"] = q.AfterSaleStatusList
	}
	if len(q.AfterSaleTypeList) > 0 {
		body["afterSaleTypeList"] = q.AfterSaleTypeList
	}
	return body, nil
}

func (s *Session) resolveRefundShopIDs(ctx context.Context, platform, shopID string) ([]string, error) {
	if shopID != "" {
		return []string{shopID}, nil
	}
	shops, err := s.client.ListEcommerceShops(ctx)
	if err != nil {
		return nil, err
	}
	platform = strings.ToUpper(platform)
	ids := make([]string, 0)
	for _, shop := range shops {
		if shop.Platform == platform && shop.MallUserID != "" {
			ids = append(ids, shop.MallUserID)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no shops for platform %s", platform)
	}
	return ids, nil
}

func parseRefundItem(raw json.RawMessage, platform string) *RefundItem {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	item := &RefundItem{
		Platform:            platform,
		PlatformName:        PlatformLabel(platform),
		RefundID:            strVal(m["refundId"]),
		Tid:                 strVal(m["tid"]),
		SysTid:              strVal(m["sysTid"]),
		AfterSaleStatus:     strVal(m["afterSaleStatus"]),
		AfterSaleStatusText: AfterSaleStatusLabel(strVal(m["afterSaleStatus"])),
		AfterSaleType:       intVal(m["afterSaleType"]),
		AfterSaleTypeText:   AfterSaleTypeLabel(intVal(m["afterSaleType"])),
		RefundReason:        strVal(m["refundReason"]),
		RefundAmount:        strVal(m["refundAmount"]),
		ConfirmTime:         strVal(m["confirmTime"]),
		Created:             strVal(m["created"]),
		BuyerNick:           strVal(m["buyerNick"]),
		ShopName:            strVal(m["shopName"]),
		ShopID:              strVal(m["ownerShopId"]),
		Sid:                 strVal(m["sid"]),
		SidCode:             strVal(m["sidCode"]),
		FactoryName:         strVal(m["factoryName"]),
		DaifaStatus:         intVal(m["daifaStatus"]),
		ReviewStatus:        intVal(m["reviewStatus"]),
		OrderSent:           boolVal(m["orderSent"]),
	}
	if details, ok := m["orderDetails"].([]any); ok {
		for _, d := range details {
			dm, _ := d.(map[string]any)
			if dm == nil {
				continue
			}
			item.Goods = append(item.Goods, RefundGoods{
				Title:        strVal(dm["title"]),
				SkuName:      strVal(dm["skuName"]),
				PicURL:       firstNonEmpty(strVal(dm["picUrl"]), strVal(dm["picPath"])),
				Num:          intVal(dm["num"]),
				RefundAmount: strVal(dm["refundAmount"]),
			})
		}
	}
	return item
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

func intVal(v any) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		return 0
	}
}

func boolVal(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	default:
		return fmt.Sprint(v) == "true"
	}
}
