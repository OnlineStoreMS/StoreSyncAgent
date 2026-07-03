package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"storesyncagent/internal/config"
	"storesyncagent/internal/kdzs"
)

type SyncService struct {
	cfg             *config.Config
	client          *kdzs.Client
	session         *kdzs.Session
	mu              sync.Mutex
	activeAccountID string
}

func NewSyncService(cfg *config.Config) *SyncService {
	client := kdzs.NewClient(cfg.Kdzs.BaseURL)
	svc := &SyncService{
		cfg:             cfg,
		client:          client,
		session:         kdzs.NewSession(client),
		activeAccountID: cfg.Kdzs.ActiveAccountID(),
	}
	return svc
}

func (s *SyncService) activeAccount() (config.KdzsAccount, error) {
	acc, ok := s.cfg.Kdzs.AccountByID(s.activeAccountID)
	if !ok {
		return config.KdzsAccount{}, fmt.Errorf("account %s not found", s.activeAccountID)
	}
	if acc.Password == "" {
		acc.Password = s.cfg.Kdzs.Password
	}
	if acc.Mobile == "" {
		return config.KdzsAccount{}, fmt.Errorf("account mobile is empty")
	}
	return acc, nil
}

func (s *SyncService) ensureLogin(ctx context.Context) error {
	acc, err := s.activeAccount()
	if err != nil {
		return err
	}
	s.mu.Lock()
	needSwitch := s.session.AccountID() != acc.ID || s.client.Token() == ""
	s.mu.Unlock()
	if needSwitch {
		return s.session.SwitchAccount(ctx, acc.ID, acc.Name, acc.Role, acc.Mobile, acc.Password)
	}
	return s.session.EnsureLogin(ctx, acc.Mobile, acc.Password)
}

type ShopView struct {
	ID           int64  `json:"id"`
	Platform     string `json:"platform"`
	PlatformName string `json:"platformName"`
	MallUserID   string `json:"mallUserId"`
	MallUserName string `json:"mallUserName"`
	BindTime     string `json:"bindTime"`
	ExpireTime   string `json:"expireTime"`
	TokenValid   bool   `json:"tokenValid"`
}

func (s *SyncService) ListShops(ctx context.Context) ([]ShopView, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	shops, err := s.client.ListEcommerceShops(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ShopView, 0, len(shops))
	for _, shop := range shops {
		out = append(out, ShopView{
			ID:           shop.ID,
			Platform:     shop.Platform,
			PlatformName: kdzs.PlatformLabel(shop.Platform),
			MallUserID:   shop.MallUserID,
			MallUserName: shop.MallUserName,
			BindTime:     shop.BindTime,
			ExpireTime:   shop.ExpireTime,
			TokenValid:   shop.TokenValid,
		})
	}
	return out, nil
}

type OrderQuery struct {
	Platform      string `form:"platform"`
	ShopID        string `form:"shopId"`
	TradeStatus   string `form:"tradeStatus"`
	PageNo        int    `form:"pageNo"`
	PageSize      int    `form:"pageSize"`
	TimeType      int    `form:"timeType"`
	StartDateTime string `form:"startDateTime"`
	EndDateTime   string `form:"endDateTime"`
}

type OrderFiltersView struct {
	TimeType      int    `json:"timeType"`
	TimeTypeLabel string `json:"timeTypeLabel"`
	StartDateTime string `json:"startDateTime"`
	EndDateTime   string `json:"endDateTime"`
	Platform      string `json:"platform,omitempty"`
	PlatformName  string `json:"platformName,omitempty"`
	ShopID        string `json:"shopId,omitempty"`
	TradeStatus   string `json:"tradeStatus"`
	StatusLabel   string `json:"statusLabel"`
}

type OrderListView struct {
	Total    int                  `json:"total"`
	PageNo   int                  `json:"pageNo"`
	PageSize int                  `json:"pageSize"`
	Items    []kdzs.TradeListItem `json:"items"`
	Filters  *OrderFiltersView    `json:"filters,omitempty"`
	Stats    *OrderStatsView      `json:"stats,omitempty"`
	Hint     string               `json:"hint,omitempty"`
}

