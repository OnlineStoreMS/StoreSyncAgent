package kdzs

import "testing"

func TestApplyOidSplitFilter(t *testing.T) {
	list := []TradeInfoItem{
		{SysTid: "s1", OidList: []string{"a", "b", "c"}},
	}
	out := applyOidSplitFilter(list, []string{"a", "c"}, false)
	if len(out) != 1 || len(out[0].OidList) != 2 || !out[0].Split {
		t.Fatalf("got %+v", out)
	}
	out2 := applyOidSplitFilter(list, []string{"z"}, false)
	if len(out2) != 1 || len(out2[0].OidList) != 3 {
		// no intersection → fallback to original list
		t.Fatalf("fallback got %+v", out2)
	}
}
