package main

import (
	"context"
	"fmt"
	"time"

	"storesyncagent/internal/kdzs"
)

func main() {
	client := kdzs.NewClient("https://df.kdzs.com")
	sess := kdzs.NewSession(client)
	ctx := context.Background()
	_ = sess.EnsureLogin(ctx, "13083989258", "Yyz@201314.")
	time.Sleep(2 * time.Second)
	for _, st := range []string{"shipped", "wait_send", "all"} {
		res, err := sess.QueryTrades(ctx, kdzs.TradeQuery{
			Platform: "DFHAND", TradeStatus: st, PageNo: 1, PageSize: 10,
			StartDateTime: "2026-10-01 00:00:00", EndDateTime: "2026-10-03 23:59:59",
		})
		if err != nil {
			fmt.Printf("[%s] err=%v\n", st, err)
		} else {
			fmt.Printf("[%s] total=%d n=%d\n", st, res.Total, len(res.Items))
			for i, it := range res.Items {
				if i >= 5 {
					break
				}
				fmt.Printf("  status=%s express=%s/%s tids=%v sys=%v\n", it.TradeStatus, it.ExpressCompany, it.ExpressNo, it.Tids, it.SysTids)
			}
		}
		time.Sleep(2 * time.Second)
	}
}
