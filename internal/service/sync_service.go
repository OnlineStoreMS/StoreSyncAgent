package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"storesyncagent/internal/config"
	"storesyncagent/internal/feishu"
	"storesyncagent/internal/kdzs"
	"storesyncagent/internal/model"
	"storesyncagent/internal/repo"
)

type SyncService struct {
	cfg                 *config.Config
	tenantID            uint64
	globalBaseURL       string
	kdzsRepo            *repo.KdzsRepo
	settings            *model.TenantKdzsSetting
	client              *kdzs.Client
	session             *kdzs.Session
	mu                  sync.Mutex
	activeAccountID     string
	returnExchangeRepo  *repo.ReturnExchangeRepo
	notificationRepo    *repo.NotificationRepo
	feishuClient        *feishu.Client
}

func NewSyncService(
	baseCfg *config.Config,
	tenantID uint64,
	kdzsRepo *repo.KdzsRepo,
	returnExchangeRepo *repo.ReturnExchangeRepo,
	notificationRepo *repo.NotificationRepo,
) (*SyncService, error) {
	globalBaseURL := baseCfg.Kdzs.BaseURL
	if globalBaseURL == "" {
		globalBaseURL = "https://df.kdzs.com"
	}
	tenantCfg := &config.Config{
		Server: baseCfg.Server,
	}
	svc := &SyncService{
		cfg:                tenantCfg,
		tenantID:           tenantID,
		globalBaseURL:      globalBaseURL,
		kdzsRepo:           kdzsRepo,
		returnExchangeRepo: returnExchangeRepo,
		notificationRepo:   notificationRepo,
		feishuClient:       feishu.NewClient(),
	}
	if err := svc.loadSettings(); err != nil {
		return nil, fmt.Errorf("kdzs settings: %w", err)
	}
	svc.client = kdzs.NewClient(svc.kdzsBaseURL())
	svc.session = kdzs.NewSession(svc.client)
	if err := svc.ensureDefaultActiveAccount(); err != nil {
		svc.activeAccountID = ""
	}
	return svc, nil
}

