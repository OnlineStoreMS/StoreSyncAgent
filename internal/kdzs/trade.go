package kdzs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type TradeQuery struct {
	Platform      string
	TradeStatus   string
	PageNo        int
	PageSize      int
	ShopID        string
	TimeType      int
	StartDateTime string
	EndDateTime   string
	Tid           string
}

type TradeListResult struct {
	Total    int             `json:"total"`
	PageNo   int             `json:"pageNo"`
	PageSize int             `json:"pageSize"`
	Items    []TradeListItem `json:"items"`
}

type TradeListItem struct {
	Platform        string         `json:"platform"`
	PlatformName    string         `json:"platformName"`
	TogetherID      string         `json:"togetherId,omitempty"`
	SysTids         []string       `json:"sysTids,omitempty"`
	Tids            []string       `json:"tids,omitempty"`
	BuyerNick       string         `json:"buyerNick,omitempty"`
	ReceiverName    string         `json:"receiverName,omitempty"`
	ReceiverMobile  string         `json:"receiverMobile,omitempty"`
	ReceiverAddress string         `json:"receiverAddress,omitempty"`
	Payment         float64        `json:"payment,omitempty"`
	TradeStatus     string         `json:"tradeStatus,omitempty"` // 快递助手列表态：wait_audit/wait_send
	StatusText      string         `json:"statusText,omitempty"`  // 快递助手列表态文案
	PlatformOrderStatus     string `json:"platformOrderStatus,omitempty"`     // 电商平台订单状态码
	PlatformOrderStatusText string `json:"platformOrderStatusText,omitempty"` // 电商平台订单状态文案
	AfterSaleStatus         string `json:"afterSaleStatus,omitempty"`
	AfterSaleStatusText     string `json:"afterSaleStatusText,omitempty"`
	CreateTime      string         `json:"createTime,omitempty"`
	PayTime         string         `json:"payTime,omitempty"`
	ShopName        string         `json:"shopName,omitempty"`
	ShopID          string         `json:"shopId,omitempty"`
	Goods           []TradeGoods   `json:"goods,omitempty"`
	BuyerMemo       string         `json:"buyerMemo,omitempty"`
	SellerMemo      string         `json:"sellerMemo,omitempty"`
	FenFaMemo       string         `json:"fenFaMemo,omitempty"`
	PrinterMemo     string         `json:"printerMemo,omitempty"`
	AgentType       int            `json:"agentType,omitempty"`
	FactoryID       string         `json:"factoryId,omitempty"`
	FactoryName     string         `json:"factoryName,omitempty"`
	Decrypted       bool           `json:"decrypted,omitempty"`
	FormattedReceiver string       `json:"formattedReceiver,omitempty"`
	DecryptMeta     *TradeDecryptMeta `json:"-"`
}

type TradeGoods struct {
	Title   string  `json:"title,omitempty"`
	SkuName string  `json:"skuName,omitempty"`
	PicURL  string  `json:"picUrl,omitempty"`
	Num     int     `json:"num,omitempty"`
	OuterID string  `json:"outerId,omitempty"`
	Price   float64 `json:"price,omitempty"`
}

type tradeListRequest struct {
	RDSUser         bool     `json:"rdsUser"`
	AsyncCode       string   `json:"asyncCode"`
	Platform        string   `json:"platform"`
	TradeStatus     string   `json:"tradeStatus"`
	Status          string   `json:"status"`
	PageNo          int      `json:"pageNo"`
	PageSize        int      `json:"pageSize"`
	UserID          int64    `json:"userId,omitempty"`
	ShopIDs         []string `json:"shopIds,omitempty"`
	FactoryIDs      []string `json:"factoryIds,omitempty"`
	DistributorIDs  []string `json:"distributorIds,omitempty"`
	Tids            []string `json:"tids,omitempty"`
	StartDateTime   string   `json:"startDateTime,omitempty"`
	EndDateTime     string   `json:"endDateTime,omitempty"`
	TimeType        int      `json:"timeType,omitempty"`
	ShowDaifaTrade  int      `json:"showDaifaTrade,omitempty"`
	IsFXGDaifa      int      `json:"isFXGDaifa,omitempty"`
	FxgDaifaPage    bool     `json:"fxgDaifaPage,omitempty"`
}

