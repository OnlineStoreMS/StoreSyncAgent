package kdzs

import (
	"strings"
	"unicode"
)

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
		return PlatformOrderStatusLabel(tradeStatus)
	}
}

// PlatformOrderStatusLabel maps platform raw status codes to 快递助手-style Chinese labels.
func PlatformOrderStatusLabel(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "WAIT_AUDIT":
		return "待推单"
	case "ORDER_PAID", "WAIT_SELLER_SEND_GOODS", "WAIT_SELLER_STOCK_OUT", "PAID":
		return "待发货"
	case "SELLER_CONSIGNED", "ORDER_SHIPPED", "WAIT_BUYER_CONFIRM_GOODS", "SHIPPED":
		return "已发货"
	case "ORDER_COMPLETED", "TRADE_FINISHED", "COMPLETED", "FINISHED", "SUCCESS":
		return "交易完成"
	case "ORDER_CANCEL", "TRADE_CLOSED", "CANCEL", "CLOSED":
		return "交易关闭"
	case "WAIT_BUYER_PAY", "UNPAID":
		return "待付款"
	case "REFUNDING", "REFUND":
		return "退款中"
	default:
		if looksLikeChineseLabel(code) {
			return code
		}
		lower := strings.ToLower(code)
		switch lower {
		case "wait_audit":
			return "待推单"
		case "wait_send":
			return "待发货"
		case "shipped":
			return "已发货"
		case "completed":
			return "交易完成"
		}
		return code
	}
}

// ResolveOrderStatusText prefers existing Chinese description, else maps platform codes.
func ResolveOrderStatusText(statusText, tradeStatus string) string {
	if t := strings.TrimSpace(statusText); t != "" && !looksLikeStatusCode(t) {
		return t
	}
	for _, raw := range []string{tradeStatus, statusText} {
		if t := strings.TrimSpace(raw); t != "" {
			if label := PlatformOrderStatusLabel(t); label != t || looksLikeChineseLabel(label) {
				return label
			}
		}
	}
	return strings.TrimSpace(statusText)
}

func looksLikeStatusCode(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if looksLikeChineseLabel(s) {
		return false
	}
	upper := strings.ToUpper(s)
	if upper == s && strings.Contains(s, "_") {
		return true
	}
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return strings.Contains(s, "_")
}

func looksLikeChineseLabel(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func DefaultTradeStatus() string {
	return "wait_audit"
}
