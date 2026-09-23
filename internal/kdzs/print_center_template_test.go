package kdzs

import (
	"encoding/json"
	"testing"
)

func TestParseKddTemplateListUsesPrintPageFields(t *testing.T) {
	raw := []byte(`{
		"result": 100,
		"data": {
			"ModeListShows": [
				{"Mode_ListShowId": 11, "ExcodeName": "抖音中通一联单", "ExCode": "ZTO", "KddType": 8, "Modeid": "kdd"},
				{"Mode_ListShowId": 22, "ExcodeName": "普通面单", "ExCode": "YTO", "KddType": 1, "Modeid": "kdd"},
				{"Mode_ListShowId": 23, "ExcodeName": "网点面单", "ExCode": "STO", "KddType": 2, "Modeid": "kdd"},
				{"Mode_ListShowId": 33, "ExcodeName": "小红书新模板", "ExCode": "SF", "KddType": 16, "Modeid": "kdd"},
				{"Mode_ListShowId": 44, "ExcodeName": "发货单", "ExCode": "FHD", "KddType": 17, "Modeid": "fhd"}
			]
		}
	}`)
	var resp flexAPIResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatal(err)
	}
	items, err := parseKddTemplateList(resp)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items: %#v", len(items), items)
	}
	if items[0].TemplateID != "11" || items[0].TemplateName != "抖音中通一联单" || items[0].Platform != "抖店" || items[0].CarrierCode != "ZTO" {
		t.Fatalf("first: %#v", items[0])
	}
	if items[1].TemplateID != "33" || items[1].Platform != "小红书" {
		t.Fatalf("xhs: %#v", items[1])
	}
}

func TestParseKddTemplateListRejectsMissingList(t *testing.T) {
	raw := []byte(`{"result":100,"data":{"list":[]}}`)
	var resp flexAPIResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatal(err)
	}
	if _, err := parseKddTemplateList(resp); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseKddTemplateListAllowsEmpty(t *testing.T) {
	raw := []byte(`{"result":100,"data":{"ModeListShows":[]}}`)
	var resp flexAPIResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatal(err)
	}
	items, err := parseKddTemplateList(resp)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("got %#v", items)
	}
}
