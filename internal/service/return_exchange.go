package service

import (
	"context"
	"fmt"
	"strings"

	"storesyncagent/internal/kdzs"
	"storesyncagent/internal/store"
)

type OrderLookupView struct {
	Found                 bool              `json:"found"`
	OrderNo               string            `json:"orderNo"`
	Platform              string            `json:"platform,omitempty"`
	SysTid                string            `json:"sysTid,omitempty"`
	ShopName              string            `json:"shopName,omitempty"`
	OrderBuyerNick        string            `json:"orderBuyerNick,omitempty"`
	GoodsSummary          string            `json:"goodsSummary,omitempty"`
	GoodsTitle            string            `json:"goodsTitle,omitempty"`
	Goods                 []kdzs.TradeGoods `json:"goods,omitempty"`
	MemoNotes             string            `json:"memoNotes,omitempty"`
	OriginalRecipientInfo string            `json:"originalRecipientInfo,omitempty"`
	Payment               float64           `json:"payment,omitempty"`
	PayTime               string            `json:"payTime,omitempty"`
	StatusText            string            `json:"statusText,omitempty"`
	Source                string            `json:"source,omitempty"`
}

func (s *SyncService) LookupOrderByTid(ctx context.Context, platform, tid string) (*OrderLookupView, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	tid = strings.TrimSpace(tid)
	if tid == "" {
		return nil, fmt.Errorf("orderNo is required")
	}
	if platform == "" {
		platform = "FXG"
	}

	view := &OrderLookupView{Found: false, OrderNo: tid, Platform: platform}

	item, tradeStatus, err := s.session.LookupTradeByTid(ctx, platform, tid)
	if err != nil {
		return nil, err
	}
	if item != nil {
		s.enrichOrderLookupFromTrade(ctx, view, item, platform, tradeStatus)
		view.Source = "trade"
		return view, nil
	}

	refundResult, err := s.session.QueryRefunds(ctx, kdzs.RefundQuery{
		Platform: platform,
		Tid:      tid,
		PageNo:   1,
		PageSize: 5,
	})
	if err != nil {
		return nil, err
	}
	for i := range refundResult.Items {
		ref := &refundResult.Items[i]
		if ref.Tid != "" && ref.Tid != tid {
			continue
		}
		view.Found = true
		view.Source = "refund"
		view.OrderBuyerNick = ref.BuyerNick
		view.ShopName = ref.ShopName
		view.SysTid = ref.SysTid
		view.StatusText = ref.AfterSaleStatusText
		if len(ref.Goods) > 0 {
			view.GoodsTitle = ref.Goods[0].Title
			goods := make([]kdzs.TradeGoods, 0, len(ref.Goods))
			for _, g := range ref.Goods {
				goods = append(goods, kdzs.TradeGoods{
					Title:   g.Title,
					SkuName: g.SkuName,
					PicURL:  g.PicURL,
					Num:     g.Num,
				})
			}
			view.Goods = goods
			view.GoodsSummary = joinGoodsSummary(goods)
		}
		if ref.SysTid != "" {
			for _, status := range []string{"shipped", "completed", "wait_send", "wait_audit", kdzs.DefaultTradeStatus()} {
				details, err := s.session.FetchTradeDetails(ctx, platform, status, []string{ref.SysTid})
				if err != nil || len(details) == 0 {
					continue
				}
				if parsed := kdzs.ParseTradeItemFromJSON(details[0], platform); parsed != nil {
					s.enrichOrderLookupFromTrade(ctx, view, parsed, platform, status)
					view.Source = "refund+trade"
					break
				}
			}
		}
		return view, nil
	}

	return view, nil
}

