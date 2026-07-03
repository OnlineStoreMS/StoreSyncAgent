package kdzs

import (
	"context"
	"encoding/json"
)

type FactoryBind struct {
	FactoryID string `json:"factoryId"`
	FactoryName string `json:"factoryName"`
	Remark string `json:"remark"`
}

type MallBind struct {
	MallID string `json:"mallId"`
	MallName string `json:"mallName"`
}

type platformListResponse[T any] struct {
	Result int `json:"result"`
	Data   T   `json:"data"`
}

type factoryListData struct {
	List []FactoryBind `json:"list"`
}

type mallListData struct {
	List []MallBind `json:"list"`
}

func (s *Session) loadQueryContext(ctx context.Context, platform string) (factoryIDs, mallIDs []string, err error) {
	return s.LoadQueryContext(ctx, platform)
}

func (s *Session) LoadQueryContext(ctx context.Context, platform string) (factoryIDs, mallIDs []string, err error) {
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, nil, err
	}

	var factoryResp platformListResponse[factoryListData]
	if err := s.client.postPlatform(ctx, ps, "/agent/factory/list", map[string]any{
		"pageNo": 1, "pageSize": 100,
	}, &factoryResp); err == nil && len(factoryResp.Data.List) > 0 {
		for _, f := range factoryResp.Data.List {
			if f.FactoryID != "" {
				factoryIDs = append(factoryIDs, f.FactoryID)
			}
		}
	}

	var mallResp platformListResponse[mallListData]
	if err := s.client.postPlatform(ctx, ps, "/agent/mall/list", map[string]any{
		"pageNo": 1, "pageSize": 100,
	}, &mallResp); err == nil && len(mallResp.Data.List) > 0 {
		for _, m := range mallResp.Data.List {
			if m.MallID != "" {
				mallIDs = append(mallIDs, m.MallID)
			}
		}
	}
	return factoryIDs, mallIDs, nil
}

type MainPageStats struct {
	WaitingPushOrderNum int            `json:"waitingPushOrderNum"`
	WaitingSendOrderNum int            `json:"waitingSendOrderNum"`
	WaitingPushByPlatform map[string]int `json:"waitingPushByPlatform"`
}

func (c *Client) GetMainPageStats(ctx context.Context) (*MainPageStats, error) {
	var resp APIResponse[struct {
		WaitingPushOrderNum    int            `json:"waitingPushOrderNum"`
		WaitingSendOrderNum    int            `json:"waitingSendOrderNum"`
		WaitingPushOrderDetail map[string]int `json:"waitingPushOrderDetail"`
	}]
	if err := c.post(ctx, "/factory/management/mainPageData", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	data, err := checkResult(&resp)
	if err != nil {
		return nil, err
	}
	return &MainPageStats{
		WaitingPushOrderNum:   data.WaitingPushOrderNum,
		WaitingSendOrderNum:   data.WaitingSendOrderNum,
		WaitingPushByPlatform: data.WaitingPushOrderDetail,
	}, nil
}

type WaitSendCount struct {
	WaitAudit int `json:"waitAudit"`
	WaitSend  int `json:"waitSend"`
}

func (s *Session) GetWaitSendCount(ctx context.Context, platform string, factoryIDs, mallIDs, shopIDs []string) (*WaitSendCount, error) {
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}
	if len(shopIDs) == 0 {
		shopIDs, _ = s.platformShopIDs(ctx, platform)
	}
	start, end := DefaultDateRange()
	body := map[string]any{
		"startDateTime": start,
		"endDateTime":   end,
		"shopIds":       shopIDs,
	}
	var resp APIResponse[map[string]json.RawMessage]
	if err := s.client.postPlatform(ctx, ps, "/tradeManage/queryWaitSendCount", body, &resp); err != nil {
		return nil, err
	}
	data, err := checkResult(&resp)
	if err != nil {
		return nil, err
	}
	out := &WaitSendCount{}
	if raw, ok := data["wait_audit"]; ok {
		var item struct {
			TradeTotal int `json:"tradeTotal"`
		}
		_ = json.Unmarshal(raw, &item)
		out.WaitAudit = item.TradeTotal
	}
	if raw, ok := data["wait_send"]; ok {
		var item struct {
			TradeTotal int `json:"tradeTotal"`
		}
		_ = json.Unmarshal(raw, &item)
		out.WaitSend = item.TradeTotal
	}
	return out, nil
}
