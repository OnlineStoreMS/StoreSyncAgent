package kdzs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type TradeDecryptMeta struct {
	Tid                   string
	AppSource             string
	Oaid                  string
	OwnerShopID           string
	EncodeReceiverName    string
	EncodeReceiverMobile  string
	EncodeReceiverPhone   string
	EncodeReceiverAddress string
	ReceiverProvince      string
	ReceiverCity          string
	ReceiverDistrict      string
	ReceiverTown          string
	ReceiverZip           string
}

type DecryptedReceiver struct {
	ReceiverName    string `json:"receiverName"`
	ReceiverMobile  string `json:"receiverMobile"`
	ReceiverPhone   string `json:"receiverPhone,omitempty"`
	ReceiverAddress string `json:"receiverAddress"`
	IsVirtualTel    bool   `json:"isVirtualTel,omitempty"`
	Extension       string `json:"extension,omitempty"`
	FormattedText   string `json:"formattedText"`
}

type decodePhoneResponse struct {
	Result       json.RawMessage `json:"result"`
	Message      string          `json:"message"`
	ErrorMessage string          `json:"errorMessage"`
	Data         decodePhoneData `json:"data"`
}

type decodePhoneData struct {
	ReceiverName    string `json:"receiverName"`
	Mobile          string `json:"mobile"`
	ReceiverPhone   string `json:"receiverPhone"`
	ReceiverAddress string `json:"receiverAddress"`
	IsVirtualTel    string `json:"isVirtualTel"`
	Extension       string `json:"extension"`
}

func ParseDecryptMeta(pkg map[string]any) TradeDecryptMeta {
	meta := TradeDecryptMeta{
		AppSource:             asString(pkg["appSource"]),
		Oaid:                  asString(pkg["oaid"]),
		OwnerShopID:           asString(pkg["ownerShopId"]),
		EncodeReceiverName:    asString(pkg["encodeReceiverName"]),
		EncodeReceiverMobile:  asString(pkg["encodeReceiverMobile"]),
		EncodeReceiverPhone:   asString(pkg["encodeReceiverPhone"]),
		EncodeReceiverAddress: asString(pkg["encodeReceiverAddress"]),
		ReceiverProvince:      asString(pkg["receiverProvince"]),
		ReceiverCity:          asString(pkg["receiverCity"]),
		ReceiverDistrict:      asString(pkg["receiverDistrict"]),
		ReceiverTown:          asString(pkg["receiverTown"]),
		ReceiverZip:           asString(pkg["receiverZip"]),
	}
	if meta.Tid == "" {
		meta.Tid = asString(pkg["tid"])
	}
	if meta.Tid == "" {
		if orders, ok := pkg["orderDetails"].([]any); ok && len(orders) > 0 {
			if order, _ := orders[0].(map[string]any); order != nil {
				meta.Tid = asString(order["oid"], order["relationTid"], order["tid"])
			}
		}
	}
	return meta
}

