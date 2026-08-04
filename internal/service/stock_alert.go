package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"storesyncagent/internal/feishu"
	"storesyncagent/internal/kdzs"
	"storesyncagent/internal/store"
)

type StockAlertConfigView struct {
	Enabled             bool     `json:"enabled"`
	WebhookURL          string   `json:"webhookUrl"`
	SecretSet           bool     `json:"secretSet"`
	Platform            string   `json:"platform"`
	ShopIDs             []string `json:"shopIds"`
	StockThreshold      int      `json:"stockThreshold"`
	CheckLevel          string   `json:"checkLevel"`
	OnlyOnsale          bool     `json:"onlyOnsale"`
	PollIntervalMinutes int      `json:"pollIntervalMinutes"`
}

type StockAlertView struct {
	Config StockAlertConfigView  `json:"config"`
	State  store.StockAlertState `json:"state"`
	Shops  []ShopView            `json:"shops"`
}

type StockAlertHit struct {
	ItemID             string `json:"itemId"`
	Title              string `json:"title"`
	ShortTitle         string `json:"shortTitle,omitempty"`
	OuterID            string `json:"outerId,omitempty"`
	PicURL             string `json:"picUrl,omitempty"`
	Platform           string `json:"platform"`
	PlatformName       string `json:"platformName"`
	ShopID             string `json:"shopId"`
	ShopName           string `json:"shopName"`
	ApproveStatus      string `json:"approveStatus,omitempty"`
	ApproveStatusLabel string `json:"approveStatusLabel,omitempty"`
	SkuID              string `json:"skuId,omitempty"`
	PropertiesName     string `json:"propertiesName,omitempty"`
	SkuOuterID         string `json:"skuOuterId,omitempty"`
	Quantity           int    `json:"quantity"`
	Level              string `json:"level"` // spu | sku
}

type StockAlertScanResult struct {
	Threshold int             `json:"threshold"`
	Total     int             `json:"total"`
	Items     []StockAlertHit `json:"items"`
	Scanned   int             `json:"scanned"`
}

type StockAlertRunResult struct {
	Sent    int `json:"sent"`
	Skipped int `json:"skipped"`
	Alerted int `json:"alerted"`
	Scanned int `json:"scanned"`
}

func (s *SyncService) GetStockAlertView(ctx context.Context) (*StockAlertView, error) {
	data, err := s.stockAlertRepo.Load(s.tenantID)
	if err != nil {
		return nil, err
	}
	shops, _ := s.ListShops(ctx)
	return s.buildStockAlertView(data, shops), nil
}

func (s *SyncService) buildStockAlertView(data store.StockAlertData, shops []ShopView) *StockAlertView {
	return &StockAlertView{
		Config: toStockAlertConfigView(data.Config),
		State:  data.State,
		Shops:  shops,
	}
}

func toStockAlertConfigView(cfg store.StockAlertConfig) StockAlertConfigView {
	return StockAlertConfigView{
		Enabled:             cfg.Enabled,
		WebhookURL:          cfg.WebhookURL,
		SecretSet:           cfg.Secret != "",
		Platform:            cfg.Platform,
		ShopIDs:             append([]string(nil), cfg.ShopIDs...),
		StockThreshold:      cfg.StockThreshold,
		CheckLevel:          cfg.CheckLevel,
		OnlyOnsale:          cfg.OnlyOnsale,
		PollIntervalMinutes: cfg.PollIntervalMinutes,
	}
}

func (s *SyncService) SaveStockAlertConfig(ctx context.Context, in store.StockAlertConfig) (*StockAlertView, error) {
	store.NormalizeStockAlertConfig(&in)
	if in.Enabled && strings.TrimSpace(in.WebhookURL) == "" {
		return nil, fmt.Errorf("启用预警时请配置飞书 Webhook 地址")
	}
	data, err := s.stockAlertRepo.SaveConfig(s.tenantID, in)
	if err != nil {
		return nil, err
	}
	shops, _ := s.ListShops(ctx)
	return s.buildStockAlertView(data, shops), nil
}

func (s *SyncService) ResetStockAlertState(ctx context.Context) (*StockAlertView, int, error) {
	cleared, err := s.stockAlertRepo.ResetState(s.tenantID)
	if err != nil {
		return nil, 0, err
	}
	view, err := s.GetStockAlertView(ctx)
	return view, cleared, err
}

func (s *SyncService) TestStockAlert(ctx context.Context, text string) error {
	data, err := s.stockAlertRepo.Load(s.tenantID)
	if err != nil {
		return err
	}
	cfg := data.Config
	if cfg.WebhookURL == "" {
		return fmt.Errorf("请先配置 Webhook 地址")
	}
	if text == "" {
		text = "线上商品库存预警测试消息"
	}
	card := feishu.InteractiveCard{
		Title:    "StoreSyncAgent · 库存预警测试",
		Template: "orange",
		Markdown: fmt.Sprintf("**说明：** %s\n\n<font color='grey'>若能看到本条卡片，说明库存预警 Webhook 配置正确。</font>", escapeLarkMD(text)),
	}
	return s.feishuClient.SendInteractiveCard(ctx, cfg.WebhookURL, cfg.Secret, card)
}

