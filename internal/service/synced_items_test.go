package service

import (
	"strings"
	"testing"
)

func TestItemSearchTextIncludesSkuFields(t *testing.T) {
	text := itemSearchText(ItemView{
		ItemID:     "100",
		Title:      "夏季短袖",
		ShortTitle: "短袖",
		OuterID:    "SPU-1",
		ProductNum: "PN1",
		Skus: []ItemSkuView{{
			SkuID:          "sku-9",
			PropertiesName: "黑色 L",
			OuterID:        "SKU-BLK",
			ShortTitle:     "黑L",
			ProductNum:     "PN-SKU",
		}},
	})
	for _, kw := range []string{"夏季短袖", "100", "spu-1", "黑色 l", "sku-9", "sku-blk", "pn-sku"} {
		if !strings.Contains(text, kw) {
			t.Fatalf("search text %q missing %q", text, kw)
		}
	}
}

func TestNormalizeKeywordsSplitsAndDedupes(t *testing.T) {
	got := normalizeKeywords([]string{" 夏季,短袖 ", "短袖", "黑色 L"})
	want := []string{"夏季", "短袖", "黑色 L"}
	if len(got) != len(want) {
		t.Fatalf("got %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v", got)
		}
	}
}
