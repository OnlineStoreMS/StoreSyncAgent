package kdzs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ManualShipRequest 快递助手「手动填写单号」发货。
type ManualShipRequest struct {
	Platform       string
	TradeStatus    string // 默认 wait_send
	SysTid         string
	Tid            string
	ExpressCompany string // 中通
	ExpressCode    string // ZTO，可空则按公司名推断
	ExpressNo      string
}

// ManualShipResult 发货结果。
type ManualShipResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	TogetherID string `json:"togetherId,omitempty"`
	ExpressNo string `json:"expressNo,omitempty"`
	Company   string `json:"company,omitempty"`
}

type deliverySendResponse struct {
	Result       int             `json:"result"`
	Message      string          `json:"message"`
	ErrorMessage string          `json:"errorMessage"`
	Data         json.RawMessage `json:"data"`
}

type deliverySendData struct {
	TogetherID string `json:"togetherId"`
	TradeInfo  []struct {
		Success    bool     `json:"success"`
		Message    string   `json:"message"`
		Tid        string   `json:"tid"`
		TogetherID string   `json:"togetherId"`
		BuyerNick  string   `json:"buyerNick"`
		Oids       []string `json:"oids"`
	} `json:"tradeInfo"`
}

// common express code aliases used by 手动填写单号.
var expressCodeByName = map[string]string{
	"中通": "ZTO", "中通快递": "ZTO", "圆通": "YTO", "圆通速递": "YTO",
	"韵达": "YUNDA", "韵达快递": "YUNDA", "申通": "STO", "申通快递": "STO",
	"极兔": "LB", "极兔速递": "LB", "顺丰": "SF", "顺丰速运": "SF",
	"邮政快递包裹": "POSTB", "德邦": "DBKD", "德邦快递": "DBKD",
	"京东": "JD", "京东快递": "JD", "EMS": "EMS",
}

// ManualShip 调用 /delivery/send，等价于打单发货里「手动填写单号」后发货。
func (s *Session) ManualShip(ctx context.Context, req ManualShipRequest) (*ManualShipResult, error) {
	expressNo := strings.TrimSpace(req.ExpressNo)
	if expressNo == "" {
		return nil, fmt.Errorf("expressNo is required")
	}
	if strings.TrimSpace(req.SysTid) == "" && strings.TrimSpace(req.Tid) == "" {
		return nil, fmt.Errorf("sysTid or tid is required")
	}
	platform := strings.TrimSpace(req.Platform)
	if platform == "" {
		return nil, fmt.Errorf("platform is required")
	}
	tradeStatus := strings.TrimSpace(req.TradeStatus)
	if tradeStatus == "" {
		tradeStatus = "wait_send"
	}
	code := strings.TrimSpace(req.ExpressCode)
	name := strings.TrimSpace(req.ExpressCompany)
	if code == "" {
		code = expressCodeByName[name]
	}
	if code == "" && name != "" {
		// allow passing code in company field
		code = strings.ToUpper(name)
	}
	if name == "" {
		name = code
	}
	if code == "" {
		return nil, fmt.Errorf("express company code is required")
	}

	sysTid := strings.TrimSpace(req.SysTid)
	tidHint := strings.TrimSpace(req.Tid)
	if sysTid == "" && tidHint != "" {
		if item, _, err := s.LookupTradeByTid(ctx, platform, tidHint); err == nil && item != nil && len(item.SysTids) > 0 {
			sysTid = item.SysTids[0]
		}
	}
	if sysTid == "" {
		return nil, fmt.Errorf("sysTid is required (无法从 tid 解析)")
	}
	lookups := []string{tradeStatus, "wait_send", "wait_audit", "all", "shipped"}
	var trade map[string]any
	var lastErr error
	for _, st := range lookups {
		pkgs, err := s.FetchTradeDetails(ctx, platform, st, []string{sysTid})
		if err != nil {
			lastErr = err
			continue
		}
		for _, raw := range pkgs {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil || m == nil {
				continue
			}
			if asString(m["sysTid"]) != "" && asString(m["sysTid"]) != sysTid {
				continue
			}
			if tidHint != "" && asString(m["tid"]) != tidHint && !oidsContain(m, tidHint) {
				continue
			}
			trade = m
			break
		}
		if trade != nil {
			break
		}
	}
	if trade == nil {
		if lastErr != nil {
			return nil, fmt.Errorf("查询快递助手订单失败: %w", lastErr)
		}
		return nil, fmt.Errorf("快递助手未找到订单")
	}

	// 已发货且单号一致：幂等成功
	if already := matchShippedTracking(trade, expressNo); already != nil {
		return already, nil
	}

	meta := ParseDecryptMeta(trade)
	dec, _ := s.DecodeTradeReceiver(ctx, platform, meta)
	shipInfo := buildManualShipInfo(trade, dec, code, name, expressNo)

	shipInfosJSON, err := json.Marshal([]any{shipInfo})
	if err != nil {
		return nil, err
	}
	ps, err := s.PlatformSession(ctx, platform)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("shipInfos", string(shipInfosJSON))
	form.Set("shipType", "1")
	form.Set("multiPack", "false")
	form.Set("multipleYdNo", "0")
	form.Set("uuid", newShipUUID())

	var resp deliverySendResponse
	if err := s.client.postPlatformForm(ctx, ps, "/delivery/send", form, &resp); err != nil {
		return nil, err
	}

	result := &ManualShipResult{
		ExpressNo: expressNo,
		Company:   name,
		TogetherID: asString(shipInfo["togetherId"]),
	}
	var data deliverySendData
	_ = json.Unmarshal(resp.Data, &data)
	if data.TogetherID != "" {
		result.TogetherID = data.TogetherID
	}
	ok := false
	msg := firstNonEmpty(resp.Message, resp.ErrorMessage)
	for _, ti := range data.TradeInfo {
		if ti.Success {
			ok = true
		} else if strings.TrimSpace(ti.Message) != "" {
			msg = ti.Message
		}
	}
	// 助手偶发 KddRecord 写库失败但仍已向平台发货：再查详情确认
	if !ok {
		time.Sleep(500 * time.Millisecond)
		if verify, _ := s.verifyShipped(ctx, platform, asString(trade["sysTid"]), expressNo); verify != nil && verify.Success {
			return verify, nil
		}
	}
	if ok || resp.Result == 0 || resp.Result == ResultSuccess || resp.Result == 100 || resp.Result == 101 {
		result.Success = true
		if msg == "" || strings.Contains(msg, "SqlMapClient") || strings.Contains(msg, "factory_user_id") {
			msg = fmt.Sprintf("已发货 %s %s", name, expressNo)
		}
		result.Message = msg
		return result, nil
	}
	if msg == "" {
		msg = fmt.Sprintf("发货失败 result=%d", resp.Result)
	}
	// 非待发货但单号已存在
	if strings.Contains(msg, "非【待发货】") || strings.Contains(msg, "非待发货") {
		if verify, _ := s.verifyShipped(ctx, platform, asString(trade["sysTid"]), expressNo); verify != nil {
			return verify, nil
		}
	}
	return &ManualShipResult{Success: false, Message: msg, ExpressNo: expressNo, Company: name}, fmt.Errorf("%s", msg)
}

