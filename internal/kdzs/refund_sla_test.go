package kdzs

import (
	"testing"
	"time"
)

func TestRefundOnlySLAShorterDeadline(t *testing.T) {
	now := parseTestTime(t, "2026-07-03 12:00:00")
	apply := "2026-07-02 12:00:00"
	item := &RefundItem{
		AfterSaleType:   1,
		AfterSaleStatus: "WAIT_SELLER_AGREE",
		Created:         apply,
	}
	sla := ComputeRefundSLA(item, nil, now)
	if sla.Scenario != "refund_only" {
		t.Fatalf("scenario=%q want refund_only", sla.Scenario)
	}
	if sla.RemainingSeconds != 12*3600 {
		t.Fatalf("remaining=%d want %d (36h from apply, now at +24h)", sla.RemainingSeconds, 12*3600)
	}
	if !sla.Important {
		t.Fatal("expected important flag")
	}
}

func parseTestTime(t *testing.T, s string) time.Time {
	t.Helper()
	got, ok := parseKdzsTime(s)
	if !ok {
		t.Fatalf("parse time %q", s)
	}
	return got
}

func TestSortRefundItemsBySLAUrgency(t *testing.T) {
	items := []RefundItem{
		{RefundID: "normal", SLA: &RefundSLA{Urgency: "normal", RemainingSeconds: 86400}},
		{RefundID: "expired", SLA: &RefundSLA{Urgency: "expired", RemainingSeconds: -7200}},
		{RefundID: "critical", SLA: &RefundSLA{Urgency: "critical", RemainingSeconds: 3600}},
		{RefundID: "warning", SLA: &RefundSLA{Urgency: "warning", RemainingSeconds: 10000}},
	}
	SortRefundItemsBySLAUrgency(items)
	order := []string{items[0].RefundID, items[1].RefundID, items[2].RefundID, items[3].RefundID}
	want := []string{"expired", "critical", "warning", "normal"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order=%v want=%v", order, want)
		}
	}
}
