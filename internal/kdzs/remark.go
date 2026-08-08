package kdzs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	MemoTypeSeller  = "sellerMemo"  // type=1 卖家备注
	MemoTypeFenFa   = "fenFaMemo"   // type=2 分发备注
	MemoTypePrinter = "printerMemo" // type=3 打单备注
)

var memoTypeCode = map[string]string{
	MemoTypeSeller:  "1",
	MemoTypeFenFa:   "2",
	MemoTypePrinter: "3",
}

// UpdateTradeRemarkRequest 写回快递助手卖家/分发/打单备注。
type UpdateTradeRemarkRequest struct {
	Platform    string
	TradeStatus string
	SysTids     []string
	MemoType    string // sellerMemo | fenFaMemo | printerMemo
	Remark      string
	// SellerFlag 卖家备注旗帜（0灰 1红 2黄 3绿 4蓝 5紫）；仅 sellerMemo 生效，nil 表示沿用原旗
	SellerFlag *int
}

type updateRemarkResponse struct {
	Result       int    `json:"result"`
	Message      string `json:"message"`
	ErrorMessage string `json:"errorMessage"`
	Data         *struct {
		Success []map[string]any `json:"success"`
		Failure []struct {
			Tid    string `json:"tid"`
			ErrMsg string `json:"errMsg"`
		} `json:"failure"`
	} `json:"data"`
}