type OrderStatsView struct {
	WaitingPushTotal int            `json:"waitingPushTotal"`
	WaitingSendTotal int            `json:"waitingSendTotal"`
	WaitingPushByPlatform map[string]int `json:"waitingPushByPlatform,omitempty"`
	TabWaitAudit     int            `json:"tabWaitAudit,omitempty"`
	TabWaitSend      int            `json:"tabWaitSend,omitempty"`
}

func (s *SyncService) ListOrders(ctx context.Context, q OrderQuery) (*OrderListView, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	if q.TradeStatus == "" {
		q.TradeStatus = kdzs.DefaultTradeStatus()
	}

	var result *OrderListView
	var err error
	if q.Platform == "" {
		result, err = s.listOrdersAllPlatforms(ctx, q)
	} else {
		var tradeResult *kdzs.TradeListResult
		tradeResult, err = s.session.QueryTrades(ctx, s.toTradeQuery(q))
		if err == nil {
			result = &OrderListView{
				Total:    tradeResult.Total,
				PageNo:   tradeResult.PageNo,
				PageSize: tradeResult.PageSize,
				Items:    tradeResult.Items,
			}
		}
	}
	if err != nil {
		return nil, err
	}

	stats, _ := s.client.GetMainPageStats(ctx)
	if stats != nil {
		result.Stats = &OrderStatsView{
			WaitingPushTotal:      stats.WaitingPushOrderNum,
			WaitingSendTotal:      stats.WaitingSendOrderNum,
			WaitingPushByPlatform: stats.WaitingPushByPlatform,
		}
		if q.Platform != "" {
			if c, err := s.session.GetWaitSendCount(ctx, q.Platform, nil, nil, shopIDs(q.ShopID)); err == nil && c != nil {
				result.Stats.TabWaitAudit = c.WaitAudit
				result.Stats.TabWaitSend = c.WaitSend
			}
		}
	}

	if result.Total == 0 && result.Stats != nil && result.Stats.WaitingPushTotal > 0 && q.TradeStatus == "wait_audit" {
		result.Hint = "快递助手首页显示有待推单，但列表为空。请检查抖店店铺授权是否有效（店铺列表中「授权状态」），并在快递助手网页端「推送订单-待推单」确认是否可见。"
	}
	result.Filters = buildOrderFilters(q)
	return result, nil
}

func buildOrderFilters(q OrderQuery) *OrderFiltersView {
	start, end := kdzs.ResolveDateRange(q.StartDateTime, q.EndDateTime)
	filters := &OrderFiltersView{
		TimeType:      q.TimeType,
		TimeTypeLabel: kdzs.TimeTypeLabel(q.TimeType),
		StartDateTime: start,
		EndDateTime:   end,
		Platform:      q.Platform,
		ShopID:        q.ShopID,
		TradeStatus:   q.TradeStatus,
		StatusLabel:   kdzs.TradeStatusLabel(q.TradeStatus),
	}
	if q.Platform != "" {
		filters.PlatformName = kdzs.PlatformLabel(q.Platform)
	}
	return filters
}

func (s *SyncService) toTradeQuery(q OrderQuery) kdzs.TradeQuery {
	return kdzs.TradeQuery{
		Platform:      q.Platform,
		TradeStatus:   q.TradeStatus,
		PageNo:        q.PageNo,
		PageSize:      q.PageSize,
		ShopID:        q.ShopID,
		TimeType:      q.TimeType,
		StartDateTime: q.StartDateTime,
		EndDateTime:   q.EndDateTime,
	}
}

func shopIDs(id string) []string {
	if id == "" {
		return nil
	}
	return []string{id}
}