type tradeListResponse struct {
	Result       int               `json:"result"`
	Message      string            `json:"message"`
	ErrorMessage string            `json:"errorMessage"`
	Data         []json.RawMessage `json:"data"`
	Total        int               `json:"total"`
	PageNo       int               `json:"pageNo"`
	PageSize     int               `json:"pageSize"`
}

func defaultDateRange() (start, end string) {
	return DefaultDateRange()
}

func DefaultDateRange() (start, end string) {
	now := time.Now()
	end = now.Format("2006-01-02 15:04:05")
	// Kdzs allPack default filter uses last 30 days.
	start = now.AddDate(0, 0, -29).Truncate(24 * time.Hour).Format("2006-01-02 15:04:05")
	return start, end
}

func ResolveDateRange(start, end string) (string, string) {
	if start != "" && end != "" {
		return start, end
	}
	return DefaultDateRange()
}

func ResolveTradeSearchDateRange(start, end string) (string, string) {
	if start != "" && end != "" {
		return start, end
	}
	now := time.Now()
	endTime := now.Format("2006-01-02 15:04:05")
	startTime := now.AddDate(-1, 0, 0).Truncate(24 * time.Hour).Format("2006-01-02 15:04:05")
	return startTime, endTime
}

func DefaultTimeType() int {
	return 0
}

func TimeTypeLabel(timeType int) string {
	if timeType == 1 {
		return "发货时间"
	}
	return "下单时间"
}