// UpdateTradeRemark 调用 /tradeManage/batchUpdateStarAndRemark 写回备注。
func (s *Session) UpdateTradeRemark(ctx context.Context, req UpdateTradeRemarkRequest) error {
	if len(req.SysTids) == 0 {
		return fmt.Errorf("sysTids is required")
	}
	code, ok := memoTypeCode[strings.TrimSpace(req.MemoType)]
	if !ok {
		return fmt.Errorf("unsupported memoType %q", req.MemoType)
	}
	remark := strings.TrimSpace(req.Remark)
	// 打单/分发备注在助手侧不允许空；卖家备注允许空（清备注）
	if remark == "" && req.MemoType != MemoTypeSeller {
		return fmt.Errorf("备注不能为空")
	}

	statuses := remarkTradeStatuses(req.TradeStatus)
	var list []map[string]any
	var lastErr error
	for _, st := range statuses {
		pkgs, err := s.FetchTradeDetails(ctx, req.Platform, st, req.SysTids)
		if err != nil {
			lastErr = err
			continue
		}
		for _, raw := range pkgs {
			list = append(list, parseRemarkUpdateItems(raw, remark, req.MemoType)...)
		}
		if len(list) > 0 {
			break
		}
	}
	if len(list) == 0 {
		if lastErr != nil {
			return fmt.Errorf("查询订单详情失败: %w", lastErr)
		}
		return fmt.Errorf("快递助手未找到可写回备注的订单")
	}
	if req.MemoType == MemoTypeSeller && req.SellerFlag != nil {
		star := strconv.Itoa(*req.SellerFlag)
		for _, item := range list {
			item["star"] = star
		}
	}

	ps, err := s.PlatformSession(ctx, req.Platform)
	if err != nil {
		return err
	}
	// 前端 axios 默认 form-urlencoded，嵌套对象会 JSON.stringify 后再 qs 编码；
	// 直接发 JSON body 时后端读不到 remark，会返回「备注信息不能未空」。
	listJSON, err := json.Marshal(list)
	if err != nil {
		return fmt.Errorf("marshal remark list: %w", err)
	}
	form := url.Values{}
	form.Set("starAndRemarkList", string(listJSON))
	form.Set("type", code)
	var resp updateRemarkResponse
	if err := s.client.postPlatformForm(ctx, ps, "/tradeManage/batchUpdateStarAndRemark", form, &resp); err != nil {
		return err
	}
	if resp.Result == 102 {
		return fmt.Errorf("修改备注不成功，请重试")
	}
	if resp.Result != 0 && resp.Result != ResultSuccess && resp.Result != 100 && resp.Result != 101 {
		return fmt.Errorf("%s", firstNonEmpty(resp.Message, resp.ErrorMessage, "写回备注失败"))
	}
	if resp.Data != nil && len(resp.Data.Failure) > 0 && len(resp.Data.Success) == 0 {
		msg := resp.Data.Failure[0].ErrMsg
		if strings.TrimSpace(msg) == "" {
			msg = firstNonEmpty(resp.Message, resp.ErrorMessage, "写回备注失败")
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func remarkTradeStatuses(preferred string) []string {
	preferred = strings.TrimSpace(preferred)
	out := make([]string, 0, 5)
	seen := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(preferred)
	for _, s := range []string{"wait_audit", "wait_send", "shipped", "completed", "all"} {
		add(s)
	}
	return out
}

func parseRemarkUpdateItems(raw json.RawMessage, remark, memoType string) []map[string]any {
	var pkg map[string]any
	if err := json.Unmarshal(raw, &pkg); err != nil || pkg == nil {
		return nil
	}
	togetherID := asString(pkg["togetherId"])
	out := make([]map[string]any, 0, 2)

	if trades, ok := pkg["trades"].([]any); ok && len(trades) > 0 {
		for _, t := range trades {
			trade, _ := t.(map[string]any)
			if trade == nil {
				continue
			}
			out = append(out, expandRemarkItems(trade, togetherID, remark, memoType)...)
		}
		return out
	}
	return expandRemarkItems(pkg, togetherID, remark, memoType)
}

func expandRemarkItems(trade map[string]any, togetherID, remark, memoType string) []map[string]any {
	base := buildRemarkItem(trade, togetherID, remark, memoType)
	sysTid := asString(base["sysTid"])
	if sysTid == "" {
		return nil
	}
	// 合单多 sysTid：助手侧按 sysTidList 展开
	if arr, ok := trade["sysTidList"].([]any); ok && len(arr) > 0 {
		out := make([]map[string]any, 0, len(arr))
		for _, v := range arr {
			sid := asString(v)
			if sid == "" {
				continue
			}
			item := cloneStringMap(base)
			item["sysTid"] = sid
			out = append(out, item)
		}
		if len(out) > 0 {
			return out
		}
	}
	return []map[string]any{base}
}

func buildRemarkItem(trade map[string]any, togetherID, remark, memoType string) map[string]any {
	tid := asString(trade["tid"])
	sysTid := asString(trade["sysTid"])
	if tid == "" {
		if orders, ok := trade["orderDetails"].([]any); ok {
			for _, o := range orders {
				order, _ := o.(map[string]any)
				if order == nil {
					continue
				}
				if tid == "" {
					tid = asString(order["oid"], order["tid"])
				}
				if sysTid == "" {
					sysTid = asString(order["sysTid"])
				}
			}
		}
	}
	if togetherID == "" {
		togetherID = asString(trade["togetherId"])
	}
	if togetherID == "" {
		// 详情接口常无 togetherId；助手侧用 tid 作为合单键
		togetherID = firstNonEmpty(tid, sysTid)
	}
	// FXG 等平台 mallUserId 可能为空，店铺维度用 ownerShopId；勿用助手账号 userId
	userID := asString(trade["mallUserId"], trade["ownerShopId"], trade["shopId"], trade["userId"])
	item := map[string]any{
		"remark":     remark,
		"tid":        tid,
		"togetherId": togetherID,
		"userId":     userID,
		"sysTid":     sysTid,
	}
	if sub := asString(trade["outerUserId"], trade["substituteUserId"]); sub != "" {
		item["substituteUserId"] = sub
	}
	if memoType == MemoTypeSeller {
		star := asString(trade["sellerFlag"], trade["sellerFlagId"], trade["flagId"])
		if star == "" {
			star = "0"
		}
		item["star"] = star
		if tag := asString(trade["sellerFlagTag"], trade["tagContent"]); tag != "" {
			item["tagContent"] = tag
		} else if infos, ok := trade["sellerFlagInfos"].([]any); ok {
			for _, info := range infos {
				m, _ := info.(map[string]any)
				if m == nil {
					continue
				}
				if asString(m["flagId"]) == star {
					if tc := asString(m["tagContent"]); tc != "" {
						item["tagContent"] = tc
					}
					break
				}
			}
		}
	}
	return item
}

func cloneStringMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