func (s *SyncService) listOrdersAllPlatforms(ctx context.Context, q OrderQuery) (*OrderListView, error) {
	shops, err := s.client.ListEcommerceShops(ctx)
	if err != nil {
		return nil, err
	}
	platforms := uniquePlatforms(shops)
	if len(platforms) == 0 {
		return &OrderListView{Items: []kdzs.TradeListItem{}}, nil
	}

	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	allItems := make([]kdzs.TradeListItem, 0)
	total := 0
	for _, platform := range platforms {
		pq := q
		pq.Platform = platform
		pq.PageNo = 1
		pq.PageSize = pageSize
		result, err := s.session.QueryTrades(ctx, s.toTradeQuery(pq))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", platform, err)
		}
		total += result.Total
		allItems = append(allItems, result.Items...)
	}

	if len(allItems) > pageSize {
		allItems = allItems[:pageSize]
	}

	return &OrderListView{
		Total:    total,
		PageNo:   1,
		PageSize: pageSize,
		Items:    allItems,
	}, nil
}

type DecryptOrdersRequest struct {
	Platform    string   `json:"platform"`
	TradeStatus string   `json:"tradeStatus"`
	SysTids     []string `json:"sysTids"`
}

type DecryptOrdersView struct {
	Items []kdzs.TradeListItem `json:"items"`
}

func (s *SyncService) DecryptOrders(ctx context.Context, req DecryptOrdersRequest) (*DecryptOrdersView, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	if req.Platform == "" {
		return nil, fmt.Errorf("platform is required")
	}
	if len(req.SysTids) == 0 {
		return nil, fmt.Errorf("sysTids is required")
	}
	tradeStatus := req.TradeStatus
	if tradeStatus == "" {
		tradeStatus = kdzs.DefaultTradeStatus()
	}

	metaBySysTid, err := s.session.FetchDecryptMetaBySysTids(ctx, req.Platform, tradeStatus, req.SysTids)
	if err != nil {
		return nil, err
	}

	items := make([]kdzs.TradeListItem, 0, len(req.SysTids))
	for _, sysTid := range req.SysTids {
		meta, ok := metaBySysTid[sysTid]
		if !ok {
			return nil, fmt.Errorf("order %s not found", sysTid)
		}
		decrypted, err := s.session.DecodeTradeReceiver(ctx, req.Platform, meta)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", sysTid, err)
		}
		item := kdzs.TradeListItem{
			Platform:     req.Platform,
			PlatformName: kdzs.PlatformLabel(req.Platform),
			SysTids:      []string{sysTid},
			ShopID:       meta.OwnerShopID,
		}
		if meta.Tid != "" {
			item.Tids = []string{meta.Tid}
		}
		kdzs.ApplyDecryptedReceiver(&item, decrypted)
		items = append(items, item)
	}
	return &DecryptOrdersView{Items: items}, nil
}

func uniquePlatforms(shops []kdzs.BindShop) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, shop := range shops {
		if !kdzs.IsEcommercePlatform(shop.Platform) {
			continue
		}
		if _, ok := seen[shop.Platform]; ok {
			continue
		}
		seen[shop.Platform] = struct{}{}
		out = append(out, shop.Platform)
	}
	return out
}

func (s *SyncService) LoginStatus(ctx context.Context) (map[string]any, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	acc, _ := s.activeAccount()
	return map[string]any{
		"loggedIn":        true,
		"userId":          s.session.UserID(),
		"mobile":          s.session.Mobile(),
		"accountId":       acc.ID,
		"accountName":     acc.Name,
		"accountRole":     acc.Role,
		"accountRoleLabel": accountRoleLabel(acc.Role),
	}, nil
}

type KdzsAccountView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	RoleLabel string `json:"roleLabel"`
	Mobile   string `json:"mobile"`
	Active   bool   `json:"active"`
}

func accountRoleLabel(role string) string {
	switch role {
	case "factory":
		return "厂家版"
	case "merchant":
		return "商家版"
	default:
		return role
	}
}

func (s *SyncService) ListAccounts() []KdzsAccountView {
	items := make([]KdzsAccountView, 0, len(s.cfg.Kdzs.ResolveAccounts()))
	for _, acc := range s.cfg.Kdzs.ResolveAccounts() {
		items = append(items, KdzsAccountView{
			ID:        acc.ID,
			Name:      acc.Name,
			Role:      acc.Role,
			RoleLabel: accountRoleLabel(acc.Role),
			Mobile:    acc.Mobile,
			Active:    acc.ID == s.activeAccountID,
		})
	}
	return items
}