func (s *Session) QueryTrades(ctx context.Context, q TradeQuery) (*TradeListResult, error) {
	if q.PageNo <= 0 {
		q.PageNo = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	idSearch := strings.TrimSpace(q.Tid) != ""
	if q.TradeStatus == "" && !idSearch {
		q.TradeStatus = DefaultTradeStatus()
	}

	ps, err := s.PlatformSession(ctx, q.Platform)
	if err != nil {
		return nil, err
	}

	body, err := s.buildTradeListRequest(ctx, q)
	if err != nil {
		return nil, err
	}

	var resp tradeListResponse
	if err := s.client.postPlatform(ctx, ps, "/tradeManage/queryRdsTradeList", body, &resp); err != nil {
		return nil, err
	}
	if resp.Result != 0 && resp.Result != ResultSuccess {
		return nil, fmt.Errorf("%s", firstNonEmpty(resp.Message, resp.ErrorMessage, "query trades failed"))
	}

	items := make([]TradeListItem, 0, len(resp.Data))
	sysTids := make([]string, 0, len(resp.Data))
	for _, raw := range resp.Data {
		item := parseTradeItem(raw, q.Platform)
		if item != nil {
			items = append(items, *item)
			for _, sid := range item.SysTids {
				sysTids = appendUnique(sysTids, sid)
			}
		}
	}

	if len(sysTids) > 0 {
		if enriched, err := s.enrichTradeItems(ctx, q.Platform, q.TradeStatus, sysTids, items); err == nil {
			items = enriched
		}
	}

	finalizeTradeListItems(items, q.TradeStatus)

	return &TradeListResult{
		Total:    resp.Total,
		PageNo:   resp.PageNo,
		PageSize: resp.PageSize,
		Items:    items,
	}, nil
}

func (s *Session) QueryTradesRaw(ctx context.Context, q TradeQuery) (map[string]any, error) {
	if q.PageNo <= 0 {
		q.PageNo = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.TradeStatus == "" {
		q.TradeStatus = DefaultTradeStatus()
	}

	ps, err := s.PlatformSession(ctx, q.Platform)
	if err != nil {
		return nil, err
	}

	body, err := s.buildTradeListRequest(ctx, q)
	if err != nil {
		return nil, err
	}

	var resp map[string]any
	if err := s.client.postPlatform(ctx, ps, "/tradeManage/queryRdsTradeList", body, &resp); err != nil {
		return nil, err
	}
	resp["_request"] = body
	return resp, nil
}

func (s *Session) buildTradeListRequest(ctx context.Context, q TradeQuery) (tradeListRequest, error) {
	userID, _ := strconv.ParseInt(s.UserID(), 10, 64)
	start, end := ResolveDateRange(q.StartDateTime, q.EndDateTime)
	if strings.TrimSpace(q.Tid) != "" {
		start, end = ResolveTradeSearchDateRange(q.StartDateTime, q.EndDateTime)
	}
	timeType := q.TimeType

	body := tradeListRequest{
		RDSUser:       true,
		AsyncCode:     "",
		Platform:      q.Platform,
		TradeStatus:   q.TradeStatus,
		Status:        TradeStatusToAPIStatus(q.TradeStatus),
		PageNo:        q.PageNo,
		PageSize:      q.PageSize,
		UserID:        userID,
		StartDateTime: start,
		EndDateTime:   end,
		TimeType:      timeType,
	}
	if tid := strings.TrimSpace(q.Tid); tid != "" {
		body.Tids = []string{tid}
	}
	if q.ShopID != "" {
		body.ShopIDs = []string{q.ShopID}
	} else if shopIDs, err := s.platformShopIDs(ctx, q.Platform); err != nil {
		return body, err
	} else if len(shopIDs) > 0 {
		body.ShopIDs = shopIDs
	}
	if q.Platform == PlatformDouyin {
		body.ShowDaifaTrade = 1
		body.IsFXGDaifa = 1
		body.FxgDaifaPage = true
	}
	return body, nil
}

func (s *Session) platformShopIDs(ctx context.Context, platform string) ([]string, error) {
	shops, err := s.client.ListEcommerceShops(ctx)
	if err != nil {
		return nil, err
	}
	platform = strings.ToUpper(platform)
	out := make([]string, 0)
	for _, shop := range shops {
		if strings.ToUpper(shop.Platform) == platform && shop.MallUserID != "" {
			out = append(out, shop.MallUserID)
		}
	}
	return out, nil
}

func applyListStatusText(items []TradeListItem, listTradeStatus string) {
	finalizeTradeListItems(items, listTradeStatus)
}

// finalizeTradeListItems 分离「快递助手列表态」与「电商平台订单状态」，避免互相覆盖。
func finalizeTradeListItems(items []TradeListItem, listTradeStatus string) {
	for i := range items {
		normalizeAgentType(&items[i], 0, 0)
		preservePlatformOrderStatus(&items[i])
		if items[i].AfterSaleStatus != "" && items[i].AfterSaleStatusText == "" {
			items[i].AfterSaleStatusText = AfterSaleStatusLabel(items[i].AfterSaleStatus)
		}
		listStatus := strings.TrimSpace(listTradeStatus)
		if listStatus == "" || strings.EqualFold(listStatus, "all") {
			// tid 回查等无列表筛选项：用文案/已有字段推断快递助手列表态
			if inferred := InferKDZSListStatus(items[i].TradeStatus, items[i].StatusText); inferred != "" {
				items[i].TradeStatus = inferred
				items[i].StatusText = TradeStatusLabel(inferred)
			}
			continue
		}
		items[i].TradeStatus = listStatus
		items[i].StatusText = TradeStatusLabel(listStatus)
	}
}

func preservePlatformOrderStatus(item *TradeListItem) {
	if item == nil {
		return
	}
	if item.PlatformOrderStatus == "" {
		cand := item.TradeStatus
		if cand != "" && !IsKDZSListStatus(cand) {
			item.PlatformOrderStatus = cand
		}
	}
	if item.PlatformOrderStatusText == "" {
		// StatusText 在覆盖前可能已是平台中文描述
		st := strings.TrimSpace(item.StatusText)
		if st != "" && st != TradeStatusLabel("wait_audit") && st != TradeStatusLabel("wait_send") &&
			st != TradeStatusLabel("shipped") && st != TradeStatusLabel("completed") {
			item.PlatformOrderStatusText = ResolveOrderStatusText(st, item.PlatformOrderStatus)
		} else {
			item.PlatformOrderStatusText = ResolveOrderStatusText("", item.PlatformOrderStatus)
		}
	}
}

// InferKDZSListStatus 从文案或混入的电商状态推断快递助手列表态。
func InferKDZSListStatus(tradeStatus, statusText string) string {
	if IsKDZSListStatus(tradeStatus) {
		return strings.ToLower(strings.TrimSpace(tradeStatus))
	}
	st := strings.TrimSpace(statusText)
	switch st {
	case "待推单":
		return "wait_audit"
	case "待发货":
		return "wait_send"
	case "已发货":
		return "shipped"
	case "交易完成", "已完成":
		return "completed"
	}
	return ""
}

func IsKDZSListStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "wait_audit", "wait_send", "shipped", "completed", "all":
		return true
	default:
		return false
	}
}

