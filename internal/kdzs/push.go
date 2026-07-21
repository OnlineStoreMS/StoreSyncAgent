package kdzs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		msg := firstNonEmpty(resp.Message, resp.ErrorMessage, "set trade agent type failed")
		if len(resp.FailMessage) > 0 {
			for _, v := range resp.FailMessage {
				if strings.TrimSpace(v) != "" {
					msg = v
					break
				}
			}
		}
		if len(resp.FailList) > 0 && len(resp.SuccessList) == 0 {
			return nil, fmt.Errorf("%s", msg)
		}
		// 部分成功仍返回结果，由调用方看 FailList
	}
	if len(resp.FailList) > 0 && len(resp.SuccessList) == 0 {
		msg := firstNonEmpty(resp.Message, resp.ErrorMessage, "set trade agent type failed")
		if len(resp.FailMessage) > 0 {
			for _, v := range resp.FailMessage {
				if strings.TrimSpace(v) != "" {
					msg = v
					break
				}
			}
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return &AgentTypeResult{
		SuccessList: resp.SuccessList,
		FailList:    resp.FailList,
		FailMessage: resp.FailMessage,
	}, nil
}

type CancelTradePushRequest struct {
	Platform    string
	TradeStatus string
	SysTids     []string
}

type cancelTradePushResponse struct {
	Result           int               `json:"result"`
	Message          string            `json:"message"`
	ErrorMessage     string            `json:"errorMessage"`
	AllSuccess       bool              `json:"allSuccess"`
	AllFail          bool              `json:"allFail"`
	SuccessList      []string          `json:"successList"`
	FailList         []string          `json:"failList"`
	FailMessage      map[string]string `json:"failMessageMap"`
	SuccessRealList  []string          `json:"successRealList"`
}

// CancelTradePush 快递助手「批量撤单/退审」：待发货回退到待推单。
func (s *Session) CancelTradePush(ctx context.Context, req CancelTradePushRequest) (*AgentTypeResult, error) {
	if len(req.SysTids) == 0 {
		return nil, fmt.Errorf("sysTids is required")
	}
	tradeStatus := req.TradeStatus
	if tradeStatus == "" {
		tradeStatus = "wait_send"
	}
	tradeInfoList, err := s.BuildTradeInfoList(ctx, req.Platform, tradeStatus, req.SysTids)
	if err != nil {
		return nil, err
	}
	if len(tradeInfoList) == 0 {
		// 已不在待发货：再试待推单，便于幂等（已撤则视为成功）
		if tradeStatus != "wait_audit" {
			tradeInfoList, err = s.BuildTradeInfoList(ctx, req.Platform, "wait_audit", req.SysTids)
			if err != nil {
				return nil, err
			}
		}
		if len(tradeInfoList) == 0 {
			return nil, fmt.Errorf("快递助手未找到可撤单订单")
		}
		// 已在待推单，无需再撤
		return &AgentTypeResult{SuccessList: req.SysTids}, nil
	}

	ps, err := s.PlatformSession(ctx, req.Platform)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"tradeInfoList": tradeInfoList}
	var resp cancelTradePushResponse
	if err := s.client.postPlatform(ctx, ps, "/tradeManage/batchCancelCheck", body, &resp); err != nil {
		return nil, err
	}
	failMsg := firstNonEmpty(resp.Message, resp.ErrorMessage, "")
	if len(resp.FailMessage) > 0 {
		for _, v := range resp.FailMessage {
			if strings.TrimSpace(v) != "" {
				failMsg = v
				break
			}
		}
	}
	// 已在待推/待撤：幂等成功
	if strings.Contains(failMsg, "待撤单") || strings.Contains(failMsg, "待推单") {
		return &AgentTypeResult{SuccessList: req.SysTids, FailMessage: resp.FailMessage}, nil
	}
	ok := resp.Result == 0 || resp.Result == ResultSuccess || resp.Result == 100
	if ok && (resp.AllSuccess || len(resp.SuccessList) > 0) {
		return &AgentTypeResult{
			SuccessList: resp.SuccessList,
			FailList:    resp.FailList,
			FailMessage: resp.FailMessage,
		}, nil
	}
	if failMsg == "" {
		failMsg = "快递助手撤单失败"
	}
	return nil, fmt.Errorf("%s", failMsg)
}
