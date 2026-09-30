package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"storesyncagent/internal/kdzs"
)

func main() {
	tid := "6929962463808618335"
	client := kdzs.NewClient("https://df.kdzs.com")
	sess := kdzs.NewSession(client)
	ctx := context.Background()
	if err := sess.EnsureLogin(ctx, "13083989258", "Yyz@201314."); err != nil {
		panic(err)
	}
	time.Sleep(2 * time.Second)
	for _, st := range []string{"all", "wait_audit", "wait_send", "shipped", ""} {
		res, err := sess.QueryTrades(ctx, kdzs.TradeQuery{
			Platform: "FXG", TradeStatus: st, Tid: tid, PageNo: 1, PageSize: 20,
		})
		if err != nil {
			fmt.Printf("[%s] err=%v\n", st, err)
			time.Sleep(3 * time.Second)
			continue
		}
		fmt.Printf("==== status=%q total=%d items=%d ====\n", st, res.Total, len(res.Items))
		for i, it := range res.Items {
			b, _ := json.MarshalIndent(map[string]any{
				"sysTids": it.SysTids, "tids": it.Tids,
				"tradeStatus": it.TradeStatus, "statusText": it.StatusText,
				"agentType": it.AgentType, "factoryName": it.FactoryName,
				"payment": it.Payment,
				"goods": func() []map[string]any {
					out := make([]map[string]any, 0, len(it.Goods))
					for _, g := range it.Goods {
						out = append(out, map[string]any{
							"title": g.Title, "sku": g.SkuName, "num": g.Num, "oid": g.Oid,
							"after": g.AfterSaleStatusText, "orderStatus": g.OrderStatus,
						})
					}
					return out
				}(),
			}, "", "  ")
			fmt.Printf("-- #%d --\n%s\n", i+1, string(b))
		}
		time.Sleep(2 * time.Second)
	}
}
