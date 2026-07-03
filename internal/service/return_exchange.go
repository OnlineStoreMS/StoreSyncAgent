package service

import (
	"context"
	"fmt"
	"strings"

	"storesyncagent/internal/kdzs"
	"storesyncagent/internal/store"
)

type OrderLookupView struct {
	Found              bool              `json:"found"`
	OrderNo            string            `json:"orderNo"`
	BuyerNick          string            `json:"buyerNick,omitempty"`
	RecipientInfo      string            `json:"recipientInfo,omitempty"`
	Spec               string            `json:"spec,omitempty"`
	OutboundTrackingNo string            `json:"outboundTrackingNo,omitempty"`
	Platform           string            `json:"platform,omitempty"`
	SysTid             string            `json:"sysTid,omitempty"`
	ShopName           string            `json:"shopName,omitempty"`
	GoodsTitle         string            `json:"goodsTitle,omitempty"`
	Goods              []kdzs.TradeGoods `json:"goods,omitempty"`
	Source             string            `json:"source,omitempty"`
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
		if ref.Tid != tid && ref.Tid != "" {
			continue
		}
		view.Found = true
		view.Source = "refund"
		view.BuyerNick = ref.BuyerNick
		view.ShopName = ref.ShopName
		view.SysTid = ref.SysTid
		if len(ref.Goods) > 0 {
			view.Spec = firstNonEmptySpec(ref.Goods)
			view.GoodsTitle = ref.Goods[0].Title
			for _, g := range ref.Goods {
				view.Goods = append(view.Goods, kdzs.TradeGoods{
					Title:   g.Title,
					SkuName: g.SkuName,
					PicURL:  g.PicURL,
					Num:     g.Num,
				})
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
	view.BuyerNick = item.BuyerNick
	view.ShopName = item.ShopName
	view.Goods = item.Goods
	if len(item.SysTids) > 0 {
		view.SysTid = item.SysTids[0]
	}
	if len(item.Goods) > 0 {
		view.GoodsTitle = item.Goods[0].Title
		view.Spec = joinGoodsSpec(item.Goods)
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
					view.RecipientInfo = decrypted.FormattedText
					if view.RecipientInfo == "" {
						view.RecipientInfo = formatReceiverFallback(decrypted)
					}
					break
				}
			}
		}
	}
	if view.RecipientInfo == "" {
		view.RecipientInfo = firstNonEmpty(item.FormattedReceiver, formatTradeReceiver(item))
	}
}

func joinGoodsSpec(goods []kdzs.TradeGoods) string {
	parts := make([]string, 0, len(goods))
	for _, g := range goods {
		label := strings.TrimSpace(g.SkuName)
		if label == "" {
			label = strings.TrimSpace(g.Title)
		}
		if label != "" {
			parts = append(parts, label)
		}
	}
	return strings.Join(parts, "；")
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

func firstNonEmptySpec(goods []kdzs.RefundGoods) string {
	parts := make([]string, 0, len(goods))
	for _, g := range goods {
		if s := strings.TrimSpace(g.SkuName); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "；")
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