func (s *SyncService) SwitchAccount(ctx context.Context, accountID string) (map[string]any, error) {
	acc, ok := s.cfg.Kdzs.AccountByID(accountID)
	if !ok {
		return nil, fmt.Errorf("account %s not found", accountID)
	}
	if acc.Password == "" {
		acc.Password = s.cfg.Kdzs.Password
	}
	s.mu.Lock()
	s.activeAccountID = accountID
	s.mu.Unlock()
	if err := s.session.SwitchAccount(ctx, acc.ID, acc.Name, acc.Role, acc.Mobile, acc.Password); err != nil {
		return nil, err
	}
	return s.LoginStatus(ctx)
}

type FactoryQuery struct {
	Platform string `form:"platform"`
	PageNo   int    `form:"pageNo"`
	PageSize int    `form:"pageSize"`
}

func (s *SyncService) ListFactories(ctx context.Context, q FactoryQuery) (*kdzs.FactoryListResult, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	platform := q.Platform
	if platform == "" {
		platform = "FXG"
	}
	return s.session.ListFactories(ctx, platform, q.PageNo, q.PageSize)
}

type SetOrderAgentTypeRequest struct {
	Platform    string   `json:"platform"`
	TradeStatus string   `json:"tradeStatus"`
	Action      string   `json:"action"` // self_print | push_factory
	FactoryID   string   `json:"factoryId"`
	SysTids     []string `json:"sysTids"`
}

func (s *SyncService) SetOrderAgentType(ctx context.Context, req SetOrderAgentTypeRequest) (*kdzs.AgentTypeResult, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	if req.Platform == "" {
		return nil, fmt.Errorf("platform is required")
	}
	if len(req.SysTids) == 0 {
		return nil, fmt.Errorf("sysTids is required")
	}
	agentType := kdzs.AgentTypeSelfPrint
	switch req.Action {
	case "self_print":
		agentType = kdzs.AgentTypeSelfPrint
	case "push_factory":
		agentType = kdzs.AgentTypePushFactory
	default:
		return nil, fmt.Errorf("invalid action")
	}
	return s.session.SetTradeAgentType(ctx, kdzs.SetTradeAgentTypeRequest{
		Platform:    req.Platform,
		TradeStatus: req.TradeStatus,
		AgentType:   agentType,
		FactoryID:   req.FactoryID,
		SysTids:     req.SysTids,
	})
}

type RefundQuery struct {
	Platform          string `form:"platform"`
	ShopID            string `form:"shopId"`
	PageNo            int    `form:"pageNo"`
	PageSize          int    `form:"pageSize"`
	DateType          int    `form:"dateType"`
	StartDateTime     string `form:"startDateTime"`
	EndDateTime       string `form:"endDateTime"`
	AfterSaleStatus   string `form:"afterSaleStatus"`
	AfterSaleType     string `form:"afterSaleType"`
	Sid               string `form:"sid"`
	RefundID          string `form:"refundId"`
	Tid               string `form:"tid"`
	SysTid            string `form:"sysTid"`
	Scenario          string `form:"scenario"`
	EnrichLogistics   bool   `form:"enrichLogistics"`
	TradeDaifaStatus  string `form:"tradeDaifaStatus"`
}

type RefundFiltersView struct {
	DateType      int    `json:"dateType"`
	DateTypeLabel string `json:"dateTypeLabel"`
	StartDateTime string `json:"startDateTime"`
	EndDateTime   string `json:"endDateTime"`
	Platform      string `json:"platform,omitempty"`
	PlatformName  string `json:"platformName,omitempty"`
	ShopID        string `json:"shopId,omitempty"`
	Scenario      string `json:"scenario,omitempty"`
	Sid           string `json:"sid,omitempty"`
}

type RefundStatsView struct {
	Total                    int `json:"total"`
	WaitSellerConfirmReceive int `json:"waitSellerConfirmReceive"`
	WaitSellerAgree          int `json:"waitSellerAgree"`
	RefundOnlyPending        int `json:"refundOnlyPending"`
	ExchangePending          int `json:"exchangePending"`
	ReturnSigned             int `json:"returnSigned"`
	PickupPending            int `json:"pickupPending"`
	Urgent                   int `json:"urgent"`
	Critical                 int `json:"critical"`
	Expired                  int `json:"expired"`
}

