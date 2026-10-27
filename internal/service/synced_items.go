package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"storesyncagent/internal/kdzs"
	"storesyncagent/internal/model"
)

const syncedItemSearchLimit = 500

func (s *SyncService) ListSyncedItems(shopID string, keywords []string) (*ItemListView, error) {
	if s.syncedItemRepo == nil {
		return nil, fmt.Errorf("本地商品库不可用")
	}
	if err := s.ensureDefaultActiveAccount(); err != nil {
		return nil, err
	}
	shopID = strings.TrimSpace(shopID)
	if shopID == "" {
		return nil, fmt.Errorf("shopId is required")
	}
	keywords = normalizeKeywords(keywords)
	if len(keywords) == 0 {
		return nil, fmt.Errorf("请输入搜索关键字")
	}
	account := s.activeAccountID
	n, err := s.syncedItemRepo.CountShop(s.tenantID, account, shopID)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("该店铺还没有本地同步商品，请先在商品列表点击「同步商品」")
	}
	rows, err := s.syncedItemRepo.Search(s.tenantID, account, shopID, keywords, syncedItemSearchLimit)
	if err != nil {
		return nil, err
	}
	items := make([]ItemView, 0, len(rows))
	for _, row := range rows {
		var item ItemView
		if err := json.Unmarshal([]byte(row.Payload), &item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return &ItemListView{
		Total:    len(items),
		PageNo:   1,
		PageSize: len(items),
		Items:    items,
	}, nil
}

func (s *SyncService) startSyncedItemSnapshot(platform string, shopIDs []string) {
	if s.syncedItemRepo == nil {
		return
	}
	account := s.activeAccountID
	if account == "" || platform == "" {
		return
	}
	key := account + "\x00" + platform
	s.mu.Lock()
	if s.snapshotRunning == nil {
		s.snapshotRunning = map[string]struct{}{}
	}
	if _, ok := s.snapshotRunning[key]; ok {
		s.mu.Unlock()
		return
	}
	s.snapshotRunning[key] = struct{}{}
	s.mu.Unlock()

	ids := append([]string(nil), shopIDs...)
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.snapshotRunning, key)
			s.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
		defer cancel()
		if err := s.snapshotSyncedItems(ctx, account, platform, ids); err != nil {
			log.Printf("[synced-items] snapshot account=%s platform=%s: %v", account, platform, err)
			return
		}
		log.Printf("[synced-items] snapshot account=%s platform=%s shops=%d done", account, platform, len(ids))
	}()
}

func (s *SyncService) snapshotSyncedItems(ctx context.Context, account, platform string, shopIDs []string) error {
	if err := s.waitItemSyncSettled(ctx, platform); err != nil {
		return err
	}
	if s.activeAccountID != account {
		return fmt.Errorf("快递助手账号已切换，取消写入本地商品")
	}
	if err := s.ensureLogin(ctx); err != nil {
		return err
	}
	shops, err := s.client.ListEcommerceShops(ctx)
	if err != nil {
		return err
	}
	nameByID := map[string]string{}
	allowed := map[string]struct{}{}
	for _, id := range shopIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			allowed[id] = struct{}{}
		}
	}
	targets := make([]string, 0)
	for _, shop := range shops {
		if shop.Platform != platform {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[shop.MallUserID]; !ok {
				continue
			}
		}
		nameByID[shop.MallUserID] = shop.MallUserName
		targets = append(targets, shop.MallUserID)
	}
	if len(targets) == 0 {
		targets = shopIDs
	}
	var saved int
	for _, shopID := range targets {
		if s.activeAccountID != account {
			return fmt.Errorf("快递助手账号已切换，取消写入本地商品")
		}
		items, err := s.listAllShopItems(ctx, platform, shopID)
		if err != nil {
			return fmt.Errorf("shop %s: %w", shopID, err)
		}
		rows := make([]model.SyncedShopItem, 0, len(items))
		seen := make(map[string]struct{}, len(items))
		now := time.Now()
		for _, item := range items {
			if item.ItemID == "" {
				continue
			}
			if _, ok := seen[item.ItemID]; ok {
				continue
			}
			seen[item.ItemID] = struct{}{}
			if item.ShopID == "" {
				item.ShopID = shopID
			}
			if item.ShopName == "" {
				item.ShopName = nameByID[shopID]
			}
			if item.Platform == "" {
				item.Platform = platform
			}
			payload, err := json.Marshal(item)
			if err != nil {
				return err
			}
			rows = append(rows, model.SyncedShopItem{
				Platform:   platform,
				ShopName:   item.ShopName,
				ItemID:     item.ItemID,
				SearchText: itemSearchText(item),
				Payload:    string(payload),
				SyncedAt:   now,
			})
		}
		if err := s.syncedItemRepo.ReplaceShop(s.tenantID, account, shopID, rows); err != nil {
			return err
		}
		saved += len(rows)
	}
	log.Printf("[synced-items] saved account=%s platform=%s items=%d", account, platform, saved)
	return nil
}

func (s *SyncService) listAllShopItems(ctx context.Context, platform, shopID string) ([]ItemView, error) {
	all := make([]ItemView, 0)
	for page := 1; ; page++ {
		result, err := s.session.ListShopItems(ctx, platform, kdzs.ItemListQuery{
			PageNo:     page,
			PageSize:   50,
			ShopIDList: []string{shopID},
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
		if len(result.List) == 0 || len(result.List) < 50 {
			break
		}
		if result.Count > 0 && page*50 >= result.Count {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return all, nil
}

func (s *SyncService) waitItemSyncSettled(ctx context.Context, platform string) error {
	if err := s.ensureLogin(ctx); err != nil {
		return err
	}
	initial, err := s.session.GetItemSyncProgress(ctx, platform)
	if err != nil {
		return err
	}
	initialStamp := syncProgressStamp(initial)
	sawRunning := initial != nil && !initial.Finish
	started := time.Now()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if err := s.ensureLogin(ctx); err != nil {
			return err
		}
		progress, err := s.session.GetItemSyncProgress(ctx, platform)
		if err != nil {
			return err
		}
		if progress == nil {
			return fmt.Errorf("同步进度为空")
		}
		if msg := strings.TrimSpace(progress.ErrorMessage); msg != "" && !progress.Finish {
			return fmt.Errorf("%s", msg)
		}
		if !progress.Finish {
			sawRunning = true
			continue
		}
		if msg := strings.TrimSpace(progress.ErrorMessage); msg != "" {
			return fmt.Errorf("%s", msg)
		}
		if sawRunning || syncProgressStamp(progress) != initialStamp {
			return nil
		}
		if time.Since(started) >= 20*time.Second {
			return nil
		}
	}
}

func syncProgressStamp(p *kdzs.SyncProgress) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%t|%d|%s|%s", p.Finish, p.Process, p.FinishSyncDate, p.ErrorMessage)
}

func itemSearchText(item ItemView) string {
	parts := []string{item.Title, item.ShortTitle, item.OuterID, item.ItemID, item.ProductNum}
	for _, sku := range item.Skus {
		parts = append(parts, sku.PropertiesName, sku.SkuID, sku.OuterID, sku.ShortTitle, sku.ProductNum)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func normalizeKeywords(keywords []string) []string {
	out := make([]string, 0, len(keywords))
	seen := map[string]struct{}{}
	for _, raw := range keywords {
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == '，' || r == ';' || r == '；'
		}) {
			t := strings.TrimSpace(part)
			if t == "" {
				continue
			}
			key := strings.ToLower(t)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}
