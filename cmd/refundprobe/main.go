package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"storesyncagent/internal/config"
	"storesyncagent/internal/kdzs"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	mobile := flag.String("mobile", "", "KDZS mobile (required)")
	password := flag.String("password", "", "KDZS password (required)")
	flag.Parse()

	if *mobile == "" || *password == "" {
		log.Fatal("usage: refundprobe -mobile MOBILE -password PASSWORD [-config configs/config.yaml]")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	client := kdzs.NewClient(cfg.Kdzs.BaseURL)
	s := kdzs.NewSession(client)
	ctx := context.Background()
	if err := s.EnsureLogin(ctx, *mobile, *password); err != nil {
		log.Fatal(err)
	}
	ps, _ := s.PlatformSession(ctx, "FXG")
	start, end := kdzs.DefaultDateRange()
	shopList, _ := client.ListEcommerceShops(ctx)
	var shopIds []string
	for _, sh := range shopList {
		if sh.Platform == "FXG" && sh.MallUserID != "" {
			shopIds = append(shopIds, sh.MallUserID)
		}
	}
	fmt.Println("shops", len(shopIds), "range", start, "~", end)

	base := map[string]any{
		"platform": "FXG", "roleSource": "SELLER", "pageNo": 1, "pageSize": 1,
		"dateType": 4, "startDateTime": start, "endDateTime": end,
		"shopIds": shopIds,
	}
	tests := []struct {
		name string
		body map[string]any
	}{
		{"tradeDaifaStatus empty", merge(base, map[string]any{"tradeDaifaStatus": ""})},
		{"no tradeDaifaStatus", clone(base)},
		{"tradeDaifaStatus 0", merge(base, map[string]any{"tradeDaifaStatus": 0})},
		{"tradeDaifaStatus 1", merge(base, map[string]any{"tradeDaifaStatus": 1})},
		{"tradeDaifaStatus 2", merge(base, map[string]any{"tradeDaifaStatus": 2})},
		{"tradeDaifaStatus null omit", merge(base, map[string]any{})},
	}
	for _, t := range tests {
		var resp map[string]any
		_ = client.PostPlatform(ctx, ps, "/refund/fxdf/queryRefund", t.body, &resp)
		fmt.Printf("%s => total=%v\n", t.name, resp["total"])
	}

	// tab count
	var tab map[string]any
	_ = client.PostPlatform(ctx, ps, "/refund/fxdf/queryRefundTabCount", merge(base, map[string]any{"tradeDaifaStatus": ""}), &tab)
	b, _ := json.MarshalIndent(tab, "", "  ")
	fmt.Println("\ntabCount:", string(b))

	// per shop totals
	for _, sid := range shopIds {
		var resp map[string]any
		body := merge(base, map[string]any{"tradeDaifaStatus": "", "shopIds": []string{sid}})
		_ = client.PostPlatform(ctx, ps, "/refund/fxdf/queryRefund", body, &resp)
		fmt.Printf("shop %s total=%v\n", sid, resp["total"])
	}
}

func merge(base, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
func clone(m map[string]any) map[string]any { return merge(m, nil) }