type RefundListView struct {
	Total    int                  `json:"total"`
	PageNo   int                  `json:"pageNo"`
	PageSize int                  `json:"pageSize"`
	Items    []kdzs.RefundItem    `json:"items"`
	Filters  *RefundFiltersView   `json:"filters,omitempty"`
	Stats    *RefundStatsView     `json:"stats,omitempty"`
}

func (s *SyncService) ListRefunds(ctx context.Context, q RefundQuery) (*RefundListView, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	platform := q.Platform
	if platform == "" {
		platform = "FXG"
	}

	rq := s.toRefundQuery(q)
	var result *kdzs.RefundListResult
	var err error

	if kdzs.ScenarioNeedsFullListScan(q.Scenario) && q.Sid == "" {
		result, err = s.listRefundsByScenario(ctx, platform, q, rq)
	} else {
		result, err = s.session.QueryRefunds(ctx, rq)
		if err == nil {
			if q.EnrichLogistics || q.Scenario != "" || q.Sid != "" {
				s.session.EnrichRefundsLogistics(ctx, platform, result.Items, 5)
			} else {
				for i := range result.Items {
					result.Items[i].SLA = kdzs.ComputeRefundSLA(&result.Items[i], nil, time.Now())
				}
			}
		}
	}
	if err != nil {
		return nil, err
	}

	stats, _ := s.fetchRefundStats(ctx, platform, q)
	start, end := kdzs.ResolveRefundSearchDateRange(q.StartDateTime, q.EndDateTime, q.Sid != "")

	return &RefundListView{
		Total:    result.Total,
		PageNo:   result.PageNo,
		PageSize: result.PageSize,
		Items:    result.Items,
		Filters: &RefundFiltersView{
			DateType:      kdzs.RefundDateTypeLabel(q.DateType),
			DateTypeLabel: "申请时间",
			StartDateTime: start,
			EndDateTime:   end,
			Platform:      platform,
			PlatformName:  kdzs.PlatformLabel(platform),
			ShopID:        q.ShopID,
			Scenario:      q.Scenario,
			Sid:           q.Sid,
		},
		Stats: stats,
	}, nil
}

func (s *SyncService) GetRefundStats(ctx context.Context, q RefundQuery) (*RefundStatsView, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	platform := q.Platform
	if platform == "" {
		platform = "FXG"
	}
	return s.fetchRefundStats(ctx, platform, q)
}

