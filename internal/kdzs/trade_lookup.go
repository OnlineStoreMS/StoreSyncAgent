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

func (s *Session) LookupTradeBySid(ctx context.Context, platform, sid, tradeStatus string) (*TradeListItem, error) {
	sid = strings.TrimSpace(sid)
	if sid == "" {
		return nil, nil
	}
	if tradeStatus == "" {
		tradeStatus = "shipped"
	}
	timeType := 1
	result, err := s.QueryTrades(ctx, TradeQuery{
		Platform:    platform,
		TradeStatus: tradeStatus,
		Sid:         sid,
		TimeType:    timeType,
		PageNo:      1,
		PageSize:    10,
	})
	if err != nil {
		return nil, err
	}
	for i := range result.Items {
		item := &result.Items[i]
		if tradeItemMatchesSid(item, sid) {
			return item, nil
		}
	}
	if len(result.Items) == 1 {
		return &result.Items[0], nil
	}
	if len(result.Items) > 0 {
		return &result.Items[0], nil
	}
	return nil, nil
}

func tradeItemMatchesSid(item *TradeListItem, sid string) bool {
	if item == nil {
		return false
	}
	want := strings.ToUpper(strings.TrimSpace(sid))
	for _, v := range item.Waybills {
		if strings.ToUpper(strings.TrimSpace(v)) == want {
			return true
		}
	}
	return false
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
