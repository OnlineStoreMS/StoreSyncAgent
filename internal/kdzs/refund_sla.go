package kdzs

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	FXGConfirmReceiveHoursAfterSign = 48
	FXGAgreeHoursAfterApply         = 48
	FXGRefundOnlyUrgeBufferHours    = 12 // 仅退款预留缓冲，应对消费者催促
	FXGDaysAfterAcceptBeforeSign    = 7
)

type RefundSLA struct {
	Scenario            string `json:"scenario,omitempty"`
	DeadlineAt          string `json:"deadlineAt,omitempty"`
	RemainingSeconds    int64  `json:"remainingSeconds,omitempty"`
	RemainingText       string `json:"remainingText,omitempty"`
	Urgency             string `json:"urgency,omitempty"` // critical | warning | normal | expired | unknown | none
	Source              string `json:"source,omitempty"`  // platform_rule | logistics_inferred | apply_time_inferred
	Hint                string `json:"hint,omitempty"`
	LogisticsStatus     string `json:"logisticsStatus,omitempty"`
	LogisticsStatusDesc string `json:"logisticsStatusDesc,omitempty"`
	AcceptTime          string `json:"acceptTime,omitempty"`
	SignTime            string `json:"signTime,omitempty"`
	InboundTime         string `json:"inboundTime,omitempty"`
	IsSigned            bool   `json:"isSigned,omitempty"`
	IsInbound           bool   `json:"isInbound,omitempty"`
	IsPickupPending     bool   `json:"isPickupPending,omitempty"`
	PickupHint          string `json:"pickupHint,omitempty"`
	Important           bool   `json:"important,omitempty"`
}

func ComputeRefundSLA(item *RefundItem, lg *LogisticsDetail, now time.Time) *RefundSLA {
	sla := &RefundSLA{}
	if item == nil {
		return sla
	}

	switch item.AfterSaleStatus {
	case "WAIT_SELLER_AGREE":
		agreeHours := FXGAgreeHoursAfterApply
		hint := fmt.Sprintf("抖店卖家处理时限约%d小时（由申请时间推算）", FXGAgreeHoursAfterApply)
		if item.AfterSaleType == 1 {
			sla.Scenario = "refund_only"
			sla.Important = true
			agreeHours = FXGAgreeHoursAfterApply - FXGRefundOnlyUrgeBufferHours
			hint = fmt.Sprintf("仅退款建议%d小时内处理（平台约%d小时，预留%d小时应对催促）",
				agreeHours, FXGAgreeHoursAfterApply, FXGRefundOnlyUrgeBufferHours)
		} else if item.AfterSaleType == 3 {
			sla.Scenario = "exchange"
		} else {
			sla.Scenario = "wait_agree"
		}
		applyFrom := firstNonEmpty(item.ConfirmTime, item.Created)
		if t, ok := parseKdzsTime(applyFrom); ok {
			deadline := t.Add(time.Duration(agreeHours) * time.Hour)
			fillSLADeadline(sla, deadline, now, "apply_time_inferred", hint)
		} else {
			sla.Urgency = "unknown"
			sla.Hint = "无法获取申请时间，请尽快处理"
		}

	case "WAIT_SELLER_CONFIRM_RECEIVE":
		sla.Scenario = "confirm_receive"
		if item.AfterSaleType == 3 {
			sla.Scenario = "exchange"
		}
		if lg != nil {
			sla.LogisticsStatus = lg.LogisticsStatus
			sla.LogisticsStatusDesc = lg.LogisticsStatusDesc
			sla.AcceptTime = lg.AcceptTime
			sla.SignTime = lg.SignTime
			sla.InboundTime = lg.InboundTime
			sla.IsSigned = lg.IsSigned
			sla.IsInbound = lg.IsInbound
			sla.IsPickupPending = lg.IsPickupPending
			sla.PickupHint = lg.PickupHint
		}
		if lg != nil && lg.IsSigned {
			sla.Scenario = "return_signed"
			if t, ok := parseKdzsTime(lg.SignTime); ok {
				deadline := t.Add(FXGConfirmReceiveHoursAfterSign * time.Hour)
				fillSLADeadline(sla, deadline, now, "platform_rule",
					fmt.Sprintf("抖店规则：签收后%d小时内需确认收货", FXGConfirmReceiveHoursAfterSign))
			} else {
				sla.Urgency = "warning"
				sla.Hint = "物流已签收但无法解析签收时间，请尽快确认收货"
			}
		} else if lg != nil && lg.IsPickupPending {
			sla.Scenario = "pickup_pending"
			fillPreSignDeadline(sla, lg, now, "退货在驿站/快递柜待取件（快递助手可能显示为派件中）")
		} else if lg != nil && lg.AcceptTime != "" {
			fillPreSignDeadline(sla, lg, now, "抖店规则：揽收后7天内需确认收货（未签收）")
		} else if item.Sid != "" {
			sla.Urgency = "unknown"
			sla.Hint = "已填退货物流，揽收后将开始7天确认收货倒计时"
			if lg != nil {
				sla.LogisticsStatusDesc = lg.LogisticsStatusDesc
			}
		} else {
			sla.Urgency = "none"
			sla.Hint = "等待买家填写退货物流"
		}

	case "WAIT_BUYER_RETURN_ITEM":
		sla.Scenario = "wait_return"
		if item.AfterSaleType == 3 {
			sla.Scenario = "exchange"
		}
		sla.Urgency = "none"
		sla.Hint = "等待买家退货"

	case "WAIT_SEND_EXCHANGE_ITEM":
		sla.Scenario = "exchange"
		applyFrom := firstNonEmpty(item.ConfirmTime, item.Created)
		if t, ok := parseKdzsTime(applyFrom); ok {
			deadline := t.Add(FXGAgreeHoursAfterApply * time.Hour)
			fillSLADeadline(sla, deadline, now, "apply_time_inferred", "换货待发出，请尽快处理")
		} else {
			sla.Urgency = "unknown"
			sla.Hint = "换货待发出，请尽快处理"
		}

	case "WAIT_RECEIVE_EXCHANGE_ITEM":
		sla.Scenario = "exchange"
		applyFrom := firstNonEmpty(item.ConfirmTime, item.Created)
		if t, ok := parseKdzsTime(applyFrom); ok {
			deadline := t.Add(FXGAgreeHoursAfterApply * time.Hour)
			fillSLADeadline(sla, deadline, now, "apply_time_inferred", "换货补寄待买家收货")
		} else {
			sla.Urgency = "none"
			sla.Hint = "换货补寄待买家收货"
		}

	case "WAIT_BUYER_MODIFY":
		sla.Scenario = "exchange"
		sla.Urgency = "none"
		sla.Hint = "待买家修改换货申请"

	default:
		sla.Urgency = "none"
	}

	if sla.Urgency == "" {
		sla.Urgency = "none"
	}
	return sla
}

