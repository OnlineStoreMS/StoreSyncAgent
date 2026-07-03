package kdzs

import "strings"

// PlatformHost maps platform code to trade API host (derived from getRedirectUrl).
var PlatformHost = map[string]string{
	PlatformDouyin: "dydf.kdzs.com",
	PlatformTaobao: "tbdf.kdzs.com",
	PlatformXHS:    "xhsdf.kdzs.com",
	"PDD":          "pdddf.kdzs.com",
	"KSXD":         "ksdf.kdzs.com",
}

func PlatformLabel(code string) string {
	switch code {
	case PlatformDouyin:
		return "抖店"
	case PlatformTaobao:
		return "淘宝"
	case PlatformXHS:
		return "小红书"
	case PlatformManual:
		return "手工单"
	case "PDD":
		return "拼多多"
	case "KSXD":
		return "快手"
	default:
		return code
	}
}

func IsEcommercePlatform(code string) bool {
	return code != "" && code != PlatformManual
}

func TradeStatusToAPIStatus(tradeStatus string) string {
	switch strings.ToLower(tradeStatus) {
	case "wait_send":
		return "ORDER_PAID"
	case "wait_audit":
		return "WAIT_AUDIT"
	case "shipped":
		return "SELLER_CONSIGNED"
	case "completed":
		return "TRADE_FINISHED"
	default:
		return "ALL_STATUS"
	}
}

func TradeStatusLabel(tradeStatus string) string {
	switch strings.ToLower(tradeStatus) {
	case "wait_audit":
		return "待推单"
	case "wait_send":
		return "待发货"
	case "order_paid":
		return "已付款"
	case "shipped", "order_shipped", "seller_consigned":
		return "已发货"
	case "completed", "trade_finished":
		return "交易完成"
	case "all":
		return "全部"
	default:
		return tradeStatus
	}
}

func DefaultTradeStatus() string {
	return "wait_audit"
}
