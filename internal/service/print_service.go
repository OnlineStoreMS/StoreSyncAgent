package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"storesyncagent/internal/kdzs"
)

type PrintWaybillQueryItem struct {
	SysTid string `json:"sysTid,omitempty"`
	Tid    string `json:"tid,omitempty"`
}

type PrintWaybillQueryRequest struct {
	Platform string                 `json:"platform"`
	Items    []PrintWaybillQueryItem `json:"items"`
}

type PrintWaybillResult struct {
	SysTid         string `json:"sysTid,omitempty"`
	Tid            string `json:"tid,omitempty"`
	Found          bool   `json:"found"`
	ExpressNo      string `json:"expressNo,omitempty"`
	ExpressCompany string `json:"expressCompany,omitempty"`
	ExpressCode    string `json:"expressCode,omitempty"`
	Message        string `json:"message,omitempty"`
}

func (s *SyncService) ListElecAuth(ctx context.Context) ([]kdzs.ElecAuthRecord, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	return s.client.ListElecAuth(ctx)
}

func (s *SyncService) ListExpressTemplates(ctx context.Context) ([]kdzs.ExpressTemplate, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	userID := strings.TrimSpace(s.session.UserID())
	if userID == "" {
		return nil, fmt.Errorf("missing user id after login")
	}
	return s.client.ListExpressTemplates(ctx, userID)
}

func (s *SyncService) ListSharedExpressAccounts(ctx context.Context, platform string) ([]kdzs.SharedExpressAccount, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	var body any = map[string]any{}
	if p := strings.TrimSpace(platform); p != "" {
		body = map[string]any{"platform": strings.ToUpper(p)}
	}
	return s.client.ListSharedAccounts(ctx, body)
}

// GetBatchPrintURL 返回带 kdzsMallToken 的批打页地址。
// 平台 SPA 从 URL query 读取 kdzsMallToken 写入 session；仅跳转 jumpToMainPage
// 再进 iframe 时 token 常为 null（跨站 Cookie/无 query），会报「token信息为空」。
func (s *SyncService) GetBatchPrintURL(ctx context.Context, platform string) (string, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return "", err
	}
	platform = strings.ToUpper(strings.TrimSpace(platform))
	if platform == "" {
		return "", fmt.Errorf("platform is required")
	}
	switch platform {
	case "HAND", "MANUAL", "DFHAND":
		platform = kdzs.PlatformManual
	case "DY":
		platform = kdzs.PlatformDouyin
	}
	userID := strings.TrimSpace(s.session.UserID())
	if userID == "" {
		return "", fmt.Errorf("missing user id after login")
	}
	mainToken := strings.TrimSpace(s.client.Token())
	if mainToken == "" {
		return "", fmt.Errorf("missing main token after login")
	}

	ps, err := s.session.PlatformSession(ctx, platform)
	if err != nil {
		return "", fmt.Errorf("open platform session: %w", err)
	}
	mallToken := strings.TrimSpace(ps.Token)
	if mallToken == "" {
		mallToken = mainToken
	}
	host := strings.TrimSpace(ps.Host)
	if host == "" {
		host = kdzs.PlatformHost[platform]
	}
	if host == "" {
		return "", fmt.Errorf("unknown platform host for %s", platform)
	}

	platLower := strings.ToLower(platform)
	if platform == kdzs.PlatformDouyin {
		platLower = "fxg"
	}
	if platform == kdzs.PlatformManual {
		platLower = "hand"
	}

	q := url.Values{
		"userId":        {userID},
		"kdzsMallToken": {mallToken},
		"mainToken":     {mainToken},
		"mall":          {platform},
		"platform":      {platLower},
	}
	return fmt.Sprintf("https://%s/newIndex/index.xhtml?%s#/printBatch", host, q.Encode()), nil
}