func (s *Session) verifyShipped(ctx context.Context, platform, sysTid, expressNo string) (*ManualShipResult, error) {
	if sysTid == "" {
		return nil, nil
	}
	for _, st := range []string{"shipped", "wait_send", "all"} {
		pkgs, err := s.FetchTradeDetails(ctx, platform, st, []string{sysTid})
		if err != nil || len(pkgs) == 0 {
			continue
		}
		var trade map[string]any
		if err := json.Unmarshal(pkgs[0], &trade); err != nil {
			continue
		}
		if r := matchShippedTracking(trade, expressNo); r != nil {
			return r, nil
		}
	}
	return nil, nil
}

func matchShippedTracking(trade map[string]any, expressNo string) *ManualShipResult {
	expressNo = strings.TrimSpace(expressNo)
	if expressNo == "" || trade == nil {
		return nil
	}
	status := strings.ToUpper(asString(trade["platformOrderStatus"], trade["tradeStatus"]))
	shipped := strings.Contains(status, "SHIP") || status == "SELLER_CONSIGNED"
	no, company := extractLogistics(trade)
	if no == expressNo || (shipped && no != "" && strings.EqualFold(no, expressNo)) {
		return &ManualShipResult{
			Success:    true,
			Message:    fmt.Sprintf("已发货 %s %s", firstNonEmpty(company, asString(trade["expressCompany"])), no),
			ExpressNo:  no,
			Company:    company,
			TogetherID: firstNonEmpty(asString(trade["togetherId"]), asString(trade["tid"])),
		}
	}
	if shipped && no == expressNo {
		return &ManualShipResult{Success: true, Message: "已发货", ExpressNo: no, Company: company}
	}
	return nil
}

func extractLogistics(trade map[string]any) (no, company string) {
	no = asString(trade["expressNo"], trade["ydNo"], trade["trackingNo"])
	company = asString(trade["expressCompany"], trade["companyName"])
	if list, ok := trade["logisticsInfoList"].([]any); ok {
		for _, it := range list {
			m, _ := it.(map[string]any)
			if m == nil {
				continue
			}
			n := asString(m["trackingNo"], m["ydNo"], m["expressNo"])
			c := asString(m["companyName"], m["company"])
			if n != "" {
				return n, c
			}
		}
	}
	return no, company
}