func (s *SyncService) activeAccount() (config.KdzsAccount, error) {
	if err := s.ensureDefaultActiveAccount(); err != nil {
		return config.KdzsAccount{}, err
	}
	acc, ok := s.accountByCode(s.activeAccountID)
	if !ok {
		return config.KdzsAccount{}, fmt.Errorf("account %s not found", s.activeAccountID)
	}
	if acc.Mobile == "" {
		return config.KdzsAccount{}, fmt.Errorf("account mobile is empty")
	}
	if acc.Password == "" {
		return config.KdzsAccount{}, fmt.Errorf("account password is empty")
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

// switchSessionAccount 切换 KDZS 会话到指定账号，不改变 Web 当前选中的 activeAccountID。
func (s *SyncService) switchSessionAccount(ctx context.Context, accountID string) (config.KdzsAccount, error) {
	acc, ok := s.accountByCode(accountID)
	if !ok {
		return config.KdzsAccount{}, fmt.Errorf("account %s not found", accountID)
	}
	if acc.Mobile == "" {
		return config.KdzsAccount{}, fmt.Errorf("account %s mobile is empty", accountID)
	}
	if acc.Password == "" {
		return config.KdzsAccount{}, fmt.Errorf("account %s password is empty", accountID)
	}
	if err := s.session.SwitchAccount(ctx, acc.ID, acc.Name, acc.Role, acc.Mobile, acc.Password); err != nil {
		return config.KdzsAccount{}, err
	}
	return acc, nil
}

func (s *SyncService) restoreSessionAccount(ctx context.Context, accountID string) {
	if accountID == "" {
		return
	}
	if _, err := s.switchSessionAccount(ctx, accountID); err != nil {
		// best effort restore after notification poll
	}
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
	Tid           string `form:"tid"`
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
	if q.TradeStatus == "" && strings.TrimSpace(q.Tid) == "" {
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
		} else if shops, err := s.client.ListEcommerceShops(ctx); err == nil {
			// 全平台：汇总各平台待推/待发数量
			for i, p := range uniquePlatforms(shops) {
				if i > 0 {
					select {
					case <-ctx.Done():
					case <-time.After(800 * time.Millisecond):
					}
				}
				if c, err := s.session.GetWaitSendCount(ctx, p, nil, nil, nil); err == nil && c != nil {
					result.Stats.TabWaitAudit += c.WaitAudit
					result.Stats.TabWaitSend += c.WaitSend
				}
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
		Tid:           q.Tid,
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

	pageNo := q.PageNo
	if pageNo <= 0 {
		pageNo = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	// 各平台拉全量（按筛选条件），合并后再本地分页，保证待推/待发/全部跨平台完整
	fetchSize := pageSize
	if fetchSize < 50 {
		fetchSize = 50
	}
	allItems := make([]kdzs.TradeListItem, 0)
	for i, platform := range platforms {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(3500 * time.Millisecond):
			}
		}
		for page := 1; ; page++ {
			if page > 1 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(3500 * time.Millisecond):
				}
			}
			pq := q
			pq.Platform = platform
			pq.PageNo = page
			pq.PageSize = fetchSize
			result, err := s.session.QueryTrades(ctx, s.toTradeQuery(pq))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", platform, err)
			}
			allItems = append(allItems, result.Items...)
			if len(result.Items) == 0 {
				break
			}
			if result.Total > 0 && page*fetchSize >= result.Total {
				break
			}
			if result.Total <= 0 && len(result.Items) < fetchSize {
				break
			}
		}
	}

	kdzs.SortTradeItemsByOrderTimeDesc(allItems)
	total := len(allItems)
	start := (pageNo - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	pageItems := allItems[start:end]

	return &OrderListView{
		Total:    total,
		PageNo:   pageNo,
		PageSize: pageSize,
		Items:    pageItems,
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
	items := make([]KdzsAccountView, 0)
	accounts, err := s.resolveAccounts()
	if err != nil {
		return items
	}
	for _, acc := range accounts {
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
	acc, ok := s.accountByCode(accountID)
	if !ok {
		return nil, fmt.Errorf("account %s not found", accountID)
	}
	s.mu.Lock()
	s.activeAccountID = accountID
	s.mu.Unlock()
	_ = s.kdzsRepo.UpdateActiveAccount(s.tenantID, accountID)
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

type CancelOrderPushRequest struct {
	Platform    string   `json:"platform"`
	TradeStatus string   `json:"tradeStatus"`
	SysTids     []string `json:"sysTids"`
}

// CancelOrderPush 撤回推单/退审：快递助手待发货 → 待推单。
func (s *SyncService) CancelOrderPush(ctx context.Context, req CancelOrderPushRequest) (*kdzs.AgentTypeResult, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	if req.Platform == "" {
		return nil, fmt.Errorf("platform is required")
	}
	if len(req.SysTids) == 0 {
		return nil, fmt.Errorf("sysTids is required")
	}
	return s.session.CancelTradePush(ctx, kdzs.CancelTradePushRequest{
		Platform:    req.Platform,
		TradeStatus: req.TradeStatus,
		SysTids:     req.SysTids,
	})
}

// ShipCallbackRequest 订单中心回传物流信息。
type ShipCallbackRequest struct {
	Platform       string `json:"platform"`
	ShopID         string `json:"shopId"`
	PlatformTid    string `json:"platformTid"`
	PlatformSysTid string `json:"platformSysTid"`
	ExpressCompany string `json:"expressCompany"`
	ExpressNo      string `json:"expressNo"`
	OrderNo        string `json:"orderNo"`
	Remark         string `json:"remark"`
}

type ShipCallbackResult struct {
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}

// ShipCallback 接收 OrderCore 物流回传。
// 当前先校验并落日志，后续对接快递助手/电商平台「上传运单号」接口完成真实发货。
func (s *SyncService) ShipCallback(ctx context.Context, req ShipCallbackRequest) (*ShipCallbackResult, error) {
	if strings.TrimSpace(req.ExpressNo) == "" {
		return nil, fmt.Errorf("expressNo is required")
	}
	if strings.TrimSpace(req.PlatformTid) == "" && strings.TrimSpace(req.PlatformSysTid) == "" {
		return nil, fmt.Errorf("platformTid or platformSysTid is required")
	}
	// TODO: 对接 KDZS / 平台发货 API（按 platform + tid 上传运单号）
	_ = ctx
	_ = s
	return &ShipCallbackResult{
		Accepted: true,
		Message:  fmt.Sprintf("已接收物流回传（待对接平台发货）: %s %s", req.ExpressCompany, req.ExpressNo),
	}, nil
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
	Tid           string `json:"tid,omitempty"`
}

type RefundStatsView struct {
	Total                    int `json:"total"`
	WaitSellerConfirmReceive int `json:"waitSellerConfirmReceive"`
	WaitSellerAgree          int `json:"waitSellerAgree"`
	RefundOnlyPending        int `json:"refundOnlyPending"`
	ExchangePending          int `json:"exchangePending"`
	WaitSendExchange         int `json:"waitSendExchange"`
	ReturnSigned             int `json:"returnSigned"`
	PickupPending            int `json:"pickupPending"`
	Urgent                   int `json:"urgent"`
	Imminent                 int `json:"imminent"`
	Critical                 int `json:"critical"`
	Expired                  int `json:"expired"`
	WaitBuyerReturn          int `json:"waitBuyerReturn"`
	SellerRefuse             int `json:"sellerRefuse"`
	RefundCloseWithSid       int `json:"refundCloseWithSid"`
	RefundSuccess            int `json:"refundSuccess"`
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
	hasIDSearch := strings.TrimSpace(q.Sid) != "" || strings.TrimSpace(q.Tid) != "" || strings.TrimSpace(q.SysTid) != ""
	var result *kdzs.RefundListResult
	var err error

	if kdzs.ScenarioNeedsFullListScan(q.Scenario) && !hasIDSearch {
		result, err = s.listRefundsByScenario(ctx, platform, q, rq)
	} else {
		result, err = s.session.QueryRefunds(ctx, rq)
		if err == nil && q.Scenario != "" {
			filtered := filterRefundsByScenario(result.Items, q.Scenario)
			result.Items = filtered
			result.Total = len(filtered)
		}
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
	start, end := kdzs.ResolveRefundSearchDateRange(q.StartDateTime, q.EndDateTime, hasIDSearch)

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
			Tid:           q.Tid,
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

	case "wait_send_exchange":
		q := base
		q.AfterSaleStatusList = []string{"WAIT_SEND_EXCHANGE_ITEM"}
		q.AfterSaleTypeList = nil
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].SLA = kdzs.ComputeRefundSLA(&items[i], nil, now)
		}
		return items, nil

	case "urgent":
		return s.collectUrgentRefunds(ctx, platform, base, now)

	case "wait_return":
		q := base
		q.AfterSaleStatusList = []string{"WAIT_BUYER_RETURN_ITEM"}
		q.AfterSaleTypeList = nil
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].SLA = kdzs.ComputeRefundSLA(&items[i], nil, now)
		}
		return items, nil

	case "refund_success":
		q := base
		q.AfterSaleStatusList = []string{"REFUND_SUCCESS"}
		q.AfterSaleTypeList = []int{kdzs.AfterSaleTypeReturnRefund}
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		return items, nil

	case "seller_refuse":
		q := base
		q.AfterSaleStatusList = []string{kdzs.SellerRefuseQueryStatus()}
		q.AfterSaleTypeList = nil
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		return items, nil

	case "refund_close_with_sid":
		q := base
		q.AfterSaleStatusList = []string{"REFUND_CLOSE"}
		q.AfterSaleTypeList = nil
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		filtered := filterRefundsWithReturnLogistics(items)
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
	return matched, nil
}

func (s *SyncService) collectUrgentRefunds(ctx context.Context, platform string, base kdzs.RefundQuery, now time.Time) ([]kdzs.RefundItem, error) {
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
		if kdzs.IsUrgentSLA(item.SLA) {
			matched = append(matched, item)
		}
	}

	for _, status := range []string{"WAIT_SELLER_AGREE", "WAIT_SEND_EXCHANGE_ITEM", "WAIT_RECEIVE_EXCHANGE_ITEM"} {
		q := base
		q.AfterSaleStatusList = []string{status}
		q.AfterSaleTypeList = nil
		items, _, err := s.session.QueryAllRefunds(ctx, q)
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i].SLA = kdzs.ComputeRefundSLA(&items[i], nil, now)
			if kdzs.IsUrgentSLA(items[i].SLA) {
				matched = append(matched, items[i])
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
	}
}