func (s *SyncService) ScanStockAlerts(ctx context.Context) (*StockAlertScanResult, error) {
	data, err := s.stockAlertRepo.Load(s.tenantID)
	if err != nil {
		return nil, err
	}
	return s.scanStockAlertsWithConfig(ctx, data.Config)
}

func (s *SyncService) scanStockAlertsWithConfig(ctx context.Context, cfg store.StockAlertConfig) (*StockAlertScanResult, error) {
	store.NormalizeStockAlertConfig(&cfg)
	items, err := s.listAllItemsForStockAlert(ctx, cfg)
	if err != nil {
		return nil, err
	}
	hits := collectStockAlertHits(items, cfg)
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Quantity != hits[j].Quantity {
			return hits[i].Quantity < hits[j].Quantity
		}
		return hits[i].Title < hits[j].Title
	})
	return &StockAlertScanResult{
		Threshold: cfg.StockThreshold,
		Total:     len(hits),
		Items:     hits,
		Scanned:   len(items),
	}, nil
}

func (s *SyncService) listAllItemsForStockAlert(ctx context.Context, cfg store.StockAlertConfig) ([]ItemView, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	shops, err := s.client.ListEcommerceShops(ctx)
	if err != nil {
		return nil, err
	}

	shopFilter := map[string]struct{}{}
	for _, id := range cfg.ShopIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			shopFilter[id] = struct{}{}
		}
	}

	platform := cfg.Platform
	if platform == "" {
		platform = "FXG"
	}
	shopIDList := make([]string, 0)
	for _, shop := range shops {
		if shop.Platform != platform {
			continue
		}
		if len(shopFilter) > 0 {
			if _, ok := shopFilter[shop.MallUserID]; !ok {
				continue
			}
		}
		shopIDList = append(shopIDList, shop.MallUserID)
	}
	if len(shopIDList) == 0 {
		return []ItemView{}, nil
	}

	itemType := ""
	if cfg.OnlyOnsale {
		itemType = "onsale"
	}

	all := make([]ItemView, 0)
	for page := 1; ; page++ {
		result, err := s.session.ListShopItems(ctx, platform, kdzs.ItemListQuery{
			PageNo:     page,
			PageSize:   50,
			ShopIDList: shopIDList,
			Type:       itemType,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range result.List {
			if isPlatformDeletedItem(item) {
				continue
			}
			all = append(all, toItemView(item))
		}
		if len(result.List) == 0 {
			break
		}
		if result.Count > 0 && page*50 >= result.Count {
			break
		}
		if len(result.List) < 50 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(900 * time.Millisecond):
		}
	}
	return all, nil
}

func collectStockAlertHits(items []ItemView, cfg store.StockAlertConfig) []StockAlertHit {
	threshold := cfg.StockThreshold
	level := cfg.CheckLevel
	hits := make([]StockAlertHit, 0)
	for _, item := range items {
		base := StockAlertHit{
			ItemID:             item.ItemID,
			Title:              item.Title,
			ShortTitle:         item.ShortTitle,
			OuterID:            item.OuterID,
			PicURL:             item.PicURL,
			Platform:           item.Platform,
			PlatformName:       item.PlatformName,
			ShopID:             item.ShopID,
			ShopName:           item.ShopName,
			ApproveStatus:      item.ApproveStatus,
			ApproveStatusLabel: item.ApproveStatusLabel,
		}
		checkSPU := level == "spu" || level == "both"
		checkSKU := level == "sku" || level == "both" || level == ""
		if checkSPU && item.Stock <= threshold {
			hit := base
			hit.Quantity = item.Stock
			hit.Level = "spu"
			hits = append(hits, hit)
		}
		if checkSKU {
			if len(item.Skus) == 0 {
				if !checkSPU && item.Stock <= threshold {
					hit := base
					hit.Quantity = item.Stock
					hit.Level = "spu"
					hits = append(hits, hit)
				}
				continue
			}
			for _, sku := range item.Skus {
				if sku.Quantity > threshold {
					continue
				}
				hit := base
				hit.SkuID = sku.SkuID
				hit.PropertiesName = sku.PropertiesName
				hit.SkuOuterID = sku.OuterID
				hit.Quantity = sku.Quantity
				hit.Level = "sku"
				if sku.PicURL != "" {
					hit.PicURL = sku.PicURL
				}
				hits = append(hits, hit)
			}
		}
	}
	return hits
}

func stockAlertDedupKey(hit StockAlertHit) string {
	if hit.Level == "sku" && hit.SkuID != "" {
		return fmt.Sprintf("%s:%s:%s:%s", hit.Platform, hit.ShopID, hit.ItemID, hit.SkuID)
	}
	return fmt.Sprintf("%s:%s:%s", hit.Platform, hit.ShopID, hit.ItemID)
}