func (s *SyncService) listRefundsByScenario(ctx context.Context, platform string, q RefundQuery, base kdzs.RefundQuery) (*kdzs.RefundListResult, error) {
	pageNo := q.PageNo
	if pageNo <= 0 {
		pageNo = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	matched, err := s.collectScenarioRefunds(ctx, platform, q.Scenario, base)
	if err != nil {
		return nil, err
	}
	kdzs.SortRefundItemsBySLAUrgency(matched)

	total := len(matched)
	start := (pageNo - 1) * pageSize
	if start >= total {
		return &kdzs.RefundListResult{Total: total, PageNo: pageNo, PageSize: pageSize, Items: []kdzs.RefundItem{}}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return &kdzs.RefundListResult{
		Total:    total,
		PageNo:   pageNo,
		PageSize: pageSize,
		Items:    matched[start:end],
	}, nil
}

func (s *SyncService) collectScenarioRefunds(ctx context.Context, platform, scenario string, base kdzs.RefundQuery) ([]kdzs.RefundItem, error) {
	scenario = strings.TrimSpace(scenario)
	now := time.Now()

	switch scenario {
	case "confirm_receive":
		q := base
		q.AfterSaleStatusList = []string{"WAIT_SELLER_CONFIRM_RECEIVE"}
		q.AfterSaleTypeList = nil
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		s.session.EnrichRefundsLogistics(ctx, platform, items, 8)
		for i := range items {
			if items[i].SLA == nil {
				items[i].SLA = kdzs.ComputeRefundSLA(&items[i], nil, now)
			}
		}
		return items, nil

	case "wait_agree":
		q := base
		q.AfterSaleStatusList = []string{"WAIT_SELLER_AGREE"}
		q.AfterSaleTypeList = nil
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].SLA = kdzs.ComputeRefundSLA(&items[i], nil, now)
		}
		return items, nil

	case "refund_only":
		q := base
		q.AfterSaleStatusList = []string{"WAIT_SELLER_AGREE"}
		q.AfterSaleTypeList = []int{1}
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].SLA = kdzs.ComputeRefundSLA(&items[i], nil, now)
		}
		return items, nil

	case "exchange":
		q := base
		q.AfterSaleTypeList = []int{3}
		q.AfterSaleStatusList = nil
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		s.session.EnrichRefundsLogistics(ctx, platform, items, 8)
		filtered := make([]kdzs.RefundItem, 0, len(items))
		for i := range items {
			if !kdzs.ActiveExchangeStatus(items[i].AfterSaleStatus) {
				continue
			}
			if items[i].SLA == nil {
				items[i].SLA = kdzs.ComputeRefundSLA(&items[i], nil, now)
			}
			filtered = append(filtered, items[i])
		}
		return filtered, nil
	}

	var matched []kdzs.RefundItem

	confirmQ := base
	confirmQ.AfterSaleStatusList = []string{"WAIT_SELLER_CONFIRM_RECEIVE"}
	confirmQ.AfterSaleTypeList = nil
	confirmItems, _, err := s.session.QueryAllRefunds(ctx, confirmQ)
	if err != nil {
		return nil, err
	}
	s.session.EnrichRefundsLogistics(ctx, platform, confirmItems, 8)
	for _, item := range confirmItems {
		if kdzs.MatchRefundScenario(item, scenario) {
			matched = append(matched, item)
		}
	}

	if scenario == "urgent" {
		agreeQ := base
		agreeQ.AfterSaleStatusList = []string{"WAIT_SELLER_AGREE"}
		agreeQ.AfterSaleTypeList = nil
		agreeItems, _, err := s.session.QueryAllRefunds(ctx, agreeQ)
		if err != nil {
			return nil, err
		}
		for i := range agreeItems {
			agreeItems[i].SLA = kdzs.ComputeRefundSLA(&agreeItems[i], nil, now)
			if kdzs.MatchRefundScenario(agreeItems[i], scenario) {
				matched = append(matched, agreeItems[i])
			}
		}
	}
	return matched, nil
}

func (s *SyncService) loadConfirmReceiveWithSLA(ctx context.Context, platform string, base kdzs.RefundQuery) ([]kdzs.RefundItem, error) {
	q := base
	q.AfterSaleStatusList = []string{"WAIT_SELLER_CONFIRM_RECEIVE"}
	q.AfterSaleTypeList = nil
	items, _, err := s.session.QueryAllRefunds(ctx, q)
	if err != nil {
		return nil, err
	}
	s.session.EnrichRefundsLogistics(ctx, platform, items, 8)
	return items, nil
}

func applyConfirmReceiveSLAStats(stats *RefundStatsView, items []kdzs.RefundItem) {
	for _, item := range items {
		if item.SLA == nil {
			continue
		}
		if item.SLA.IsSigned {
			stats.ReturnSigned++
		}
		if item.SLA.IsPickupPending && !item.SLA.IsSigned {
			stats.PickupPending++
		}
		switch item.SLA.Urgency {
		case "critical", "expired", "warning":
			stats.Urgent++
		}
		if item.SLA.Urgency == "critical" {
			stats.Critical++
		}
		if item.SLA.Urgency == "expired" {
			stats.Expired++
		}
	}
}

func (s *SyncService) GetRefundLogistics(ctx context.Context, platform, sid, sidCode string) (*kdzs.LogisticsDetail, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	if platform == "" {
		platform = "FXG"
	}
	return s.session.GetLogisticsDetail(ctx, platform, sid, sidCode)
}