func (s *Session) enrichTradeItems(ctx context.Context, platform, tradeStatus string, sysTids []string, items []TradeListItem) ([]TradeListItem, error) {
	pkgs, err := s.FetchTradeDetails(ctx, platform, tradeStatus, sysTids)
	if err != nil {
		return items, err
	}
	bySysTid := make(map[string]TradeListItem, len(pkgs))
	for _, raw := range pkgs {
		item := parseTradeItem(raw, platform)
		if item == nil {
			continue
		}
		normalizeAgentType(item, 0, 0)
		for _, sid := range item.SysTids {
			bySysTid[sid] = *item
		}
	}
	for i := range items {
		for _, sid := range items[i].SysTids {
			if detail, ok := bySysTid[sid]; ok {
				items[i] = mergeTradeListItem(items[i], detail)
				break
			}
		}
	}
	return items, nil
}

// mergeTradeListItem 用详情补全列表项，保留列表侧已有的代发/厂家信息。
func mergeTradeListItem(base, detail TradeListItem) TradeListItem {
	out := detail
	if out.AgentType == 0 {
		out.AgentType = base.AgentType
	}
	if out.FactoryID == "" {
		out.FactoryID = base.FactoryID
	}
	if out.FactoryName == "" {
		out.FactoryName = base.FactoryName
	}
	if len(out.Tids) == 0 {
		out.Tids = base.Tids
	}
	if len(out.SysTids) == 0 {
		out.SysTids = base.SysTids
	}
	if out.ShopID == "" {
		out.ShopID = base.ShopID
	}
	if out.ShopName == "" {
		out.ShopName = base.ShopName
	}
	normalizeAgentType(&out, 0, 0)
	return out
}