// QueryPrintWaybills 按 sysTid/tid 查询快递助手订单详情中的运单号与快递公司（打单后回填用）。
func (s *SyncService) QueryPrintWaybills(ctx context.Context, req PrintWaybillQueryRequest) ([]PrintWaybillResult, error) {
	if err := s.ensureLogin(ctx); err != nil {
		return nil, err
	}
	platform := strings.ToUpper(strings.TrimSpace(req.Platform))
	if platform == "" {
		return nil, fmt.Errorf("platform is required")
	}
	switch platform {
	case "HAND", "MANUAL", "DFHAND":
		platform = kdzs.PlatformManual
	case "DY":
		platform = kdzs.PlatformDouyin
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("items is required")
	}

	out := make([]PrintWaybillResult, 0, len(req.Items))
	for _, it := range req.Items {
		sysTid := strings.TrimSpace(it.SysTid)
		tid := strings.TrimSpace(it.Tid)
		row := PrintWaybillResult{SysTid: sysTid, Tid: tid}
		if sysTid == "" && tid == "" {
			row.Message = "缺少 sysTid/tid"
			out = append(out, row)
			continue
		}

		// 手工单建单回写时 SuccessList/SuccessRealList 与详情接口 sysTid/tid 常对调，
		// 因此详情查询需同时尝试两个 ID。
		candidates := uniqueNonEmpty(sysTid, tid)
		if len(candidates) == 0 {
			row.Message = "缺少 sysTid/tid"
			out = append(out, row)
			continue
		}

		var got *PrintWaybillResult
		var lastErr error
		var hint string
		for _, id := range candidates {
			r, err := s.lookupWaybillBySysTid(ctx, platform, id)
			if err != nil {
				lastErr = err
				continue
			}
			if r != nil && r.Found && strings.TrimSpace(r.ExpressNo) != "" {
				got = r
				break
			}
			if r != nil && strings.TrimSpace(r.Message) != "" {
				hint = strings.TrimSpace(r.Message)
			}
		}
		if got == nil && tid != "" {
			item, _, err := s.session.LookupTradeByTid(ctx, platform, tid)
			if err == nil && item != nil {
				if item.ExpressNo != "" {
					resolvedSys := sysTid
					if len(item.SysTids) > 0 {
						resolvedSys = item.SysTids[0]
					}
					got = &PrintWaybillResult{
						SysTid:         resolvedSys,
						Tid:            tid,
						Found:          true,
						ExpressNo:      item.ExpressNo,
						ExpressCompany: item.ExpressCompany,
						ExpressCode:    item.ExpressCode,
					}
				} else if len(item.SysTids) > 0 {
					for _, sid := range item.SysTids {
						r, err := s.lookupWaybillBySysTid(ctx, platform, sid)
						if err != nil {
							lastErr = err
							continue
						}
						if r != nil && r.Found && strings.TrimSpace(r.ExpressNo) != "" {
							got = r
							break
						}
						if r != nil && strings.TrimSpace(r.Message) != "" {
							hint = strings.TrimSpace(r.Message)
						}
					}
				}
			}
		}
		if got == nil {
			switch {
			case hint != "":
				row.Message = hint
			case lastErr != nil:
				row.Message = lastErr.Error()
			default:
				row.Message = "暂无运单号，请确认已在快递助手打印"
			}
			out = append(out, row)
			continue
		}
		row.Found = true
		if got.SysTid != "" {
			row.SysTid = got.SysTid
		}
		row.ExpressNo = got.ExpressNo
		row.ExpressCompany = got.ExpressCompany
		row.ExpressCode = got.ExpressCode
		out = append(out, row)
	}
	return out, nil
}