func (s *SyncService) enrichOrderLookupFromTrade(ctx context.Context, view *OrderLookupView, item *kdzs.TradeListItem, platform, tradeStatus string) {
	if view == nil || item == nil {
		return
	}
	view.Found = true
	view.OrderBuyerNick = item.BuyerNick
	view.ShopName = item.ShopName
	view.Goods = item.Goods
	view.Payment = item.Payment
	view.PayTime = item.PayTime
	view.StatusText = firstNonEmpty(item.StatusText, kdzs.TradeStatusLabel(item.TradeStatus))
	view.MemoNotes = joinMemoNotes(item)
	if len(item.SysTids) > 0 {
		view.SysTid = item.SysTids[0]
	}
	if len(item.Goods) > 0 {
		view.GoodsTitle = item.Goods[0].Title
		view.GoodsSummary = joinGoodsSummary(item.Goods)
	}

	if len(item.SysTids) > 0 {
		statuses := []string{tradeStatus, "shipped", "completed", "wait_send", "wait_audit", kdzs.DefaultTradeStatus()}
		seen := map[string]struct{}{}
		for _, status := range statuses {
			if status == "" {
				continue
			}
			if _, ok := seen[status]; ok {
				continue
			}
			seen[status] = struct{}{}
			metaBySysTid, err := s.session.FetchDecryptMetaBySysTids(ctx, platform, status, item.SysTids)
			if err != nil || len(metaBySysTid) == 0 {
				continue
			}
			if meta, ok := metaBySysTid[item.SysTids[0]]; ok {
				if decrypted, err := s.session.DecodeTradeReceiver(ctx, platform, meta); err == nil {
					view.OriginalRecipientInfo = decrypted.FormattedText
					if view.OriginalRecipientInfo == "" {
						view.OriginalRecipientInfo = formatReceiverFallback(decrypted)
					}
					break
				}
			}
		}
	}
	if view.OriginalRecipientInfo == "" {
		view.OriginalRecipientInfo = firstNonEmpty(item.FormattedReceiver, formatTradeReceiver(item))
	}
}

func joinGoodsSummary(goods []kdzs.TradeGoods) string {
	parts := make([]string, 0, len(goods))
	for _, g := range goods {
		title := strings.TrimSpace(g.Title)
		sku := strings.TrimSpace(g.SkuName)
		switch {
		case title != "" && sku != "":
			parts = append(parts, fmt.Sprintf("%s（%s）×%d", title, sku, maxInt(g.Num, 1)))
		case title != "":
			parts = append(parts, fmt.Sprintf("%s ×%d", title, maxInt(g.Num, 1)))
		case sku != "":
			parts = append(parts, sku)
		}
	}
	return strings.Join(parts, "；")
}

func joinMemoNotes(item *kdzs.TradeListItem) string {
	if item == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if v := strings.TrimSpace(item.BuyerMemo); v != "" {
		parts = append(parts, "买家留言: "+v)
	}
	if v := strings.TrimSpace(item.SellerMemo); v != "" {
		parts = append(parts, "卖家备注: "+v)
	}
	if v := strings.TrimSpace(item.FenFaMemo); v != "" {
		parts = append(parts, "分发备注: "+v)
	}
	if v := strings.TrimSpace(item.PrinterMemo); v != "" {
		parts = append(parts, "打单备注: "+v)
	}
	return strings.Join(parts, "\n")
}

func formatTradeReceiver(item *kdzs.TradeListItem) string {
	if item == nil {
		return ""
	}
	var parts []string
	if item.ReceiverName != "" {
		parts = append(parts, "收货人: "+item.ReceiverName)
	}
	if item.ReceiverMobile != "" {
		parts = append(parts, "手机号码: "+item.ReceiverMobile)
	}
	if item.ReceiverAddress != "" {
		parts = append(parts, "详细地址: "+item.ReceiverAddress)
	}
	return strings.Join(parts, ", ")
}

func formatReceiverFallback(d *kdzs.DecryptedReceiver) string {
	if d == nil {
		return ""
	}
	var parts []string
	if d.ReceiverName != "" {
		parts = append(parts, "收货人: "+d.ReceiverName)
	}
	if d.ReceiverMobile != "" {
		parts = append(parts, "手机号码: "+d.ReceiverMobile)
	}
	if d.ReceiverAddress != "" {
		parts = append(parts, "详细地址: "+d.ReceiverAddress)
	}
	return strings.Join(parts, ", ")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func maxInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func (s *SyncService) ListReturnExchanges() ([]store.ReturnExchangeRecord, error) {
	if s.returnExchangeStore == nil {
		return nil, fmt.Errorf("return exchange store not configured")
	}
	return s.returnExchangeStore.List()
}

func (s *SyncService) CreateReturnExchange(rec store.ReturnExchangeRecord) (store.ReturnExchangeRecord, error) {
	if s.returnExchangeStore == nil {
		return store.ReturnExchangeRecord{}, fmt.Errorf("return exchange store not configured")
	}
	return s.returnExchangeStore.Create(rec)
}

func (s *SyncService) UpdateReturnExchange(id string, rec store.ReturnExchangeRecord) (store.ReturnExchangeRecord, error) {
	if s.returnExchangeStore == nil {
		return store.ReturnExchangeRecord{}, fmt.Errorf("return exchange store not configured")
	}
	return s.returnExchangeStore.Update(id, rec)
}

func (s *SyncService) DeleteReturnExchange(id string) error {
	if s.returnExchangeStore == nil {
		return fmt.Errorf("return exchange store not configured")
	}
	return s.returnExchangeStore.Delete(id)
}
