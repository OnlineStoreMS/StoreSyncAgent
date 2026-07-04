package kdzs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type LogisticsTrace struct {
	Time              string `json:"time,omitempty"`
	Desc              string `json:"desc,omitempty"`
	LogisticsStatus   string `json:"logisticsStatus,omitempty"`
	SubLogisticsStatus string `json:"subLogisticsStatus,omitempty"`
}

type LogisticsDetail struct {
	MailNo               string           `json:"mailNo,omitempty"`
	CpCode               string           `json:"cpCode,omitempty"`
	LogisticsCompanyName string           `json:"logisticsCompanyName,omitempty"`
	LogisticsStatus      string           `json:"logisticsStatus,omitempty"`
	LogisticsStatusDesc  string           `json:"logisticsStatusDesc,omitempty"`
	TraceList            []LogisticsTrace `json:"traceList,omitempty"`
	AcceptTime           string           `json:"acceptTime,omitempty"`
	SignTime             string           `json:"signTime,omitempty"`
	InboundTime          string           `json:"inboundTime,omitempty"`
	IsSigned             bool             `json:"isSigned,omitempty"`
	IsInbound            bool             `json:"isInbound,omitempty"`
	IsPickupPending      bool             `json:"isPickupPending,omitempty"`
	PickupHint           string           `json:"pickupHint,omitempty"`
}

type logisticsDetailResponse struct {
	Result  int    `json:"result"`
	Message string `json:"message"`
	Error   string `json:"error"`
	Data    struct {
		MailNo               string `json:"mailNo"`
		CpCode               string `json:"cpCode"`
		LogisticsCompanyName string `json:"logisticsCompanyName"`
		LogisticsStatus      string `json:"logisticsStatus"`
		LogisticsStatusDesc  string `json:"logisticsStatusDesc"`
		LogisticsTraceDetailList []struct {
			Time               string `json:"time"`
			Desc               string `json:"desc"`
			LogisticsStatus    string `json:"logisticsStatus"`
			SubLogisticsStatus string `json:"subLogisticsStatus"`
		} `json:"logisticsTraceDetailList"`
	} `json:"data"`
}

var signedStatuses = map[string]struct{}{
	"SIGN":       {},
	"AGENT_SIGN": {},
}

var inboundStatuses = map[string]struct{}{
	"STA_INBOUND": {},
}

func (s *Session) GetLogisticsDetail(ctx context.Context, platform, ydNo, kdCode string) (*LogisticsDetail, error) {
	ydNo = strings.TrimSpace(ydNo)
	if ydNo == "" {
		return nil, fmt.Errorf("ydNo is required")
	}
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"ydNo": ydNo}
	if kdCode = strings.TrimSpace(kdCode); kdCode != "" {
		body["kdCode"] = kdCode
	}
	var resp logisticsDetailResponse
	if err := s.client.postPlatform(ctx, ps, "/logistics/getLogisticsDetail", body, &resp); err != nil {
		return nil, err
	}
	if resp.Result != 0 && resp.Result != ResultSuccess {
		return nil, fmt.Errorf("%s", firstNonEmpty(resp.Message, resp.Error, "get logistics failed"))
	}
	return parseLogisticsDetail(&resp), nil
}

var acceptStatuses = map[string]struct{}{
	"ACCEPT": {},
}