func (s *SyncService) RunStockAlertPoll(ctx context.Context) (*StockAlertRunResult, error) {
	data, err := s.stockAlertRepo.Load(s.tenantID)
	if err != nil {
		return nil, err
	}
	cfg := data.Config
	result := &StockAlertRunResult{}
	now := time.Now()
	runAt := now.Format("2006-01-02 15:04:05")

	updateState := func(ok bool, sent, alerted int, errMsg string, notified map[string]string) {
		_ = s.stockAlertRepo.UpdateState(s.tenantID, func(st *store.StockAlertState) error {
			st.LastRunAt = runAt
			st.LastRunOK = ok
			st.LastError = errMsg
			st.LastSentCount = sent
			st.LastAlertCount = alerted
			if notified != nil {
				st.Notified = notified
			}
			return nil
		})
	}

	if !cfg.Enabled {
		updateState(true, 0, 0, "", data.State.Notified)
		return result, nil
	}
	if cfg.WebhookURL == "" {
		err := fmt.Errorf("webhook url is empty")
		updateState(false, 0, 0, err.Error(), nil)
		return nil, err
	}

	scan, err := s.scanStockAlertsWithConfig(ctx, cfg)
	if err != nil {
		updateState(false, 0, 0, err.Error(), nil)
		return nil, err
	}
	result.Scanned = scan.Scanned
	result.Alerted = scan.Total

	notified := data.State.Notified
	if notified == nil {
		notified = map[string]string{}
	}
	activeKeys := map[string]struct{}{}
	newHits := make([]StockAlertHit, 0)
	for _, hit := range scan.Items {
		key := stockAlertDedupKey(hit)
		activeKeys[key] = struct{}{}
		if _, ok := notified[key]; ok {
			result.Skipped++
			continue
		}
		newHits = append(newHits, hit)
	}
	// Drop recovered items from dedup map
	for key := range notified {
		if _, ok := activeKeys[key]; !ok {
			delete(notified, key)
		}
	}

	if len(newHits) == 0 {
		updateState(true, 0, scan.Total, "", notified)
		return result, nil
	}

	if err := s.sendStockAlertCard(ctx, cfg, scan.Threshold, newHits, scan.Total); err != nil {
		updateState(false, 0, scan.Total, err.Error(), notified)
		return nil, err
	}
	for _, hit := range newHits {
		notified[stockAlertDedupKey(hit)] = runAt
	}
	result.Sent = 1
	updateState(true, 1, scan.Total, "", notified)
	return result, nil
}

func (s *SyncService) sendStockAlertCard(ctx context.Context, cfg store.StockAlertConfig, threshold int, hits []StockAlertHit, totalAlerted int) error {
	const maxLines = 25
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**预警阈值：** ≤ %d\n", threshold))
	b.WriteString(fmt.Sprintf("**平台：** %s\n", escapeLarkMD(kdzs.PlatformLabel(cfg.Platform))))
	b.WriteString(fmt.Sprintf("**本次新增：** %d 条（当前共 %d 条低库存）\n\n", len(hits), totalAlerted))
	show := hits
	if len(show) > maxLines {
		show = show[:maxLines]
	}
	for i, hit := range show {
		spec := hit.PropertiesName
		if spec == "" {
			spec = "整款"
		}
		shop := hit.ShopName
		if shop == "" {
			shop = hit.ShopID
		}
		title := hit.Title
		if len([]rune(title)) > 40 {
			title = string([]rune(title)[:40]) + "…"
		}
		b.WriteString(fmt.Sprintf("%d. **%s** / %s\n库存 **%d** · %s\n", i+1, escapeLarkMD(title), escapeLarkMD(spec), hit.Quantity, escapeLarkMD(shop)))
	}
	if len(hits) > maxLines {
		b.WriteString(fmt.Sprintf("\n<font color='grey'>另有 %d 条未展开，请在 StoreSyncAgent「库存预警」查看完整列表。</font>", len(hits)-maxLines))
	}
	card := feishu.InteractiveCard{
		Title:    "StoreSyncAgent · 线上商品库存预警",
		Template: "orange",
		Markdown: b.String(),
	}
	return s.feishuClient.SendInteractiveCard(ctx, cfg.WebhookURL, cfg.Secret, card)
}

func (s *SyncService) StockAlertEnabled() bool {
	data, err := s.stockAlertRepo.Load(s.tenantID)
	if err != nil {
		return false
	}
	return data.Config.Enabled
}

func (s *SyncService) StockAlertPollInterval() time.Duration {
	data, err := s.stockAlertRepo.Load(s.tenantID)
	if err != nil {
		return 60 * time.Minute
	}
	mins := data.Config.PollIntervalMinutes
	if mins < 15 {
		mins = 15
	}
	return time.Duration(mins) * time.Minute
}
