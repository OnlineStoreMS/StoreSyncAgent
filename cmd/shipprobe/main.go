package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"storesyncagent/internal/kdzs"
)

func main() {
	ctx := context.Background()
	client := kdzs.NewClient("https://df.kdzs.com")
	session := kdzs.NewSession(client)
	if err := session.EnsureLogin(ctx, os.Args[1], os.Args[2]); err != nil {
		panic(err)
	}
	res, err := session.QueryTrades(ctx, kdzs.TradeQuery{
		Platform: "FXG", TradeStatus: "shipped", PageNo: 1, PageSize: 3,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("list items=%d total=%d\n", len(res.Items), res.Total)
	var sysTids []string
	for _, it := range res.Items {
		sysTids = append(sysTids, it.SysTids...)
		fmt.Printf("list: tids=%v sys=%v status=%s agent=%d factory=%s\n", it.Tids, it.SysTids, it.TradeStatus, it.AgentType, it.FactoryName)
	}
	if len(sysTids) == 0 {
		return
	}
	n := len(sysTids)
	if n > 3 {
		n = 3
	}
	pkgs, err := session.FetchTradeDetails(ctx, "FXG", "shipped", sysTids[:n])
	if err != nil {
		panic(err)
	}
	fmt.Printf("details=%d\n", len(pkgs))
	for i, raw := range pkgs {
		if i >= 2 {
			break
		}
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Printf("\n=== detail %d keys(%d) ===\n%s\n", i, len(keys), strings.Join(keys, ", "))
		for _, k := range keys {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "sid") || strings.Contains(lk, "mail") || strings.Contains(lk, "express") ||
				strings.Contains(lk, "logistic") || strings.Contains(lk, "send") || strings.Contains(lk, "consign") ||
				strings.Contains(lk, "ship") || strings.Contains(lk, "yd") || strings.Contains(lk, "kd") ||
				strings.Contains(lk, "waybill") || strings.Contains(lk, "track") || strings.Contains(lk, "courier") ||
				strings.Contains(lk, "print") || strings.Contains(lk, "company") || strings.Contains(lk, "delivery") ||
				strings.Contains(lk, "time") {
				fmt.Printf("  %s = %v\n", k, compact(m[k]))
			}
		}
		pretty, _ := json.MarshalIndent(m, "", "  ")
		if len(pretty) > 8000 {
			pretty = append(pretty[:8000], []byte("\n...(trunc)")...)
		}
		fmt.Printf("pretty:\n%s\n", pretty)
	}
}

func compact(v any) string {
	b, _ := json.Marshal(v)
	s := string(b)
	if len(s) > 240 {
		return s[:240] + "..."
	}
	return s
}
