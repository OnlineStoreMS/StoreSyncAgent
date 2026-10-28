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
	"storesyncagent/internal/repo"
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

type snapshotMark struct {
	running    bool
	done       bool
	err        string
	shopsTotal int
	shopsDone  int
	itemCounts map[string]int
}

func (s *SyncService) ListLocalItems(q ItemQuery) (*ItemListView, error) {
	if s.syncedItemRepo == nil {
		return nil, fmt.Errorf("本地商品库不可用")
	}
	if err := s.ensureDefaultActiveAccount(); err != nil {
		return nil, err
	}
	pageNo := q.PageNo
	if pageNo < 1 {
		pageNo = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	rows, total, err := s.syncedItemRepo.List(s.tenantID, s.activeAccountID, repo.SyncedItemListQuery{
		Platform:       q.Platform,
		ShopIDs:        splitShopIDs(q.ShopID),
		Type:           q.Type,
		Title:          q.Title,
		ShortTitle:     q.ShortTitle,
		ItemIDs:        splitShopIDs(q.ItemIDs),
		OuterID:        q.OuterID,
		ProductNumLike: q.ProductNumLike,
		SkuTexts:       nonEmpty(q.SpuPropertiesName, q.SkuIDs, q.SkuOuterID, q.SkuShortTitle),
		PageNo:         pageNo,
		PageSize:       pageSize,
	})
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
		Total:    int(total),
		PageNo:   pageNo,
		PageSize: pageSize,
		Items:    items,
		Platform: strings.ToUpper(strings.TrimSpace(q.Platform)),
	}, nil
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func (s *SyncService) ProductSyncProgress(_ context.Context, platform string) (*kdzs.SyncProgress, error) {
	if err := s.ensureDefaultActiveAccount(); err != nil {
		return nil, err
	}
	key := snapshotKey(s.activeAccountID, platform)
	s.mu.Lock()
	st := s.snapshotState[key]
	counts := copyItemCounts(st.itemCounts)
	s.mu.Unlock()

	progress := &kdzs.SyncProgress{
		Finish:        st.done && !st.running,
		ErrorMessage:  st.err,
		SyncItemCount: counts,
	}
	if st.shopsTotal > 0 {
		progress.Process = st.shopsDone * 100 / st.shopsTotal
	}
	if st.running {
		progress.Finish = false
	}
	if st.err != "" {
		progress.Finish = true
	}
	return progress, nil
}

func snapshotKey(account, platform string) string {
	return account + "\x00" + strings.ToUpper(strings.TrimSpace(platform))
}

func copyItemCounts(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]int, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func (s *SyncService) updateSnapshot(account, platform string, fn func(*snapshotMark)) {
	key := snapshotKey(account, platform)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snapshotState == nil {
		s.snapshotState = map[string]snapshotMark{}
	}
	st := s.snapshotState[key]
	fn(&st)
	s.snapshotState[key] = st
}

func (s *SyncService) startSyncedItemSnapshot(platform string, shopIDs []string) {
	if s.syncedItemRepo == nil {
		return
	}
	account := s.activeAccountID
	if account == "" || platform == "" {
		return
	}
	key := snapshotKey(account, platform)
	s.mu.Lock()
	if s.snapshotState == nil {
		s.snapshotState = map[string]snapshotMark{}
	}
	if s.snapshotState[key].running {
		s.mu.Unlock()
		return
	}
	s.snapshotState[key] = snapshotMark{running: true, itemCounts: map[string]int{}}
	s.mu.Unlock()

	ids := append([]string(nil), shopIDs...)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
		defer cancel()
		err := s.snapshotSyncedItems(ctx, account, platform, ids)
		s.updateSnapshot(account, platform, func(st *snapshotMark) {
			st.running = false
			st.done = err == nil
			st.err = errorText(err)
		})
		if err != nil {
			log.Printf("[synced-items] import account=%s platform=%s: %v", account, platform, err)
			return
		}
		log.Printf("[synced-items] import account=%s platform=%s shops=%d done", account, platform, len(ids))
	}()
}

func (s *SyncService) snapshotSyncedItems(ctx context.Context, account, platform string, shopIDs []string) error {
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
	s.updateSnapshot(account, platform, func(st *snapshotMark) {
		st.running = true
		st.done = false
		st.err = ""
		st.shopsTotal = len(targets)
		st.shopsDone = 0
		st.itemCounts = map[string]int{}
	})
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
				Platform:      platform,
				ShopName:      item.ShopName,
				ItemID:        item.ItemID,
				ApproveStatus: item.ApproveStatus,
				Title:         item.Title,
				ShortTitle:    item.ShortTitle,
				OuterID:       item.OuterID,
				ProductNum:    item.ProductNum,
				SearchText:    itemSearchText(item),
				SkuSearchText: skuSearchText(item),
				Payload:       string(payload),
				SyncedAt:      now,
			})
		}
		if err := s.syncedItemRepo.ReplaceShop(s.tenantID, account, shopID, rows); err != nil {
			return err
		}
		saved += len(rows)
		shopSaved := len(rows)
		s.updateSnapshot(account, platform, func(st *snapshotMark) {
			if st.itemCounts == nil {
				st.itemCounts = map[string]int{}
			}
			st.itemCounts[shopID] = shopSaved
			st.shopsDone++
		})
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

func skuSearchText(item ItemView) string {
	parts := make([]string, 0, len(item.Skus)*5)
	for _, sku := range item.Skus {
		parts = append(parts, sku.PropertiesName, sku.SkuID, sku.OuterID, sku.ShortTitle, sku.ProductNum)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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
