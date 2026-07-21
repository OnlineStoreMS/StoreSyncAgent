package kdzs

import (
	"encoding/json"
	"testing"
)

func TestParseSelfOrderDoesNotUseFactoryUserId(t *testing.T) {
	raw := []byte(`{
		"sysTid":"6746325761873575637",
		"tid":"6928094608923131712",
		"tradeStatus":"ORDER_PAID",
		"daifaStatus":1,
		"factoryUserId":800888,
		"factoryName":"",
		"orderDetails":[{"oid":"6928094608923131712","factoryUserId":800888}]
	}`)
	item := ParseTradeItemFromJSON(raw, "FXG")
	if item == nil {
		t.Fatal("nil item")
	}
	if item.AgentType != AgentTypeSelfPrint {
		t.Fatalf("agentType=%d want self", item.AgentType)
	}
	if item.FactoryID != "" {
		t.Fatalf("factoryId=%q want empty", item.FactoryID)
	}
}

func TestParseDropshipUsesDaifaStatusAndFactoryName(t *testing.T) {
	raw := []byte(`{
		"sysTid":"33347589063139131778",
		"tid":"6928065517757431261",
		"tradeStatus":"ORDER_PAID",
		"daifaStatus":2,
		"pushType":2,
		"factoryUserId":903134,
		"factoryName":"13817054118",
		"factoryRemark":"微笑单车",
		"orderDetails":[{"oid":"6928065517757431261","factoryUserId":903134}]
	}`)
	item := ParseTradeItemFromJSON(raw, "FXG")
	if item == nil {
		t.Fatal("nil item")
	}
	if item.AgentType != AgentTypePushFactory {
		t.Fatalf("agentType=%d want factory", item.AgentType)
	}
	if item.FactoryID != "903134" {
		t.Fatalf("factoryId=%q", item.FactoryID)
	}
	if item.FactoryName != "13817054118" {
		t.Fatalf("factoryName=%q", item.FactoryName)
	}
}

func TestInferKDZSListStatus(t *testing.T) {
	if got := InferKDZSListStatus("ORDER_PAID", "待发货"); got != "" {
		t.Fatalf("ecommerce 待发货 must not infer wait_send, got %q", got)
	}
	if got := InferKDZSListStatus("ORDER_PAID", "待推单"); got != "wait_audit" {
		t.Fatalf("got %q", got)
	}
	if got := InferKDZSListStatus("wait_send", "x"); got != "wait_send" {
		t.Fatalf("got %q", got)
	}
}

func TestFinalizeSetsListStatusAndKeepsAgent(t *testing.T) {
	items := []TradeListItem{{
		TradeStatus: "ORDER_PAID",
		StatusText:  "待发货",
		AgentType:   AgentTypeSelfPrint,
		FactoryID:   "",
	}}
	finalizeTradeListItems(items, "wait_send")
	if items[0].TradeStatus != "wait_send" {
		t.Fatalf("tradeStatus=%s", items[0].TradeStatus)
	}
	if items[0].AgentType != AgentTypeSelfPrint {
		t.Fatalf("agentType=%d", items[0].AgentType)
	}
	_ = json.Marshal // keep import used if trimmed
}
