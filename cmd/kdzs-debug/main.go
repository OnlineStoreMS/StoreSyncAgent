package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"storesyncagent/internal/config"
	"storesyncagent/internal/kdzs"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	mobile := flag.String("mobile", "", "KDZS mobile (required)")
	password := flag.String("password", "", "KDZS password (required)")
	platform := flag.String("platform", "FXG", "platform code")
	flag.Parse()

	if *mobile == "" || *password == "" {
		log.Fatal("usage: kdzs-debug -mobile MOBILE -password PASSWORD [-config configs/config.yaml]")
	}

	absConfig, err := filepath.Abs(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := config.Load(absConfig)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	client := kdzs.NewClient(cfg.Kdzs.BaseURL)
	session := kdzs.NewSession(client)
	if err := session.EnsureLogin(ctx, *mobile, *password); err != nil {
		log.Fatal(err)
	}

	stats, err := client.GetMainPageStats(ctx)
	if err != nil {
		log.Fatalf("mainPage: %v", err)
	}
	printJSON("mainPageStats", stats)

	start, end := kdzs.DefaultDateRange()
	fmt.Printf("dateRange: %s ~ %s\n", start, end)

	factoryIDs, mallIDs, err := session.LoadQueryContext(ctx, *platform)
	if err != nil {
		log.Fatalf("queryContext: %v", err)
	}
	fmt.Printf("factoryIDs=%v mallIDs=%v\n", factoryIDs, mallIDs)

	raw, err := session.QueryTradesRaw(ctx, kdzs.TradeQuery{
		Platform:    *platform,
		TradeStatus: "wait_audit",
		PageNo:      1,
		PageSize:    20,
	})
	if err != nil {
		log.Fatalf("queryTradesRaw: %v", err)
	}
	printJSON("queryTradesRaw", raw)

	shops, _ := client.ListEcommerceShops(ctx)
	var fxgShopIDs []string
	for _, shop := range shops {
		if shop.Platform == *platform {
			fxgShopIDs = append(fxgShopIDs, shop.MallUserID)
			fmt.Printf("shop %s (%s) tokenValid=%v expire=%s\n", shop.MallUserName, shop.MallUserID, shop.TokenValid, shop.ExpireTime)
		}
	}

	if len(fxgShopIDs) > 0 {
		rawShop, err := session.QueryTradesRaw(ctx, kdzs.TradeQuery{
			Platform:    *platform,
			TradeStatus: "wait_audit",
			PageNo:      1,
			PageSize:    20,
			ShopID:      fxgShopIDs[0],
		})
		if err != nil {
			fmt.Printf("query with shop %s: %v\n", fxgShopIDs[0], err)
		} else {
			fmt.Printf("query with shop %s total=%v dataLen=%d\n", fxgShopIDs[0], rawShop["total"], lenSlice(rawShop["data"]))
		}

		ps, _ := session.PlatformSession(ctx, *platform)
		for _, extra := range []struct {
			name string
			body map[string]any
		}{
			{"no-factory-filter", map[string]any{
				"rdsUser": true, "platform": *platform, "tradeStatus": "wait_audit", "status": "WAIT_AUDIT",
				"pageNo": 1, "pageSize": 20, "startDateTime": start, "endDateTime": end, "timeType": 1,
				"shopIds": fxgShopIDs,
			}},
			{"all-shops", map[string]any{
				"rdsUser": true, "platform": *platform, "tradeStatus": "wait_audit", "status": "WAIT_AUDIT",
				"pageNo": 1, "pageSize": 20, "startDateTime": start, "endDateTime": end, "timeType": 1,
				"factoryIds": factoryIDs, "distributorIds": mallIDs, "shopIds": fxgShopIDs,
				"showDaifaTrade": 1, "isFXGDaifa": 1, "fxgDaifaPage": true,
			}},
		} {
			var resp map[string]any
			if err := client.PostPlatform(ctx, ps, "/tradeManage/queryRdsTradeList", extra.body, &resp); err != nil {
				fmt.Printf("%s error: %v\n", extra.name, err)
			} else {
				fmt.Printf("%s total=%v dataLen=%d\n", extra.name, resp["total"], lenSlice(resp["data"]))
			}
		}
		var prePush map[string]any
		if err := client.PostPlatform(ctx, ps, "/tradeManage/checkExistPrePushTrade", map[string]any{}, &prePush); err != nil {
			fmt.Printf("checkExistPrePushTrade error: %v\n", err)
		} else {
			printJSON("checkExistPrePushTrade", prePush)
		}
	}

	for _, path := range []string{
		"/tradeManage/getRdsShopSyncConfig",
		"/tradeManage/queryRdsTradeCount",
		"/tradeManage/hasPushTrade",
	} {
		var resp map[string]any
		ps, err := session.PlatformSession(ctx, *platform)
		if err != nil {
			log.Fatal(err)
		}
		body := map[string]any{
			"platform":       *platform,
			"tradeStatus":    "wait_audit",
			"status":         "WAIT_AUDIT",
			"rdsUser":        true,
			"startDateTime":  start,
			"endDateTime":    end,
			"factoryIds":     factoryIDs,
			"distributorIds": mallIDs,
			"pageNo":         1,
			"pageSize":       20,
			"timeType":       1,
		}
		if err := client.PostPlatform(ctx, ps, path, body, &resp); err != nil {
			fmt.Printf("%s error: %v\n", path, err)
		} else {
			printJSON(path, resp)
		}
	}

	ps, err := session.PlatformSession(ctx, *platform)
	if err != nil {
		log.Fatal(err)
	}
	commonBody := map[string]any{
		"platform":       *platform,
		"tradeStatus":    "wait_audit",
		"status":         "WAIT_AUDIT",
		"startDateTime":  start,
		"endDateTime":    end,
		"factoryIds":     factoryIDs,
		"distributorIds": mallIDs,
		"pageNo":         1,
		"pageSize":       20,
		"timeType":       1,
	}
	var asyncResp map[string]any
	if err := client.PostPlatform(ctx, ps, "/tradeManage/getQueryAsyncCode", commonBody, &asyncResp); err != nil {
		fmt.Printf("getQueryAsyncCode error: %v\n", err)
	} else {
		printJSON("getQueryAsyncCode", asyncResp)
		asyncCode := nestedString(asyncResp, "data", "asyncCode")
		listBody := map[string]any{
			"asyncCode":   asyncCode,
			"rdsUser":     false,
			"platform":    *platform,
			"tradeStatus": "wait_audit",
			"pageNo":      1,
			"pageSize":    20,
		}
		for k, v := range commonBody {
			listBody[k] = v
		}
		var listResp map[string]any
		if err := client.PostPlatform(ctx, ps, "/tradeManage/queryTradeList", listBody, &listResp); err != nil {
			fmt.Printf("queryTradeList error: %v\n", err)
		} else {
			printJSON("queryTradeList", listResp)
		}
	}

	for _, status := range []string{"wait_audit"} {
		r, err := session.QueryTrades(ctx, kdzs.TradeQuery{
			Platform:    *platform,
			TradeStatus: status,
			PageNo:      1,
			PageSize:    5,
		})
		if err != nil {
			fmt.Printf("%s: error %v\n", status, err)
			continue
		}
		fmt.Printf("%s: total=%d items=%d\n", status, r.Total, len(r.Items))
	}
}

func printJSON(label string, v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Printf("\n=== %s ===\n%s\n", label, string(b))
}

func lenSlice(v any) int {
	switch x := v.(type) {
	case []any:
		return len(x)
	case []json.RawMessage:
		return len(x)
	default:
		return 0
	}
}

func nestedString(v map[string]any, keys ...string) string {
	cur := any(v)
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[k]
	}
	s, _ := cur.(string)
	return s
}