func parseTradeItem(raw json.RawMessage, platform string) *TradeListItem {
	var pkg map[string]any
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil
	}

	item := &TradeListItem{
		Platform:     platform,
		PlatformName: PlatformLabel(platform),
		TogetherID:   asString(pkg["togetherId"]),
		BuyerNick:    asString(pkg["buyerNick"]),
	}

	trades, _ := pkg["trades"].([]any)
	if len(trades) > 0 {
		return parseTradeItemLegacyTrades(item, trades)
	}

	if orderDetails, ok := pkg["orderDetails"].([]any); ok && len(orderDetails) > 0 {
		meta := ParseDecryptMeta(pkg)
		item.DecryptMeta = &meta
		flattenTradeMap(item, pkg)
		if sysTid := asString(pkg["sysTid"]); sysTid != "" {
			item.SysTids = appendUnique(item.SysTids, sysTid)
		}
		for _, o := range orderDetails {
			order, _ := o.(map[string]any)
			if order == nil {
				continue
			}
			if tid := asString(order["oid"], order["relationTid"], order["tid"]); tid != "" {
				item.Tids = appendUnique(item.Tids, tid)
			}
			item.Goods = append(item.Goods, TradeGoods{
				Title:   asString(order["title"], order["itemTitle"], order["goodsName"]),
				SkuName: asString(order["skuName"], order["colorName"], order["skuPropertiesName"]),
				PicURL:  asString(order["picUrl"], order["skuPicUrl"], order["picPath"], order["itemPic"]),
				Num:     asInt(order["num"], order["buyNum"]),
				OuterID: asString(order["outerId"], order["skuOuterId"], order["outerIid"]),
				Price:   asFloat(order["payment"], order["price"]),
			})
			mergeAfterSaleFromOrder(item, order)
		}
		if item.PlatformOrderStatus == "" {
			item.PlatformOrderStatus = asString(pkg["platformOrderStatus"], pkg["orderStatus"], pkg["status"])
		}
		if item.PlatformOrderStatusText == "" {
			item.PlatformOrderStatusText = ResolveOrderStatusText(
				asString(pkg["statusDesc"], pkg["platformOrderStatusDesc"], pkg["tradeStatusDesc"]),
				item.PlatformOrderStatus,
			)
		}
		if item.StatusText == "" {
			item.StatusText = item.PlatformOrderStatusText
		}
		return item
	}

	flattenTradeMap(item, pkg)
	if tid := asString(pkg["tid"]); tid != "" {
		item.Tids = appendUnique(item.Tids, tid)
	}
	if sysTid := asString(pkg["sysTid"]); sysTid != "" {
		item.SysTids = appendUnique(item.SysTids, sysTid)
	}
	if item.ShopID == "" {
		item.ShopID = asString(pkg["ownerShopId"], pkg["shopId"])
	}
	if item.StatusText == "" && item.TradeStatus != "" {
		item.StatusText = TradeStatusLabel(item.TradeStatus)
	}
	return item
}

func ParseTradeItemFromJSON(raw json.RawMessage, platform string) *TradeListItem {
	return parseTradeItem(raw, platform)
}

func parseTradeItemLegacyTrades(item *TradeListItem, trades []any) *TradeListItem {
	for _, t := range trades {
		trade, _ := t.(map[string]any)
		if trade == nil {
			continue
		}
		flattenTradeMap(item, trade)
		if tid := asString(trade["tid"]); tid != "" {
			item.Tids = appendUnique(item.Tids, tid)
		}
		if sysTid := asString(trade["sysTid"]); sysTid != "" {
			item.SysTids = appendUnique(item.SysTids, sysTid)
		}
		if orders, ok := trade["orders"].([]any); ok {
			for _, o := range orders {
				order, _ := o.(map[string]any)
				if order == nil {
					continue
				}
				item.Goods = append(item.Goods, TradeGoods{
					Title:   asString(order["title"], order["itemTitle"], order["goodsName"]),
					SkuName: asString(order["skuPropertiesName"], order["skuName"]),
					PicURL:  asString(order["picUrl"], order["skuPicUrl"], order["picPath"], order["itemPic"]),
					Num:     asInt(order["num"], order["buyNum"]),
					OuterID: asString(order["outerId"], order["outerIid"]),
					Price:   asFloat(order["price"], order["payment"]),
				})
			}
		}
	}
	return item
}

