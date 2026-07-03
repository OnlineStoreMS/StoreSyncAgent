package kdzs

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	AgentTypeSelfPrint = 1
	AgentTypePushFactory = 2
)

type TradeInfoItem struct {
	SysTid    string   `json:"sysTid"`
	Tid       string   `json:"tid,omitempty"`
	ShopID    string   `json:"shopId,omitempty"`
	ItemID    string   `json:"itemId,omitempty"`
	SkuIDList []string `json:"skuIdList,omitempty"`
	OidList   []string `json:"oidList,omitempty"`
	Split     bool     `json:"split,omitempty"`
}

type SetTradeAgentTypeRequest struct {
	Platform    string
	TradeStatus string
	AgentType   int
	FactoryID   string
	SysTids     []string
}

type AgentTypeResult struct {
	SuccessList []string          `json:"successList"`
	FailList    []string          `json:"failList"`
	FailMessage map[string]string `json:"failMessageMap,omitempty"`
}

type setTradeAgentTypeResponse struct {
	Result       int               `json:"result"`
	Message      string            `json:"message"`
	ErrorMessage string            `json:"errorMessage"`
	SuccessList  []string          `json:"successList"`
	FailList     []string          `json:"failList"`
	FailMessage  map[string]string `json:"failMessageMap"`
}

func (s *Session) BuildTradeInfoList(ctx context.Context, platform, tradeStatus string, sysTids []string) ([]TradeInfoItem, error) {
	pkgs, err := s.FetchTradeDetails(ctx, platform, tradeStatus, sysTids)
	if err != nil {
		return nil, err
	}
	out := make([]TradeInfoItem, 0, len(pkgs))
	for _, raw := range pkgs {
		item, err := parseTradeInfoItem(raw)
		if err != nil {
			return out, fmt.Errorf("%w", err)
		}
		out = append(out, item)
	}
	return out, nil
}

func parseTradeInfoItem(raw json.RawMessage) (TradeInfoItem, error) {
	var pkg map[string]any
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return TradeInfoItem{}, err
	}
	item := TradeInfoItem{
		SysTid: asString(pkg["sysTid"]),
		Tid:    asString(pkg["tid"]),
		ShopID: asString(pkg["ownerShopId"], pkg["shopId"]),
	}
	if item.SysTid == "" {
		return item, fmt.Errorf("missing sysTid")
	}
	orders, _ := pkg["orderDetails"].([]any)
	for _, o := range orders {
		order, _ := o.(map[string]any)
		if order == nil {
			continue
		}
		if item.ItemID == "" {
			item.ItemID = asString(order["itemId"])
		}
		if oid := asString(order["oid"]); oid != "" {
			item.OidList = appendUnique(item.OidList, oid)
		}
		if sku := asString(order["skuId"], order["itemId"]); sku != "" {
			item.SkuIDList = appendUnique(item.SkuIDList, sku)
		}
	}
	if item.Tid == "" && len(item.OidList) > 0 {
		item.Tid = item.OidList[0]
	}
	return item, nil
}

func (s *Session) SetTradeAgentType(ctx context.Context, req SetTradeAgentTypeRequest) (*AgentTypeResult, error) {
	if len(req.SysTids) == 0 {
		return nil, fmt.Errorf("sysTids is required")
	}
	if req.AgentType != AgentTypeSelfPrint && req.AgentType != AgentTypePushFactory {
		return nil, fmt.Errorf("invalid agentType")
	}
	if req.AgentType == AgentTypePushFactory && req.FactoryID == "" {
		return nil, fmt.Errorf("factoryId is required for push")
	}
	tradeStatus := req.TradeStatus
	if tradeStatus == "" {
		tradeStatus = DefaultTradeStatus()
	}
	tradeInfoList, err := s.BuildTradeInfoList(ctx, req.Platform, tradeStatus, req.SysTids)
	if err != nil {
		return nil, err
	}
	if len(tradeInfoList) == 0 {
		return nil, fmt.Errorf("no trade info built")
	}

	ps, err := s.PlatformSession(ctx, req.Platform)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"tradeInfoList": tradeInfoList,
		"agentType":     req.AgentType,
	}
	if req.AgentType == AgentTypePushFactory {
		body["factoryId"] = req.FactoryID
	}
	var resp setTradeAgentTypeResponse
	if err := s.client.postPlatform(ctx, ps, "/tradeManage/setTradeAgentType", body, &resp); err != nil {
		return nil, err
	}
	if resp.Result != 0 && resp.Result != ResultSuccess && resp.Result != 100 {
		return nil, fmt.Errorf("%s", firstNonEmpty(resp.Message, resp.ErrorMessage, "set trade agent type failed"))
	}
	return &AgentTypeResult{
		SuccessList: resp.SuccessList,
		FailList:    resp.FailList,
		FailMessage: resp.FailMessage,
	}, nil
}
