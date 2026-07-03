package kdzs

import (
	"context"
	"fmt"
)

type FactoryItem struct {
	ID                    int64  `json:"id"`
	FactoryID             string `json:"factoryId"`
	FactoryName           string `json:"factoryName"`
	FactoryNick           string `json:"factoryNick,omitempty"`
	Remark                string `json:"remark,omitempty"`
	BindStatus            int    `json:"bindStatus,omitempty"`
	BindTime              string `json:"bindTime,omitempty"`
	Source                string `json:"source,omitempty"`
	HasPrePushTrade       bool   `json:"hasPrePushTrade,omitempty"`
	SupportBindItem       bool   `json:"supportBindItem,omitempty"`
	SenderName            string `json:"senderName,omitempty"`
	SenderMobile          string `json:"senderMobile,omitempty"`
	SenderAddress         string `json:"senderAddress,omitempty"`
}

type FactoryListResult struct {
	Total    int           `json:"total"`
	PageNo   int           `json:"pageNo"`
	PageSize int           `json:"pageSize"`
	Items    []FactoryItem `json:"items"`
}

type factoryPageListResponse struct {
	Result   int `json:"result"`
	Total    int `json:"total"`
	PageNo   int `json:"pageNo"`
	PageSize int `json:"pageSize"`
	Message  string `json:"message"`
	ErrorMessage string `json:"errorMessage"`
	Data     struct {
		List []FactoryItem `json:"list"`
	} `json:"data"`
}

func (s *Session) ListFactories(ctx context.Context, platform string, pageNo, pageSize int) (*FactoryListResult, error) {
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}
	var resp factoryPageListResponse
	body := map[string]any{
		"pageNo":   pageNo,
		"pageSize": pageSize,
		"platform": platform,
	}
	if err := s.client.postPlatform(ctx, ps, "/agent/factory/pageList", body, &resp); err != nil {
		return nil, err
	}
	if resp.Result != 0 && resp.Result != ResultSuccess && resp.Result != 100 {
		return nil, fmt.Errorf("%s", firstNonEmpty(resp.Message, resp.ErrorMessage, "list factories failed"))
	}
	return &FactoryListResult{
		Total:    resp.Total,
		PageNo:   resp.PageNo,
		PageSize: resp.PageSize,
		Items:    resp.Data.List,
	}, nil
}