func parseLogisticsDetail(resp *logisticsDetailResponse) *LogisticsDetail {
	detail := &LogisticsDetail{
		MailNo:               resp.Data.MailNo,
		CpCode:               resp.Data.CpCode,
		LogisticsCompanyName: resp.Data.LogisticsCompanyName,
		LogisticsStatus:      resp.Data.LogisticsStatus,
		LogisticsStatusDesc:  resp.Data.LogisticsStatusDesc,
	}
	traces := resp.Data.LogisticsTraceDetailList
	for _, t := range traces {
		trace := LogisticsTrace{
			Time:               t.Time,
			Desc:               t.Desc,
			LogisticsStatus:    t.LogisticsStatus,
			SubLogisticsStatus: t.SubLogisticsStatus,
		}
		detail.TraceList = append(detail.TraceList, trace)
		if isAcceptStatus(t.LogisticsStatus, t.SubLogisticsStatus, t.Desc) {
			if detail.AcceptTime == "" || t.Time < detail.AcceptTime {
				detail.AcceptTime = t.Time
			}
		}
		if isSignedStatus(t.LogisticsStatus, t.SubLogisticsStatus, t.Desc) {
			detail.IsSigned = true
			if detail.SignTime == "" || t.Time > detail.SignTime {
				detail.SignTime = t.Time
			}
		}
		if isPickupPendingTrace(t.LogisticsStatus, t.SubLogisticsStatus, t.Desc) {
			detail.IsInbound = true
			detail.IsPickupPending = true
			if detail.InboundTime == "" || t.Time > detail.InboundTime {
				detail.InboundTime = t.Time
				detail.PickupHint = t.Desc
			}
		}
	}
	if !detail.IsSigned && isSignedStatus(detail.LogisticsStatus, "", detail.LogisticsStatusDesc) {
		detail.IsSigned = true
	}
	// 快递助手常把驿站待取件映射为「派件中」，以最新一条轨迹描述为准二次识别。
	if len(traces) > 0 {
		latest := traces[len(traces)-1]
		if isPickupPendingTrace(latest.LogisticsStatus, latest.SubLogisticsStatus, latest.Desc) {
			detail.IsInbound = true
			detail.IsPickupPending = true
			if detail.InboundTime == "" || latest.Time >= detail.InboundTime {
				detail.InboundTime = latest.Time
				detail.PickupHint = latest.Desc
			}
		}
	}
	if isPickupPendingTrace("", "", detail.LogisticsStatusDesc) {
		detail.IsInbound = true
		detail.IsPickupPending = true
	}
	reconcileSignedVsPickupPending(detail)
	return detail
}

func isSignedStatus(status, subStatus, desc string) bool {
	if _, ok := signedStatuses[strings.ToUpper(status)]; ok {
		return true
	}
	if _, ok := signedStatuses[strings.ToUpper(subStatus)]; ok {
		return true
	}
	return isDescSigned(desc)
}

// isDescSigned 轨迹描述中的签收语义（排除派件中「准备签收」等未签收状态）。
func isDescSigned(desc string) bool {
	if !strings.Contains(desc, "签收") {
		return false
	}
	for _, neg := range []string{"待签收", "准备签收", "即将签收", "等候签收"} {
		if strings.Contains(desc, neg) {
			return false
		}
	}
	return true
}

// reconcileSignedVsPickupPending 最新轨迹为驿站/代收待取时，覆盖较早的误签收标记。
func reconcileSignedVsPickupPending(detail *LogisticsDetail) {
	if detail == nil || !detail.IsPickupPending || !detail.IsSigned {
		return
	}
	if detail.InboundTime != "" && detail.SignTime != "" && detail.InboundTime >= detail.SignTime {
		detail.IsSigned = false
		detail.SignTime = ""
		return
	}
	if len(detail.TraceList) == 0 {
		return
	}
	latest := detail.TraceList[len(detail.TraceList)-1]
	if !isPickupPendingTrace(latest.LogisticsStatus, latest.SubLogisticsStatus, latest.Desc) {
		return
	}
	if isDescSigned(latest.Desc) {
		return
	}
	detail.IsSigned = false
	detail.SignTime = ""
}

func isAcceptStatus(status, subStatus, desc string) bool {
	if _, ok := acceptStatuses[strings.ToUpper(status)]; ok {
		return true
	}
	if _, ok := acceptStatuses[strings.ToUpper(subStatus)]; ok {
		return true
	}
	desc = strings.ToLower(desc)
	return strings.Contains(desc, "揽收") ||
		strings.Contains(desc, "收取快件") ||
		strings.Contains(desc, "已收取") ||
		strings.Contains(desc, "已揽件")
}