func fillPreSignDeadline(sla *RefundSLA, lg *LogisticsDetail, now time.Time, prefix string) {
	if t, ok := parseKdzsTime(lg.AcceptTime); ok {
		deadline := t.Add(FXGDaysAfterAcceptBeforeSign * 24 * time.Hour)
		fillSLADeadline(sla, deadline, now, "platform_rule", prefix+"，自揽收时间起算")
		return
	}
	sla.Urgency = "warning"
	sla.Hint = prefix + "，暂未解析到揽收时间"
}

func fillSLADeadline(sla *RefundSLA, deadline, now time.Time, source, hint string) {
	sla.Source = source
	sla.Hint = hint
	sla.DeadlineAt = deadline.Format("2006-01-02 15:04:05")
	remaining := int64(deadline.Sub(now).Seconds())
	sla.RemainingSeconds = remaining
	sla.RemainingText = formatRemaining(remaining)
	sla.Urgency = urgencyLevel(remaining)
}

func urgencyLevel(remainingSec int64) string {
	if remainingSec <= 0 {
		return "expired"
	}
	if remainingSec <= 4*3600 {
		return "critical"
	}
	if remainingSec <= 12*3600 {
		return "warning"
	}
	return "normal"
}

// AfterSaleStatusesWithSLADeadline 会产生倒计时 SLA 的售后状态（用于「时效紧迫」全量扫描）。
var AfterSaleStatusesWithSLADeadline = []string{
	"WAIT_SELLER_AGREE",
	"WAIT_SELLER_CONFIRM_RECEIVE",
	"WAIT_SEND_EXCHANGE_ITEM",
	"WAIT_RECEIVE_EXCHANGE_ITEM",
}

