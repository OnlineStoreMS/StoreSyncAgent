package service

import (
	"context"
	"fmt"
	"strings"

	"storesyncagent/internal/kdzs"
)

func normalizeTrackingNo(v string) string {
	return strings.ToUpper(strings.TrimSpace(v))
}

func trackingNoMatches(sid, trackingNo string) bool {
	return normalizeTrackingNo(sid) == trackingNo
}

func refundGoodsToTradeGoods(goods []kdzs.RefundGoods) []kdzs.TradeGoods {
	out := make([]kdzs.TradeGoods, 0, len(goods))
	for _, g := range goods {
		out = append(out, kdzs.TradeGoods{
			Title:   g.Title,
			SkuName: g.SkuName,
			PicURL:  g.PicURL,
			Num:     g.Num,
		})
	}
	return out
}

func (s *SyncService) LookupOrderByTrackingNo(ctx context.Context, trackingNo string) (*OrderLookupView, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	trackingNo = normalizeTrackingNo(trackingNo)
	if trackingNo == "" {
		return nil, fmt.Errorf("trackingNo is required")
	}

	view := &OrderLookupView{Found: false, OrderNo: trackingNo}
	platforms := []string{kdzs.PlatformDouyin, kdzs.PlatformTaobao, kdzs.PlatformXHS}

	for _, platform := range platforms {
		refundResult, err := s.session.QueryRefunds(ctx, kdzs.RefundQuery{
			Platform: platform,
			Sid:      trackingNo,
			PageNo:   1,
			PageSize: 5,
		})
		if err != nil {
			continue
		}
		for i := range refundResult.Items {
			ref := &refundResult.Items[i]
			if !trackingNoMatches(ref.Sid, trackingNo) {
				continue
			}
			view.Found = true
			view.Platform = platform
			view.OrderNo = firstNonEmpty(ref.Tid, trackingNo)
			view.SysTid = ref.SysTid
			view.ShopName = ref.ShopName
			view.StatusText = ref.AfterSaleStatusText
			view.Source = "refund"
			if len(ref.Goods) > 0 {
				view.Goods = refundGoodsToTradeGoods(ref.Goods)
				view.GoodsTitle = ref.Goods[0].Title
			}
			if ref.SysTid != "" {
				for _, status := range []string{"shipped", "completed", "wait_send", "wait_audit", kdzs.DefaultTradeStatus()} {
					details, err := s.session.FetchTradeDetails(ctx, platform, status, []string{ref.SysTid})
					if err != nil || len(details) == 0 {
						continue
					}
					if parsed := kdzs.ParseTradeItemFromJSON(details[0], platform); parsed != nil && len(parsed.Goods) > 0 {
						view.Goods = parsed.Goods
						view.GoodsTitle = parsed.Goods[0].Title
						view.Source = "refund+trade"
						break
					}
				}
			}
			return view, nil
		}
	}

	for _, platform := range platforms {
		item, tradeStatus, err := s.session.LookupTradeByTid(ctx, platform, trackingNo)
		if err != nil || item == nil {
			continue
		}
		s.enrichOrderLookupFromTradeFull(ctx, view, item, platform, tradeStatus)
		view.Source = "trade"
		return view, nil
	}

	return view, nil
}

func (s *SyncService) LookupOrdersByTrackingNos(ctx context.Context, trackingNos []string) (map[string]*OrderLookupView, error) {
	out := make(map[string]*OrderLookupView, len(trackingNos))
	seen := make(map[string]struct{}, len(trackingNos))
	for _, raw := range trackingNos {
		key := normalizeTrackingNo(raw)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		view, err := s.LookupOrderByTrackingNo(ctx, key)
		if err != nil {
			return nil, err
		}
		out[key] = view
	}
	return out, nil
}

func (s *SyncService) enrichOrderLookupFromTradeFull(ctx context.Context, view *OrderLookupView, item *kdzs.TradeListItem, platform, tradeStatus string) {
	if view == nil || item == nil {
		return
	}
	view.Found = true
	view.Platform = platform
	view.ShopName = item.ShopName
	view.Goods = item.Goods
	view.Payment = item.Payment
	view.PayTime = item.PayTime
	view.StatusText = kdzs.ResolveOrderStatusText(item.StatusText, item.TradeStatus)
	if len(item.SysTids) > 0 {
		view.SysTid = item.SysTids[0]
	}
	if len(item.Tids) > 0 {
		view.OrderNo = item.Tids[0]
	}
	if len(item.Goods) > 0 {
		view.GoodsTitle = item.Goods[0].Title
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