func flattenTradeMap(item *TradeListItem, trade map[string]any) {
	if item.ReceiverName == "" {
		item.ReceiverName = asString(trade["receiverName"], trade["receiverNameMask"])
	}
	if item.ReceiverMobile == "" {
		item.ReceiverMobile = asString(trade["receiverMobile"], trade["receiverMobileMask"])
	}
	if item.ReceiverAddress == "" {
		item.ReceiverAddress = stringsJoin(
			asString(trade["receiverState"], trade["receiverProvince"]),
			asString(trade["receiverCity"]),
			asString(trade["receiverDistrict"]),
			asString(trade["receiverTown"]),
			asString(trade["receiverAddress"], trade["receiverAddressMask"]),
		)
	}
	if item.PlatformOrderStatus == "" {
		item.PlatformOrderStatus = asString(trade["platformOrderStatus"], trade["orderStatus"], trade["status"])
	}
	if item.TradeStatus == "" {
		item.TradeStatus = asString(trade["tradeStatus"], trade["status"], trade["platformOrderStatus"])
	}
	if item.Payment == 0 {
		item.Payment = asFloat(trade["payment"], trade["payAmount"])
	}
	if item.CreateTime == "" {
		item.CreateTime = asString(trade["created"], trade["createTime"])
	}
	if item.PayTime == "" {
		item.PayTime = asString(trade["payTime"])
	}
	if item.ShopName == "" {
		item.ShopName = asString(trade["shopName"], trade["sellerNick"])
	}
	if item.ShopID == "" {
		item.ShopID = asString(trade["shopId"], trade["mallUserId"], trade["ownerShopId"])
	}
	if item.PlatformOrderStatusText == "" {
		item.PlatformOrderStatusText = ResolveOrderStatusText(
			asString(trade["statusDesc"], trade["platformOrderStatusDesc"], trade["tradeStatusDesc"]),
			item.PlatformOrderStatus,
		)
	}
	if item.StatusText == "" {
		item.StatusText = asString(trade["statusDesc"], trade["tradeStatusDesc"])
	}
	if item.AfterSaleStatus == "" {
		item.AfterSaleStatus = asString(trade["afterSaleStatus"], trade["refundStatus"])
	}
	if item.AfterSaleStatusText == "" && item.AfterSaleStatus != "" {
		item.AfterSaleStatusText = firstNonEmpty(
			asString(trade["afterSaleStatusText"], trade["refundStatusDesc"]),
			AfterSaleStatusLabel(item.AfterSaleStatus),
		)
	}
	if item.BuyerMemo == "" {
		item.BuyerMemo = asString(trade["buyerMemo"], trade["buyerMessage"], trade["buyerMessageMemo"])
	}
	if item.SellerMemo == "" {
		item.SellerMemo = asString(trade["sellerMemo"])
	}
	if item.FenFaMemo == "" {
		item.FenFaMemo = asString(trade["fenFaMemo"], trade["fenfaMemo"])
	}
	if item.PrinterMemo == "" {
		item.PrinterMemo = asString(trade["printerMemo"], trade["dadanMemo"])
	}
	if item.FactoryName == "" {
		item.FactoryName = asString(trade["factoryName"], trade["factoryNick"], trade["pushFactoryName"])
	}
	// 厂家备注不等于厂家名，仅在名称缺失时兜底
	if item.FactoryName == "" {
		if remark := strings.TrimSpace(asString(trade["factoryRemark"])); remark != "" {
			item.FactoryName = remark
		}
	}
	if at := asInt(trade["agentType"], trade["tradeAgentType"]); at != 0 {
		item.AgentType = at
	}
	// daifaStatus: 1自营 2代发；pushType: 2 推厂家
	daifa := asInt(trade["daifaStatus"], trade["daifaType"])
	pushType := asInt(trade["pushType"])
	explicitFactoryID := asString(trade["factoryId"], trade["pushFactoryId"])
	userFactoryID := asString(trade["factoryUserId"], trade["factoryUserID"])
	applyFactoryFields(item, explicitFactoryID, userFactoryID, daifa, pushType)
	normalizeAgentType(item, daifa, pushType)
}

// applyFactoryFields 仅在明确代发时采用 factoryUserId；裸 factoryUserId 常为商家自身 ID，不可当作厂家。
func applyFactoryFields(item *TradeListItem, explicitFactoryID, userFactoryID string, daifa, pushType int) {
	if item == nil {
		return
	}
	isDropship := daifa == 2 || pushType == 2 || item.AgentType == AgentTypePushFactory ||
		strings.TrimSpace(item.FactoryName) != ""
	if item.FactoryID == "" && explicitFactoryID != "" {
		item.FactoryID = explicitFactoryID
	}
	if item.FactoryID == "" && isDropship && userFactoryID != "" {
		item.FactoryID = userFactoryID
	}
}