func countUrgentSLAStats(stats *RefundStatsView, items []kdzs.RefundItem) {
	for _, item := range items {
		if item.SLA == nil {
			continue
		}
		if !kdzs.IsUrgentSLA(item.SLA) {
			continue
		}
		stats.Urgent++
		switch item.SLA.Urgency {
		case "imminent":
			stats.Imminent++
		case "critical":
			stats.Critical++
		case "expired":
			stats.Expired++
		}
	}
}

func (s *SyncService) countUrgentStatsForStatus(ctx context.Context, platform string, base kdzs.RefundQuery, status string, withLogistics bool, stats *RefundStatsView) error {
	q := base
	q.AfterSaleStatusList = []string{status}
	q.AfterSaleTypeList = nil
	items, _, err := s.session.QueryAllRefunds(ctx, q)
	if err != nil {
		return err
	}
	if withLogistics {
		s.session.EnrichRefundsLogistics(ctx, platform, items, 8)
	} else {
		now := time.Now()
		for i := range items {
			items[i].SLA = kdzs.ComputeRefundSLA(&items[i], nil, now)
		}
	}
	countUrgentSLAStats(stats, items)
	return nil
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
	if q.Scenario != "" && len(rq.AfterSaleStatusList) == 0 && q.Sid == "" && q.Tid == "" && q.SysTid == "" {
		switch q.Scenario {
		case "confirm_receive", "return_signed", "pickup_pending":
			rq.AfterSaleStatusList = []string{"WAIT_SELLER_CONFIRM_RECEIVE"}
		case "wait_agree", "refund_only":
			rq.AfterSaleStatusList = []string{"WAIT_SELLER_AGREE"}
		case "wait_return":
			rq.AfterSaleStatusList = []string{"WAIT_BUYER_RETURN_ITEM"}
		case "refund_success":
			rq.AfterSaleStatusList = []string{"REFUND_SUCCESS"}
			rq.AfterSaleTypeList = []int{kdzs.AfterSaleTypeReturnRefund}
		case "seller_refuse":
			rq.AfterSaleStatusList = []string{kdzs.SellerRefuseQueryStatus()}
		case "refund_close_with_sid":
			rq.AfterSaleStatusList = []string{"REFUND_CLOSE"}
		}
		if q.Scenario == "refund_only" {
			rq.AfterSaleTypeList = []int{1}
		}
		if q.Scenario == "exchange" {
			rq.AfterSaleTypeList = []int{3}
		}
		if q.Scenario == "wait_send_exchange" {
			rq.AfterSaleStatusList = []string{"WAIT_SEND_EXCHANGE_ITEM"}
		}
	}
	return rq
}

