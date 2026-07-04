package service

import (
	"testing"

	"storesyncagent/internal/kdzs"
)

func TestNotificationKeyUrgentEscalation(t *testing.T) {
	item := kdzs.RefundItem{
		RefundID: "123",
		SLA:      &kdzs.RefundSLA{Urgency: "warning"},
	}
	kWarning := notificationKey("acc1", item, "urgent")
	item.SLA.Urgency = "critical"
	kCritical := notificationKey("acc1", item, "urgent")
	item.SLA.Urgency = "imminent"
	kImminent := notificationKey("acc1", item, "urgent")
	item.SLA.Urgency = "expired"
	kExpired := notificationKey("acc1", item, "urgent")

	keys := []string{kWarning, kCritical, kImminent, kExpired}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] == keys[j] {
				t.Fatalf("urgency keys must differ: %q vs %q", keys[i], keys[j])
			}
		}
	}

	kOther := notificationKey("acc1", item, "pickup_pending")
	if kOther == kExpired {
		t.Fatal("other scenario key format unchanged")
	}
}