// normalizeAgentType 以 daifaStatus/agentType/pushType 为准，禁止仅凭 factoryUserId 推断代发。
func normalizeAgentType(item *TradeListItem, daifa, pushType int) {
	if item == nil {
		return
	}
	switch {
	case item.AgentType == AgentTypePushFactory || daifa == 2 || pushType == 2:
		item.AgentType = AgentTypePushFactory
	case item.AgentType == AgentTypeSelfPrint || daifa == 1:
		item.AgentType = AgentTypeSelfPrint
		// 自营单清理误写入的厂家 ID（多为商家自身 factoryUserId）
		if strings.TrimSpace(item.FactoryName) == "" {
			item.FactoryID = ""
		}
	case strings.TrimSpace(item.FactoryName) != "" && strings.TrimSpace(item.FactoryID) != "":
		// 名称+厂家 ID 同时存在才视为代发（兼容接口未返回 daifaStatus）
		item.AgentType = AgentTypePushFactory
	default:
		if item.AgentType == 0 {
			item.AgentType = AgentTypeSelfPrint
		}
	}
}

func mergeAfterSaleFromOrder(item *TradeListItem, order map[string]any) {
	if item == nil || order == nil {
		return
	}
	status := asString(order["afterSaleStatus"], order["refundStatus"])
	if status == "" {
		return
	}
	// 优先保留进行中的售后；否则后写覆盖
	if item.AfterSaleStatus == "" || afterSalePriority(status) >= afterSalePriority(item.AfterSaleStatus) {
		item.AfterSaleStatus = status
		item.AfterSaleStatusText = firstNonEmpty(
			asString(order["afterSaleStatusText"], order["refundStatusDesc"]),
			AfterSaleStatusLabel(status),
		)
	}
	if plat := asString(order["orderStatus"], order["platformOrderStatus"]); plat != "" && item.PlatformOrderStatus == "" {
		item.PlatformOrderStatus = plat
	}
}

func afterSalePriority(status string) int {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "WAIT_SELLER_AGREE", "WAIT_BUYER_RETURN_ITEM", "WAIT_SELLER_CONFIRM_RECEIVE", "WAIT_BUYER_MODIFY",
		"WAIT_SEND_EXCHANGE_ITEM", "WAIT_RECEIVE_EXCHANGE_ITEM":
		return 3
	case "REFUND_SUCCESS":
		return 2
	case "SELLER_REFUSAL_REFUND", "SELLER_REFUSE", "REFUND_CLOSE":
		return 1
	default:
		return 0
	}
}

func asString(values ...any) string {
	for _, v := range values {
		switch x := v.(type) {
		case string:
			if x != "" {
				return x
			}
		case float64:
			if x != 0 {
				return strconv.FormatInt(int64(x), 10)
			}
		}
	}
	return ""
}

func asInt(values ...any) int {
	for _, v := range values {
		switch x := v.(type) {
		case float64:
			return int(x)
		case int:
			return x
		case string:
			n, _ := strconv.Atoi(x)
			return n
		}
	}
	return 0
}

func asFloat(values ...any) float64 {
	for _, v := range values {
		switch x := v.(type) {
		case float64:
			return x
		case string:
			f, _ := strconv.ParseFloat(x, 64)
			return f
		}
	}
	return 0
}

func appendUnique(list []string, v string) []string {
	for _, s := range list {
		if s == v {
			return list
		}
	}
	return append(list, v)
}

func stringsJoin(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return stringsJoinSpace(out)
}

func stringsJoinSpace(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	for i := 1; i < len(parts); i++ {
		s += " " + parts[i]
	}
	return s
}
