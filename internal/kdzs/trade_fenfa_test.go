package kdzs

import (
	"encoding/json"
	"testing"
)

func TestParseSysFenfaMemo(t *testing.T) {
	raw := json.RawMessage(`{"sysTid":"1","sysFenfaMemo":"加急发","sellerMemo":"商家","distributorRemark":"","orderDetails":[{"oid":"t1","title":"x","num":1,"price":1}]}`)
	item := ParseTradeItemFromJSON(raw, "FXG")
	if item == nil {
		t.Fatal("nil item")
	}
	if item.FenFaMemo != "加急发" {
		t.Fatalf("FenFaMemo=%q", item.FenFaMemo)
	}
	if item.SellerMemo != "商家" {
		t.Fatalf("SellerMemo=%q", item.SellerMemo)
	}
}
