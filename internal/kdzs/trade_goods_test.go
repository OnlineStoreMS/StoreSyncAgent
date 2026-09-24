package kdzs

import (
	"encoding/json"
	"testing"
)

func TestParseTradeGoodsCapturesLineAfterSale(t *testing.T) {
	raw := json.RawMessage(`{
		"sysTid":"s1",
		"tid":"t-main",
		"orderDetails":[
			{"oid":"a","title":"盘片","skuName":"50-34T","num":1,"price":158,"afterSaleStatus":"REFUND_SUCCESS","orderStatus":"TRADE_CLOSED"},
			{"oid":"b","title":"链条","skuName":"HG95","num":1,"price":135,"afterSaleStatus":"REFUND_MONEY_NONE","orderStatus":"ORDER_PAID"}
		]
	}`)
	item := ParseTradeItemFromJSON(raw, "FXG")
	if item == nil || len(item.Goods) != 2 {
		t.Fatalf("goods=%v", item)
	}
	if item.Goods[0].AfterSaleStatus != "REFUND_SUCCESS" || item.Goods[0].OrderStatus != "TRADE_CLOSED" {
		t.Fatalf("line0=%+v", item.Goods[0])
	}
	if !GoodsLineExcludedFromFulfillment(item.Goods[0]) {
		t.Fatal("refunded line should be excluded")
	}
	if GoodsLineExcludedFromFulfillment(item.Goods[1]) {
		t.Fatalf("active line should keep: %+v", item.Goods[1])
	}
	// 平台单号应是主单 tid，而不是子单 oid
	if len(item.Tids) < 1 || item.Tids[0] != "t-main" {
		t.Fatalf("tids=%v want parent first", item.Tids)
	}
	if !containsStr(item.Tids, "a") || !containsStr(item.Tids, "b") {
		t.Fatalf("oids should still be searchable in tids=%v", item.Tids)
	}
}

func containsStr(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func TestGoodsLineExcludedFromFulfillment(t *testing.T) {
	cases := []struct {
		g    TradeGoods
		want bool
	}{
		{TradeGoods{Num: 1, AfterSaleStatus: "REFUND_SUCCESS"}, true},
		{TradeGoods{Num: 1, OrderStatus: "TRADE_CLOSED"}, true},
		{TradeGoods{Num: 0, AfterSaleStatus: ""}, true},
		{TradeGoods{Num: 1, AfterSaleStatus: "WAIT_SELLER_AGREE"}, false},
		{TradeGoods{Num: 1, AfterSaleStatus: "REFUND_MONEY_NONE", OrderStatus: "ORDER_PAID"}, false},
	}
	for i, c := range cases {
		if got := GoodsLineExcludedFromFulfillment(c.g); got != c.want {
			t.Fatalf("case %d got %v want %v (%+v)", i, got, c.want, c.g)
		}
	}
}
