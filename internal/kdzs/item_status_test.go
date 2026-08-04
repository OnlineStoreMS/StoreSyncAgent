package kdzs

import "testing"

func TestIsPlatformDeletedStatus(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"normal", false},
		{"onsale", false},
		{"delete", true},
		{"deleted", true},
		{"DELETED", true},
		{"platform_deleted", true},
		{"平台已删除", true},
		{"已删除", true},
		{"商品已删除", true},
	}
	for _, tc := range cases {
		if got := IsPlatformDeletedStatus(tc.in); got != tc.want {
			t.Fatalf("IsPlatformDeletedStatus(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsPlatformDeletedSKU(t *testing.T) {
	if IsPlatformDeletedSKU(ShopItemSku{Status: "normal"}) {
		t.Fatal("normal should not be deleted")
	}
	if !IsPlatformDeletedSKU(ShopItemSku{Status: "平台已删除"}) {
		t.Fatal("平台已删除 should be deleted")
	}
	// 快递助手商品页「平台已删除」实际字段是 state=1，status 仍为 normal
	if !IsPlatformDeletedSKU(ShopItemSku{Status: "normal", State: 1}) {
		t.Fatal("state=1 should be treated as platform deleted")
	}
}