func filterRefundsByScenario(items []kdzs.RefundItem, scenario string) []kdzs.RefundItem {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return items
	}
	out := make([]kdzs.RefundItem, 0, len(items))
	for _, item := range items {
		if kdzs.MatchRefundScenario(item, scenario) {
			out = append(out, item)
		}
	}
	return out
}

func filterRefundsWithReturnLogistics(items []kdzs.RefundItem) []kdzs.RefundItem {
	out := make([]kdzs.RefundItem, 0, len(items))
	for _, item := range items {
		if kdzs.HasReturnLogistics(item) {
			out = append(out, item)
		}
	}
	return out
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

	sendExQ := base
	sendExQ.AfterSaleStatusList = []string{"WAIT_SEND_EXCHANGE_ITEM"}
	if res, err := s.session.QueryRefunds(ctx, sendExQ); err == nil {
		stats.WaitSendExchange = res.Total
	}

	stats.WaitBuyerReturn = countStatus("WAIT_BUYER_RETURN_ITEM")
	stats.SellerRefuse = countStatus(kdzs.SellerRefuseQueryStatus())
	successQ := base
	successQ.AfterSaleStatusList = []string{"REFUND_SUCCESS"}
	successQ.AfterSaleTypeList = []int{kdzs.AfterSaleTypeReturnRefund}
	if res, err := s.session.QueryRefunds(ctx, successQ); err == nil {
		stats.RefundSuccess = res.Total
	}
	closeQ := base
	closeQ.AfterSaleStatusList = []string{"REFUND_CLOSE"}
	if closeItems, _, err := s.session.QueryAllRefunds(ctx, closeQ); err == nil {
		stats.RefundCloseWithSid = len(filterRefundsWithReturnLogistics(closeItems))
	}

	// 待确认收货：拉全量并查物流，统计签收/待取件（与场景 Tab 一致）。
	if confirmItems, err := s.loadConfirmReceiveWithSLA(ctx, platform, base); err == nil {
		applyConfirmReceiveSLAStats(stats, confirmItems)
	}

	// 时效紧迫：全量扫描所有有倒计时的售后状态。
	for _, status := range kdzs.AfterSaleStatusesWithSLADeadline {
		withLogistics := status == "WAIT_SELLER_CONFIRM_RECEIVE"
		if err := s.countUrgentStatsForStatus(ctx, platform, base, status, withLogistics, stats); err != nil {
			return stats, err
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
