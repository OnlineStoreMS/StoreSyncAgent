package kdzs

import (
	"context"
	"encoding/json"
	"fmt"
)

type tradeDetailRequest struct {
	AsyncCode     string   `json:"asyncCode"`
	RDSUser       bool     `json:"rdsUser"`
	Platform      string   `json:"platform"`
	SysTids       []string `json:"sysTids"`
	TradeStatus   string   `json:"tradeStatus,omitempty"`
	FactoryUserID []string `json:"factoryUserId,omitempty"`
}

type tradeDetailResponse struct {
	Result       int             `json:"result"`
	Message      string          `json:"message"`
	ErrorMessage string          `json:"errorMessage"`
	Data         json.RawMessage `json:"data"`
}

func (s *Session) FetchTradeDetails(ctx context.Context, platform, tradeStatus string, sysTids []string) ([]json.RawMessage, error) {
	if len(sysTids) == 0 {
		return nil, nil
	}

	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}

	req := tradeDetailRequest{
		AsyncCode:   "",
		RDSUser:     true,
		Platform:    platform,
		SysTids:     sysTids,
		TradeStatus: tradeStatus,
	}
	if tradeStatus == "wait_audit" {
		if factoryIDs, _, err := s.loadQueryContext(ctx, platform); err == nil && len(factoryIDs) > 0 {
			req.FactoryUserID = factoryIDs
		}
	}

	var resp tradeDetailResponse
	if err := s.client.postPlatform(ctx, ps, "/tradeManage/queryTradeDetail", req, &resp); err != nil {
		return nil, err
	}
	if resp.Result != 0 && resp.Result != ResultSuccess && resp.Result != 101 {
		return nil, fmt.Errorf("%s", firstNonEmpty(resp.Message, resp.ErrorMessage, "query trade detail failed"))
	}

	return extractDetailPackages(resp.Data)
}

func extractDetailPackages(data json.RawMessage) ([]json.RawMessage, error) {
	if len(data) == 0 || string(data) == "null" || string(data) == "false" {
		return nil, nil
	}

	var packages []json.RawMessage
	if err := json.Unmarshal(data, &packages); err == nil {
		return packages, nil
	}

	var wrapped struct {
		Packages []json.RawMessage `json:"packages"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Packages) > 0 {
		return wrapped.Packages, nil
	}

	var single map[string]any
	if err := json.Unmarshal(data, &single); err == nil && len(single) > 0 {
		raw, _ := json.Marshal(single)
		return []json.RawMessage{raw}, nil
	}
	return nil, nil
}
