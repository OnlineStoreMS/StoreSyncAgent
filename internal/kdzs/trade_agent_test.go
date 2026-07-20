package kdzs

import "testing"

func TestNormalizeAgentTypeFromFactoryName(t *testing.T) {
	item := &TradeListItem{FactoryName: "13817054118"}
	normalizeAgentType(item)
	if item.AgentType != AgentTypePushFactory {
		t.Fatalf("want factory agent, got %d", item.AgentType)
	}
}

func TestInferKDZSListStatus(t *testing.T) {
	if got := InferKDZSListStatus("ORDER_PAID", "待发货"); got != "wait_send" {
		t.Fatalf("got %q", got)
	}
	if got := InferKDZSListStatus("wait_send", "x"); got != "wait_send" {
		t.Fatalf("got %q", got)
	}
}