func (s *SyncService) toRefundQuery(q RefundQuery) kdzs.RefundQuery {
	rq := kdzs.RefundQuery{
		Platform:         q.Platform,
		ShopID:           q.ShopID,
		PageNo:           q.PageNo,
		PageSize:         q.PageSize,
		DateType:         q.DateType,
		StartDateTime:    q.StartDateTime,
		EndDateTime:      q.EndDateTime,
		Sid:              q.Sid,
		RefundID:         q.RefundID,
		Tid:              q.Tid,
		SysTid:           q.SysTid,
		TradeDaifaStatus: q.TradeDaifaStatus,
	}
	if q.AfterSaleStatus != "" {
		rq.AfterSaleStatusList = splitCSV(q.AfterSaleStatus)
	}
	if q.AfterSaleType != "" {
		for _, part := range splitCSV(q.AfterSaleType) {
			if n, err := strconv.Atoi(part); err == nil {
				rq.AfterSaleTypeList = append(rq.AfterSaleTypeList, n)
			}
		}
	}
	if q.Scenario != "" && len(rq.AfterSaleStatusList) == 0 && q.Sid == "" {
		switch q.Scenario {
		case "confirm_receive", "return_signed", "pickup_pending":
			rq.AfterSaleStatusList = []string{"WAIT_SELLER_CONFIRM_RECEIVE"}
		case "wait_agree", "refund_only":
			rq.AfterSaleStatusList = []string{"WAIT_SELLER_AGREE"}
		}
		if q.Scenario == "refund_only" {
			rq.AfterSaleTypeList = []int{1}
		}
		if q.Scenario == "exchange" {
			rq.AfterSaleTypeList = []int{3}
		}
	}
	return rq
}

func (s *SyncService) fetchRefundStats(ctx context.Context, platform string, q RefundQuery) (*RefundStatsView, error) {
	base := s.toRefundQuery(q)
	base.PageNo = 1
	base.PageSize = 1
	base.AfterSaleStatusList = nil
	base.AfterSaleTypeList = nil

	all, err := s.session.QueryRefunds(ctx, base)
	if err != nil {
		return nil, err
	}
	stats := &RefundStatsView{Total: all.Total}

	countStatus := func(status string) int {
		rq := base
		rq.AfterSaleStatusList = []string{status}
		res, err := s.session.QueryRefunds(ctx, rq)
		if err != nil {
			return 0
		}
		return res.Total
	}
	stats.WaitSellerConfirmReceive = countStatus("WAIT_SELLER_CONFIRM_RECEIVE")
	stats.WaitSellerAgree = countStatus("WAIT_SELLER_AGREE")

	rq := base
	rq.AfterSaleStatusList = []string{"WAIT_SELLER_AGREE"}
	rq.AfterSaleTypeList = []int{1}
	if res, err := s.session.QueryRefunds(ctx, rq); err == nil {
		stats.RefundOnlyPending = res.Total
	}

	exQ := base
	exQ.AfterSaleTypeList = []int{3}
	if exItems, _, err := s.session.QueryAllRefunds(ctx, exQ); err == nil {
		for _, item := range exItems {
			if kdzs.ActiveExchangeStatus(item.AfterSaleStatus) {
				stats.ExchangePending++
			}
		}
	}

	// 待确认收货：拉全量并查物流，统计签收/待取件/紧迫（与场景 Tab 一致）。
	if confirmItems, err := s.loadConfirmReceiveWithSLA(ctx, platform, base); err == nil {
		applyConfirmReceiveSLAStats(stats, confirmItems)
	}

	// Urgent from wait_agree (仅退款).
	agreeQ := base
	agreeQ.PageSize = 50
	agreeQ.AfterSaleStatusList = []string{"WAIT_SELLER_AGREE"}
	if res, err := s.session.QueryRefunds(ctx, agreeQ); err == nil {
		for i := range res.Items {
			sla := kdzs.ComputeRefundSLA(&res.Items[i], nil, time.Now())
			if sla.Urgency == "critical" || sla.Urgency == "expired" || sla.Urgency == "warning" {
				stats.Urgent++
			}
			if sla.Urgency == "critical" {
				stats.Critical++
			}
			if sla.Urgency == "expired" {
				stats.Expired++
			}
		}
	}

	return stats, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