func isPickupPendingTrace(status, subStatus, desc string) bool {
	if _, ok := inboundStatuses[strings.ToUpper(status)]; ok {
		if !isOutboundLockerEvent(desc) {
			return true
		}
	}
	if _, ok := inboundStatuses[strings.ToUpper(subStatus)]; ok {
		if !isOutboundLockerEvent(desc) {
			return true
		}
	}
	if isOutboundLockerEvent(desc) {
		return false
	}
	return isInboundWaitingPickup(desc)
}

// isOutboundLockerEvent 发货/揽收方向：快递员从丰巢/快递柜取出快件去运输，不是收件待取。
func isOutboundLockerEvent(desc string) bool {
	d := strings.ToLower(desc)
	if strings.Contains(d, "取出快件") || strings.Contains(d, "取出包裹") {
		return true
	}
	if strings.Contains(d, "取出") && (strings.Contains(d, "收件员") || strings.Contains(d, "快递员") || strings.Contains(d, "揽投员")) {
		return true
	}
	if strings.Contains(d, "从") && strings.Contains(d, "取出") &&
		(strings.Contains(d, "丰巢") || strings.Contains(d, "快递柜") || strings.Contains(d, "柜机") || strings.Contains(d, "智能柜")) {
		return true
	}
	return false
}

// isInboundWaitingPickup 收货方向：快件已到达驿站/柜，等待卖家（退货收件方）取件。
func isInboundWaitingPickup(desc string) bool {
	d := strings.ToLower(desc)
	if strings.Contains(d, "待取件") {
		return true
	}
	if strings.Contains(d, "请凭") && strings.Contains(d, "取件码") {
		return true
	}
	if strings.Contains(d, "取件码") && !strings.Contains(d, "取出") {
		return true
	}
	if (strings.Contains(d, "放入") || strings.Contains(d, "存入") || strings.Contains(d, "投入") || strings.Contains(d, "已投") || strings.Contains(d, "投递")) &&
		(strings.Contains(d, "丰巢") || strings.Contains(d, "快递柜") || strings.Contains(d, "柜机") || strings.Contains(d, "智能柜")) {
		return true
	}
	if strings.Contains(d, "暂存") && (strings.Contains(d, "驿站") || strings.Contains(d, "代收") || strings.Contains(d, "自提")) {
		return true
	}
	if strings.Contains(d, "驿站") && (strings.Contains(d, "待取") || strings.Contains(d, "请尽快") || strings.Contains(d, "领取") || strings.Contains(d, "取件")) {
		return true
	}
	if strings.Contains(d, "保管") && (strings.Contains(d, "请") || strings.Contains(d, "取件") || strings.Contains(d, "领取")) {
		return true
	}
	if strings.Contains(d, "已代收") {
		return true
	}
	if (strings.Contains(d, "到达") || strings.Contains(d, "送达")) &&
		(strings.Contains(d, "驿站") || strings.Contains(d, "代收点") || strings.Contains(d, "自提点")) &&
		!strings.Contains(d, "转运") && !strings.Contains(d, "发往") {
		return true
	}
	return false
}

func parseKdzsTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006/01/02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (s *Session) EnrichRefundsLogistics(ctx context.Context, platform string, items []RefundItem, maxConcurrent int) {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	now := time.Now()

	for i := range items {
		if items[i].Sid == "" {
			items[i].SLA = ComputeRefundSLA(&items[i], nil, now)
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			lg, err := s.GetLogisticsDetail(ctx, platform, items[idx].Sid, items[idx].SidCode)
			if err != nil {
				items[idx].SLA = ComputeRefundSLA(&items[idx], nil, now)
				return
			}
			items[idx].SLA = ComputeRefundSLA(&items[idx], lg, now)
		}(i)
	}
	wg.Wait()
}

// RawLogisticsJSON for debugging.
func (s *Session) GetLogisticsDetailRaw(ctx context.Context, platform, ydNo, kdCode string) (map[string]any, error) {
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"ydNo": ydNo}
	if kdCode != "" {
		body["kdCode"] = kdCode
	}
	var resp map[string]any
	if err := s.client.postPlatform(ctx, ps, "/logistics/getLogisticsDetail", body, &resp); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(resp)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out, nil
}
