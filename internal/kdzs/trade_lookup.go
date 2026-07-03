package kdzs

import (
	"context"
	"strings"
)

func (s *Session) LookupTradeByTid(ctx context.Context, platform, tid string) (*TradeListItem, string, error) {
	tid = strings.TrimSpace(tid)
	if tid == "" {
		return nil, "", nil
	}
	statuses := []string{"", "shipped", "completed", "wait_send", "wait_audit"}
	for _, status := range statuses {
		result, err := s.QueryTrades(ctx, TradeQuery{
			Platform:    platform,
			TradeStatus: status,
			Tid:         tid,
			PageNo:      1,
			PageSize:    10,
		})
		if err != nil {
			continue
		}
		for i := range result.Items {
			item := &result.Items[i]
			if tradeItemMatchesTid(item, tid) {
				return item, status, nil
			}
		}
		if len(result.Items) == 1 {
			return &result.Items[0], status, nil
		}
	}
	return nil, "", nil
}

func tradeItemMatchesTid(item *TradeListItem, tid string) bool {
	if item == nil {
		return false
	}
	for _, t := range item.Tids {
		if t == tid {
			return true
		}
	}
	return false
}