func buildManualShipInfo(trade map[string]any, dec *DecryptedReceiver, kdCode, kdName, ydNo string) map[string]any {
	tid := asString(trade["tid"])
	sysTid := asString(trade["sysTid"])
	togetherID := firstNonEmpty(asString(trade["togetherId"]), tid, sysTid)

	oids := make([]string, 0, 4)
	goods := make([]map[string]any, 0, 4)
	num := 0
	if ods, ok := trade["orderDetails"].([]any); ok {
		for _, o := range ods {
			om, _ := o.(map[string]any)
			if om == nil {
				continue
			}
			oid := asString(om["oid"])
			if oid == "" {
				continue
			}
			oids = append(oids, oid)
			n := anyToInt(om["num"])
			if n <= 0 {
				n = 1
			}
			num += n
			goods = append(goods, map[string]any{
				"oid": oid, "goodsId": asString(om["itemId"], om["numIid"]),
				"skuId": asString(om["skuId"]), "goodsNum": n,
			})
		}
	}
	if num == 0 {
		num = anyToInt(trade["num"])
		if num <= 0 {
			num = 1
		}
	}

	name := asString(trade["receiverName"])
	mobile := asString(trade["receiverMobile"])
	addr := asString(trade["receiverTown"]) + asString(trade["receiverAddress"])
	if dec != nil {
		if dec.ReceiverName != "" {
			name = dec.ReceiverName
		}
		if dec.ReceiverMobile != "" {
			mobile = dec.ReceiverMobile
		}
		if dec.ReceiverAddress != "" {
			addr = dec.ReceiverAddress
		}
	}

	return map[string]any{
		"tradeType":          asString(trade["tradeType"]),
		"userId":             trade["userId"],
		"ownerUserId":        trade["ownerUserId"],
		"factoryUserId":      trade["factoryUserId"],
		"togetherId":         togetherID,
		"buyerNick":          asString(trade["buyerNick"]),
		"sellerFlag":         "",
		"sellerMemo":         asString(trade["sellerMemo"]),
		"info":               "",
		"status":             firstNonEmpty(asString(trade["platformOrderStatus"]), "ORDER_PAID"),
		"nums":               num,
		"payment":            fmt.Sprint(trade["payment"]),
		"buyerMessage":       asString(trade["buyerMessage"]),
		"kdCode":             kdCode,
		"exName":             kdName,
		"kdName":             kdName,
		"logisticsCompanyId": 9999,
		"kdId":               9999,
		"ydNo":               ydNo,
		"ydNos":              []string{ydNo},
		"kddType":            "",
		"templateId":         -9999,
		"optionType":         1,
		"isCode":             "false",
		"tel":                asString(trade["receiverPhone"]),
		"mobile":             mobile,
		"receiverZip":        asString(trade["receiverZip"]),
		"receiverAddress":    addr,
		"receiverCity":       asString(trade["receiverCity"]),
		"receiverCounty":     asString(trade["receiverDistrict"]),
		"receiverName":       name,
		"receiverProvince":   asString(trade["receiverProvince"]),
		"weight":             "0",
		"daifaPrint":         0,
		"asyncSendStatus":    firstNonEmpty(asString(trade["asyncSendStatus"]), "1"),
		"sendRemindHour":     firstNonEmpty(asString(trade["sendRemindHour"]), "0"),
		"distributorId":      "",
		"mallUserId":         asString(trade["mallUserId"], trade["ownerShopId"]),
		"sysTid":             sysTid,
		"goodsInfo":          goods,
		"orders":             goods,
		"tidAndSysTidMap":    map[string]any{tid: []string{sysTid}},
		"shareChainUserId":   "",
		"appSource":          firstNonEmpty(asString(trade["appSource"]), "KDZSFXDF"),
		"isSplit":            false,
		"tidList":            []map[string]any{{"tid": tid, "oids": oids, "split": false}},
		"caid":               asString(trade["oaid"], trade["caid"]),
		"ttCode":             1,
	}
}

func oidsContain(trade map[string]any, tid string) bool {
	if ods, ok := trade["orderDetails"].([]any); ok {
		for _, o := range ods {
			om, _ := o.(map[string]any)
			if om != nil && asString(om["oid"]) == tid {
				return true
			}
		}
	}
	return false
}

func anyToInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		var n int
		_, _ = fmt.Sscanf(t, "%d", &n)
		return n
	default:
		return 0
	}
}

func newShipUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
