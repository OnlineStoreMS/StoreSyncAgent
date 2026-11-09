package kdzs

import "testing"

func TestPickPrimaryLogisticsPrefersConsignTime(t *testing.T) {
	list := []TradeLogistics{
		{TrackingNo: "YT0090343524511", CompanyName: "圆通快递", ShipTime: "2026-11-08 15:29:03"},
		{TrackingNo: "SF1551169181248", CompanyName: "顺丰速运", ShipTime: "2026-11-09 00:45:07"},
	}
	got := pickPrimaryLogistics(list, "2026-11-09 00:45:07")
	if got.TrackingNo != "SF1551169181248" {
		t.Fatalf("got %s want SF", got.TrackingNo)
	}
}

func TestPickPrimaryLogisticsFallsBackToLatest(t *testing.T) {
	list := []TradeLogistics{
		{TrackingNo: "YT1", ShipTime: "2026-11-08 15:29:03"},
		{TrackingNo: "SF1", ShipTime: "2026-11-09 00:45:07"},
	}
	got := pickPrimaryLogistics(list, "")
	if got.TrackingNo != "SF1" {
		t.Fatalf("got %s want SF1", got.TrackingNo)
	}
}

func TestApplyTradeLogisticsSkipsAllRefundedPackage(t *testing.T) {
	item := &TradeListItem{
		Goods: []TradeGoods{{
			Title: "链条", Num: 1,
			AfterSaleStatus: "REFUND_MONEY_FINISH",
			OrderStatus:     "ORDER_CANCELLED",
		}},
	}
	trade := map[string]any{
		"consignTime": "2026-11-09 19:52:39",
		"logisticsInfoList": []any{
			map[string]any{
				"companyName": "中通快递",
				"trackingNo":  "79203276978006",
				"shipTime":    "2026-11-09 19:52:39",
			},
		},
	}
	applyTradeLogistics(item, trade)
	if item.ExpressNo != "" {
		t.Fatalf("refunded package should not take shared express, got %s", item.ExpressNo)
	}
	if len(item.Logistics) != 1 || item.Logistics[0].TrackingNo != "79203276978006" {
		t.Fatalf("logistics list should still be kept: %+v", item.Logistics)
	}
}