func uniqueNonEmpty(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (s *SyncService) lookupWaybillBySysTid(ctx context.Context, platform, sysTid string) (*PrintWaybillResult, error) {
	sysTid = strings.TrimSpace(sysTid)
	if sysTid == "" {
		return &PrintWaybillResult{Found: false}, nil
	}
	// 空 status 放前面：部分平台（含 DFHAND）打印/发货后用固定 status 会报「系统订单不存在」
	statuses := []string{"", "wait_send", "shipped", "completed", "wait_audit", "all"}
	var lastErr error
	var sawDetail bool
	var detailHint string
	for _, st := range statuses {
		pkgs, err := s.session.FetchTradeDetails(ctx, platform, st, []string{sysTid})
		if err != nil {
			lastErr = err
			continue
		}
		if len(pkgs) == 0 {
			continue
		}
		sawDetail = true
		for _, raw := range pkgs {
			if got := extractWaybillFromDetailPkg(raw, platform, sysTid); got != nil {
				return got, nil
			}
			if hint := detailMissingWaybillHint(raw); hint != "" {
				detailHint = hint
			}
		}
		// 详情命中但尚未解析出运单号：继续尝试其它状态
	}
	if sawDetail {
		msg := detailHint
		if msg == "" {
			msg = "快递助手订单已找到，但未写入运单号（请确认电子面单打单成功，或手动填写）"
		}
		return &PrintWaybillResult{SysTid: sysTid, Found: false, Message: msg}, nil
	}
	if lastErr != nil {
		// 仅当所有候选都失败时由上层决定是否暴露；这里不阻断换 ID 重试
		return &PrintWaybillResult{SysTid: sysTid, Found: false, Message: lastErr.Error()}, nil
	}
	return &PrintWaybillResult{SysTid: sysTid, Found: false}, nil
}

func detailMissingWaybillHint(raw json.RawMessage) string {
	var pkg map[string]any
	if json.Unmarshal(raw, &pkg) != nil || pkg == nil {
		return ""
	}
	status := strings.ToUpper(firstNonEmptyString(
		asMapString(pkg, "platformOrderStatus"),
		asMapString(pkg, "tradeStatus"),
	))
	consign := asMapString(pkg, "consignTime")
	printTime := asMapString(pkg, "printTime")
	shipped := consign != "" || strings.Contains(status, "SHIP")
	if !shipped {
		return "快递助手订单已找到，但尚未写入运单号，请完成电子面单打印后再同步"
	}
	if printTime == "" {
		return "快递助手显示已发货，但无打印记录/运单号；请确认是否电子面单打单成功，或手动填写单号后确认发货"
	}
	return "快递助手已发货但运单号为空，请在快递助手核对物流信息，或手动填写后确认发货"
}

func extractWaybillFromDetailPkg(raw json.RawMessage, platform, sysTid string) *PrintWaybillResult {
	item := kdzs.ParseTradeItemFromJSON(raw, platform)
	if item != nil && strings.TrimSpace(item.ExpressNo) != "" {
		return &PrintWaybillResult{
			SysTid:         sysTid,
			Found:          true,
			ExpressNo:      item.ExpressNo,
			ExpressCompany: item.ExpressCompany,
			ExpressCode:    item.ExpressCode,
		}
	}
	var pkg map[string]any
	if json.Unmarshal(raw, &pkg) != nil || pkg == nil {
		return nil
	}
	// 批打回显字段可能在包裹层：ydNo / sid / logisticsInfoList
	no := firstNonEmptyString(
		asMapString(pkg, "ydNo"),
		asMapString(pkg, "sid"),
		asMapString(pkg, "trackingNo"),
		asMapString(pkg, "expressNo"),
		asMapString(pkg, "mailNo"),
	)
	company := firstNonEmptyString(
		asMapString(pkg, "companyName"),
		asMapString(pkg, "kdName"),
		asMapString(pkg, "expressName"),
		asMapString(pkg, "exName"),
	)
	code := firstNonEmptyString(
		asMapString(pkg, "company"),
		asMapString(pkg, "kdCode"),
		asMapString(pkg, "cpCode"),
		asMapString(pkg, "exCode"),
	)
	if list, ok := pkg["logisticsInfoList"].([]any); ok {
		for _, rawLg := range list {
			m, _ := rawLg.(map[string]any)
			if m == nil {
				continue
			}
			if no == "" {
				no = firstNonEmptyString(
					asMapString(m, "ydNo"),
					asMapString(m, "trackingNo"),
					asMapString(m, "sid"),
					asMapString(m, "expressNo"),
				)
			}
			if company == "" {
				company = firstNonEmptyString(asMapString(m, "companyName"), asMapString(m, "kdName"))
			}
			if code == "" {
				code = firstNonEmptyString(asMapString(m, "company"), asMapString(m, "kdCode"))
			}
			if no != "" {
				break
			}
		}
	}
	if no == "" {
		return nil
	}
	return &PrintWaybillResult{
		SysTid:         sysTid,
		Found:          true,
		ExpressNo:      no,
		ExpressCompany: company,
		ExpressCode:    code,
	}
}

func asMapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
