package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"storesyncagent/internal/config"
	"storesyncagent/internal/kdzs"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	client := kdzs.NewClient(cfg.Kdzs.BaseURL)
	sess := kdzs.NewSession(client)
	if err := sess.EnsureLogin(ctx, "13083989258", "Yyz@201314."); err != nil {
		log.Fatal(err)
	}

	fmt.Println("======== 售后中心（申请中相关） ========")
	lists := [][]string{
		{"WAIT_SELLER_AGREE"},
		{"REFUND_MONEY_APPLY_ING"},
		{"WAIT_BUYER_RETURN_ITEM"},
		{"WAIT_SELLER_CONFIRM_RECEIVE"},
		nil, // all recent
	}
	for _, st := range lists {
		rq := kdzs.RefundQuery{
			Platform:            "FXG",
			PageNo:              1,
			PageSize:            30,
			DateType:            4,
			StartDateTime:       time.Now().AddDate(0, 0, -30).Format("2006-01-02 15:04:05"),
			EndDateTime:         time.Now().Format("2006-01-02 15:04:05"),
			AfterSaleStatusList: st,
		}
		label := "ALL"
		if len(st) > 0 {
			label = strings.Join(st, ",")
		}
		res, err := sess.QueryRefunds(ctx, rq)
		if err != nil {
			fmt.Printf("[%s] err=%v\n", label, err)
			continue
		}
		fmt.Printf("[%s] total=%d showing=%d\n", label, res.Total, len(res.Items))
		statusCnt := map[string]int{}
		for i, it := range res.Items {
			key := it.AfterSaleStatus + "|" + it.AfterSaleStatusText
			statusCnt[key]++
			if i < 8 {
				title := ""
				if len(it.Goods) > 0 {
					title = it.Goods[0].SkuName
					if title == "" {
						title = it.Goods[0].Title
					}
				}
				fmt.Printf("  tid=%s after=%s/%s type=%d/%s sku=%s\n",
					it.Tid, it.AfterSaleStatus, it.AfterSaleStatusText, it.AfterSaleType, it.AfterSaleTypeText, title)
			}
		}
		printCounts(statusCnt)
		time.Sleep(2 * time.Second)
	}

	fmt.Println("\n======== 推送订单·全部 翻页（含申请退款中） ========")
	end := time.Now()
	start := end.AddDate(0, 0, -60)
	q := kdzs.TradeQuery{
		Platform: "FXG", TradeStatus: "all", PageSize: 50, TimeType: 0,
		StartDateTime: start.Format("2006-01-02 15:04:05"),
		EndDateTime:   end.Format("2006-01-02 15:04:05"),
	}
	headerAS := map[string]int{}
	goodsAS := map[string]int{}
	applySamples := []string{}
	finishHeader, finishGoodsMarked, finishGoodsEmpty := 0, 0, 0
	totalSeen := 0
	var total int
	for page := 1; page <= 50; page++ {
		q.PageNo = page
		var res *kdzs.TradeListResult
		var err error
		for retry := 0; retry < 6; retry++ {
			res, err = sess.QueryTrades(ctx, q)
			if err == nil {
				break
			}
			if strings.Contains(err.Error(), "频繁") || strings.Contains(err.Error(), "811") {
				time.Sleep(5 * time.Second)
				continue
			}
			break
		}
		if err != nil {
			log.Printf("page %d: %v", page, err)
			break
		}
		if page == 1 {
			total = res.Total
			fmt.Printf("total=%d\n", total)
		}
		if len(res.Items) == 0 {
			break
		}
		for _, it := range res.Items {
			totalSeen++
			headerAS[nz(it.AfterSaleStatus)+"|"+nz(it.AfterSaleStatusText)]++
			hasFinishG := false
			hasAnyG := false
			for _, g := range it.Goods {
				goodsAS[nz(g.AfterSaleStatus)+"|"+nz(g.AfterSaleStatusText)]++
				if g.AfterSaleStatus != "" && g.AfterSaleStatus != "—" {
					hasAnyG = true
				}
				uas := strings.ToUpper(g.AfterSaleStatus)
				if strings.Contains(uas, "FINISH") || strings.Contains(uas, "SUCCESS") {
					hasFinishG = true
				}
				if isApplying(g.AfterSaleStatus, g.AfterSaleStatusText) {
					applySamples = append(applySamples, fmt.Sprintf("G tid=%s trade=%s after=%s/%s sku=%s",
						first(it.Tids), it.TradeStatus, g.AfterSaleStatus, g.AfterSaleStatusText, g.SkuName))
				}
			}
			if strings.Contains(strings.ToUpper(it.AfterSaleStatus), "FINISH") {
				finishHeader++
				if hasFinishG {
					finishGoodsMarked++
				} else if !hasAnyG {
					finishGoodsEmpty++
				}
			}
			if isApplying(it.AfterSaleStatus, it.AfterSaleStatusText) {
				applySamples = append(applySamples, fmt.Sprintf("H tid=%s trade=%s after=%s/%s",
					first(it.Tids), it.TradeStatus, it.AfterSaleStatus, it.AfterSaleStatusText))
			}
		}
		fmt.Printf("page=%d scanned=%d/%d apply=%d\n", page, totalSeen, total, len(applySamples))
		if total > 0 && totalSeen >= total {
			break
		}
		time.Sleep(4500 * time.Millisecond)
	}
	fmt.Printf("\nscanned=%d finishHeader=%d goodsMarked=%d goodsEmptyMark=%d\n", totalSeen, finishHeader, finishGoodsMarked, finishGoodsEmpty)
	fmt.Println("--- header ---")
	printCounts(headerAS)
	fmt.Println("--- goods ---")
	printCounts(goodsAS)
	fmt.Println("--- apply samples ---")
	for i, s := range applySamples {
		if i >= 40 {
			break
		}
		fmt.Println(s)
	}
}

func isApplying(code, text string) bool {
	u := strings.ToUpper(code + " " + text)
	return strings.Contains(u, "APPLY") || strings.Contains(u, "_ING") || strings.Contains(text, "申请") ||
		strings.Contains(u, "WAIT_SELLER") || strings.Contains(text, "退款中")
}
func nz(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(empty)"
	}
	return s
}
func first(a []string) string {
	if len(a) == 0 {
		return ""
	}
	return a[0]
}
func printCounts(m map[string]int) {
	type kv struct{ k string; v int }
	arr := make([]kv, 0, len(m))
	for k, v := range m {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
	for _, x := range arr {
		fmt.Printf("%6d  %s\n", x.v, x.k)
	}
}
