package kdzs

import "testing"

func TestSortTradeItemsByOrderTimeDesc(t *testing.T) {
	items := []TradeListItem{
		{Tids: []string{"old"}, CreateTime: "2026-07-01 10:00:00"},
		{Tids: []string{"new"}, CreateTime: "2026-07-20 12:00:00"},
		{Tids: []string{"mid"}, PayTime: "2026-07-10 08:00:00"},
	}
	SortTradeItemsByOrderTimeDesc(items)
	if items[0].Tids[0] != "new" || items[1].Tids[0] != "mid" || items[2].Tids[0] != "old" {
		t.Fatalf("order=%v %v %v", items[0].Tids, items[1].Tids, items[2].Tids)
	}
}