func (s *Session) DecodeTradeReceiver(ctx context.Context, platform string, meta TradeDecryptMeta) (*DecryptedReceiver, error) {
	if meta.EncodeReceiverName == "" && meta.EncodeReceiverMobile == "" && meta.EncodeReceiverAddress == "" {
		return nil, fmt.Errorf("missing encrypted receiver fields")
	}
	if meta.Tid == "" {
		return nil, fmt.Errorf("missing platform order id")
	}
	if meta.OwnerShopID == "" {
		return nil, fmt.Errorf("missing shop id")
	}

	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("encodeMobile", meta.EncodeReceiverMobile)
	form.Set("encodeReceiverPhone", meta.EncodeReceiverPhone)
	form.Set("encodeReceiverName", meta.EncodeReceiverName)
	form.Set("encodeReceiverAddress", meta.EncodeReceiverAddress)
	form.Set("caid", meta.Oaid)
	form.Set("tid", meta.Tid)
	form.Set("userId", meta.OwnerShopID)
	if meta.AppSource != "" {
		form.Set("appSource", meta.AppSource)
	}

	var resp decodePhoneResponse
	if err := s.client.postPlatformForm(ctx, ps, "/trade/decodePhone", form, &resp); err != nil {
		return nil, err
	}
	if !decodeResultOK(resp.Result) {
		return nil, fmt.Errorf("%s", firstNonEmpty(resp.Message, resp.ErrorMessage, "decrypt failed"))
	}
	if resp.Data.ReceiverName == "" && resp.Data.Mobile == "" && resp.Data.ReceiverAddress == "" {
		return nil, fmt.Errorf("decrypt returned empty data")
	}

	out := &DecryptedReceiver{
		ReceiverName:    resp.Data.ReceiverName,
		ReceiverMobile:  formatDecodedMobile(resp.Data),
		ReceiverPhone:   resp.Data.ReceiverPhone,
		ReceiverAddress: firstNonEmpty(resp.Data.ReceiverAddress, buildAddressFromMeta(meta)),
		IsVirtualTel:    strings.EqualFold(resp.Data.IsVirtualTel, "true"),
		Extension:       resp.Data.Extension,
	}
	out.FormattedText = FormatReceiverText(out.ReceiverName, out.ReceiverMobile, out.ReceiverPhone, meta.ReceiverZip, out.ReceiverAddress, meta)
	return out, nil
}

func decodeResultOK(result json.RawMessage) bool {
	var n int
	if err := json.Unmarshal(result, &n); err == nil {
		return n == ResultSuccess || n == 101
	}
	var s string
	if err := json.Unmarshal(result, &s); err == nil {
		return s == "100" || s == "101"
	}
	return false
}

func formatDecodedMobile(data decodePhoneData) string {
	mobile := strings.TrimSpace(data.Mobile)
	ext := strings.TrimSpace(data.Extension)
	if mobile == "" {
		return ""
	}
	if ext != "" && strings.EqualFold(data.IsVirtualTel, "true") && !strings.Contains(mobile, "-") {
		return mobile + "-" + ext
	}
	return mobile
}

func buildAddressFromMeta(meta TradeDecryptMeta) string {
	return stringsJoin(meta.ReceiverProvince, meta.ReceiverCity, meta.ReceiverDistrict, meta.ReceiverTown)
}

func FormatReceiverText(name, mobile, phone, zip, address string, meta TradeDecryptMeta) string {
	parts := []string{name, mobile}
	if phone != "" && phone != mobile {
		parts = append(parts, phone)
	}
	addr := formatFullAddress(meta, address)
	if zip != "" && addr != "" && !strings.Contains(addr, zip) {
		addr = zip + " " + addr
	}
	parts = append(parts, addr)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return strings.Join(out, ",")
}

func formatFullAddress(meta TradeDecryptMeta, detail string) string {
	detail = strings.TrimSpace(detail)
	region := stringsJoin(meta.ReceiverProvince, meta.ReceiverCity, meta.ReceiverDistrict, meta.ReceiverTown)
	if detail == "" {
		return region
	}
	if region == "" || strings.Contains(detail, meta.ReceiverProvince) {
		return detail
	}
	if meta.ReceiverTown != "" && strings.HasPrefix(detail, meta.ReceiverTown) {
		return region + detail[len(meta.ReceiverTown):]
	}
	return region + detail
}

func (s *Session) DecodeTradeReceivers(ctx context.Context, platform string, metas []TradeDecryptMeta) (map[string]*DecryptedReceiver, error) {
	out := make(map[string]*DecryptedReceiver, len(metas))
	for _, meta := range metas {
		key := meta.Tid
		if key == "" {
			continue
		}
		decrypted, err := s.DecodeTradeReceiver(ctx, platform, meta)
		if err != nil {
			return out, fmt.Errorf("%s: %w", key, err)
		}
		out[key] = decrypted
	}
	return out, nil
}