// IsUrgentSLA 是否属于时效紧迫（剩余 ≤12h、≤4h 或已超时；含无法解析签收时间等 warning）。
func IsUrgentSLA(sla *RefundSLA) bool {
	if sla == nil {
		return false
	}
	return sla.Urgency == "critical" || sla.Urgency == "expired" || sla.Urgency == "warning"
}

func formatRemaining(sec int64) string {
	if sec <= 0 {
		overdue := -sec
		h := overdue / 3600
		m := (overdue % 3600) / 60
		if h > 0 {
			return fmt.Sprintf("已超时 %d小时%d分", h, m)
		}
		return fmt.Sprintf("已超时 %d分", m)
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	if h >= 24 {
		d := h / 24
		h = h % 24
		return fmt.Sprintf("剩余 %d天%d小时", d, h)
	}
	if h > 0 {
		return fmt.Sprintf("剩余 %dh%dm", h, m)
	}
	return fmt.Sprintf("剩余 %dm", m)
}

func MatchRefundScenario(item RefundItem, scenario string) bool {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return true
	}
	sla := item.SLA
	switch scenario {
	case "confirm_receive":
		return item.AfterSaleStatus == "WAIT_SELLER_CONFIRM_RECEIVE"
	case "wait_agree":
		return item.AfterSaleStatus == "WAIT_SELLER_AGREE"
	case "refund_only":
		return item.AfterSaleType == 1 && activeRefundStatus(item.AfterSaleStatus)
	case "exchange":
		return item.AfterSaleType == 3 && ActiveExchangeStatus(item.AfterSaleStatus)
	case "wait_send_exchange":
		return item.AfterSaleStatus == "WAIT_SEND_EXCHANGE_ITEM"
	case "return_signed":
		if item.AfterSaleStatus != "WAIT_SELLER_CONFIRM_RECEIVE" {
			return false
		}
		return sla != nil && sla.IsSigned
	case "pickup_pending":
		if item.AfterSaleStatus != "WAIT_SELLER_CONFIRM_RECEIVE" {
			return false
		}
		return sla != nil && sla.IsPickupPending && !sla.IsSigned
	case "urgent":
		return IsUrgentSLA(sla)
	default:
		return true
	}
}

func activeRefundStatus(status string) bool {
	switch status {
	case "WAIT_SELLER_AGREE", "WAIT_BUYER_RETURN_ITEM", "WAIT_SELLER_CONFIRM_RECEIVE":
		return true
	default:
		return false
	}
}

func ActiveExchangeStatus(status string) bool {
	switch status {
	case "WAIT_SELLER_AGREE", "WAIT_BUYER_RETURN_ITEM", "WAIT_SELLER_CONFIRM_RECEIVE",
		"WAIT_SEND_EXCHANGE_ITEM", "WAIT_RECEIVE_EXCHANGE_ITEM", "WAIT_BUYER_MODIFY":
		return true
	default:
		return false
	}
}

// ScenarioNeedsLogisticsScan returns true when matching requires logistics enrichment across all pages.
func ScenarioNeedsLogisticsScan(scenario string) bool {
	switch strings.TrimSpace(scenario) {
	case "pickup_pending", "return_signed", "urgent":
		return true
	default:
		return false
	}
}

// ScenarioNeedsFullListScan returns true when scenario tab should load and sort all matching refunds.
func ScenarioNeedsFullListScan(scenario string) bool {
	return strings.TrimSpace(scenario) != ""
}

// SortRefundItemsBySLAUrgency sorts in place: expired/critical first, then by least remaining time.
func SortRefundItemsBySLAUrgency(items []RefundItem) {
	sort.Slice(items, func(i, j int) bool {
		pi, ri := slaSortKey(items[i].SLA)
		pj, rj := slaSortKey(items[j].SLA)
		if pi != pj {
			return pi < pj
		}
		return ri < rj
	})
}

func slaSortKey(sla *RefundSLA) (priority int, remaining int64) {
	if sla == nil {
		return 99, 1 << 62
	}
	switch sla.Urgency {
	case "expired":
		priority = 0
	case "critical":
		priority = 1
	case "warning":
		priority = 2
	case "normal":
		priority = 3
	case "unknown":
		priority = 4
	default:
		priority = 5
	}
	if sla.DeadlineAt != "" || sla.Urgency == "expired" || sla.Urgency == "critical" || sla.Urgency == "warning" || sla.Urgency == "normal" {
		return priority, sla.RemainingSeconds
	}
	return priority, 1<<62 - 1
}
