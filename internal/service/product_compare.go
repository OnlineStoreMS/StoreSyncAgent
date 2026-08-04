package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"storesyncagent/internal/kdzs"
)

type ProductCompareReq struct {
	CoreProductID uint64   `json:"coreProductId"`
	StoreItem     ItemView `json:"storeItem"`
}

type ProductCoreSearchItem struct {
	ID       uint64  `json:"id"`
	Name     string  `json:"name"`
	Pic      string  `json:"pic"`
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
	SkuCount int     `json:"skuCount"`
}

type ProductCoreSearchResult struct {
	List     []ProductCoreSearchItem `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}

type ProductCompareSummary struct {
	StoreTitle      string `json:"storeTitle"`
	StoreItemID     string `json:"storeItemId"`
	CoreProductID   uint64 `json:"coreProductId"`
	CoreProductName string `json:"coreProductName"`
	StoreSkuCount   int    `json:"storeSkuCount"`
	CoreSkuCount    int    `json:"coreSkuCount"`
	MatchedCount    int    `json:"matchedCount"`
	SpecDiffCount   int    `json:"specDiffCount"`
	PriceDiffCount  int    `json:"priceDiffCount"`
	StockDiffCount  int    `json:"stockDiffCount"`
}

// SpecDiffRow 按规格值比对后的差异。kind: store_only | core_only
type SpecDiffRow struct {
	Kind      string  `json:"kind"`
	SpecValue string  `json:"specValue"`
	StoreSpec string  `json:"storeSpec,omitempty"`
	CoreSpec  string  `json:"coreSpec,omitempty"`
	StoreCode string  `json:"storeCode,omitempty"`
	CoreCode  string  `json:"coreCode,omitempty"`
	// 中心有 / 店铺无：待新增 SKU 的图片、价格、库存（取自 ProductCore SKU）
	Image string  `json:"image,omitempty"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type PriceStockDiffRow struct {
	SpecValue  string  `json:"specValue"`
	StoreSpec  string  `json:"storeSpec"`
	CoreSpec   string  `json:"coreSpec"`
	StoreCode  string  `json:"storeCode,omitempty"`
	CoreCode   string  `json:"coreCode,omitempty"`
	StorePrice float64 `json:"storePrice"`
	CorePrice  float64 `json:"corePrice"`
	PriceDiff  bool    `json:"priceDiff"`
	StoreStock int     `json:"storeStock"`
	CoreStock  int     `json:"coreStock"`
	StockDiff  bool    `json:"stockDiff"`
}

type ProductCompareResult struct {
	Summary         ProductCompareSummary `json:"summary"`
	SpecDiffs       []SpecDiffRow         `json:"specDiffs"`
	PriceStockDiffs []PriceStockDiffRow   `json:"priceStockDiffs"`
}

type storeSkuRef struct {
	Key        string // 规格值匹配键
	SpecValue  string // 展示用规格值
	SpecLabel  string // 原始规格文案
	OuterCode  string
	Price      float64
	Stock      int
}

type coreSkuRef struct {
	Key       string
	SpecValue string
	SpecLabel string
	SkuCode   string
	Price     float64
	Stock     int
	Pic       string
}

func (s *SyncService) SearchCoreProducts(ctx context.Context, bearerToken, keyword string, page, pageSize int) (*ProductCoreSearchResult, error) {
	if s.productCore == nil || !s.productCore.Enabled() {
		return nil, fmt.Errorf("未配置 ProductCore 地址（integrations.productcore_api_url）")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	list, total, err := s.productCore.ListProducts(ctx, bearerToken, keyword, page, pageSize)
	if err != nil {
		return nil, err
	}
	out := make([]ProductCoreSearchItem, 0, len(list))
	for _, p := range list {
		out = append(out, ProductCoreSearchItem{
			ID:       p.ID,
			Name:     p.Name,
			Pic:      p.Pic,
			Price:    p.Price,
			Stock:    p.Stock,
			SkuCount: p.SkuCount,
		})
	}
	return &ProductCoreSearchResult{List: out, Total: total, Page: page, PageSize: pageSize}, nil
}

// CompareStoreItemWithCore 按 SKU「规格值」比对店铺商品与 ProductCore 上品。
// 例如店铺 propertiesName「商品规格:HG400-9速飞轮 11-28T」只取规格值「HG400-9速飞轮 11-28T」参与匹配。
func (s *SyncService) CompareStoreItemWithCore(ctx context.Context, bearerToken string, req ProductCompareReq) (*ProductCompareResult, error) {
	if s.productCore == nil || !s.productCore.Enabled() {
		return nil, fmt.Errorf("未配置 ProductCore 地址（integrations.productcore_api_url）")
	}
	if req.CoreProductID == 0 {
		return nil, fmt.Errorf("请选择 ProductCore 商品")
	}
	if strings.TrimSpace(req.StoreItem.ItemID) == "" && strings.TrimSpace(req.StoreItem.Title) == "" {
		return nil, fmt.Errorf("店铺商品无效")
	}

	core, err := s.productCore.GetProductSkus(ctx, bearerToken, req.CoreProductID)
	if err != nil {
		return nil, fmt.Errorf("拉取 ProductCore 商品失败: %w", err)
	}

	item := req.StoreItem
	storeByKey := map[string]storeSkuRef{}
	storeSkuCount := 0
	if len(item.Skus) == 0 {
		// 无 SKU 明细时无法按规格值比对
	} else {
		for _, sku := range item.Skus {
			if kdzs.IsPlatformDeletedStatus(sku.Status) || sku.State == 1 {
				continue
			}
			values := extractSpecValues(sku.PropertiesName)
			key := specValuesKey(values)
			if key == "" {
				continue
			}
			storeSkuCount++
			storeByKey[key] = storeSkuRef{
				Key:       key,
				SpecValue: strings.Join(values, " / "),
				SpecLabel: strings.Join(values, " / "),
				OuterCode: strings.TrimSpace(firstNonEmptyCode(sku.OuterID, sku.ProductNum)),
				Price:     parsePrice(sku.Price),
				Stock:     sku.Quantity,
			}
		}
	}

	coreByKey := map[string]coreSkuRef{}
	coreSkuCount := 0
	specValuePic := map[string]string{}
	for _, spec := range core.SkuSpecs {
		for _, v := range spec.Values {
			val := strings.TrimSpace(v.Value)
			pic := strings.TrimSpace(v.Pic)
			if val != "" && pic != "" {
				specValuePic[val] = pic
			}
		}
	}
	for _, sku := range core.Skus {
		values := extractSpecValuesFromMap(sku.Specs)
		key := specValuesKey(values)
		if key == "" {
			continue
		}
		coreSkuCount++
		pic := strings.TrimSpace(sku.Pic)
		if pic == "" {
			for _, v := range values {
				if p := specValuePic[v]; p != "" {
					pic = p
					break
				}
			}
		}
		if pic == "" {
			pic = strings.TrimSpace(core.Pic)
		}
		coreByKey[key] = coreSkuRef{
			Key:       key,
			SpecValue: strings.Join(values, " / "),
			SpecLabel: strings.Join(values, " / "),
			SkuCode:   strings.TrimSpace(sku.SkuCode),
			Price:     sku.Price,
			Stock:     sku.Stock,
			Pic:       pic,
		}
	}

	var specDiffs []SpecDiffRow
	var priceStockDiffs []PriceStockDiffRow
	matched := 0

	for key, st := range storeByKey {
		if _, ok := coreByKey[key]; ok {
			matched++
			continue
		}
		specDiffs = append(specDiffs, SpecDiffRow{
			Kind:      "store_only",
			SpecValue: st.SpecValue,
			StoreSpec: st.SpecLabel,
			StoreCode: st.OuterCode,
		})
	}
	for key, c := range coreByKey {
		if _, ok := storeByKey[key]; ok {
			continue
		}
		specDiffs = append(specDiffs, SpecDiffRow{
			Kind:      "core_only",
			SpecValue: c.SpecValue,
			CoreSpec:  c.SpecLabel,
			CoreCode:  c.SkuCode,
			Image:     c.Pic,
			Price:     c.Price,
			Stock:     c.Stock,
		})
	}

	priceDiffCount, stockDiffCount := 0, 0
	for key, st := range storeByKey {
		c, ok := coreByKey[key]
		if !ok {
			continue
		}
		priceDiff := !almostEqualPrice(st.Price, c.Price)
		stockDiff := st.Stock != c.Stock
		if !priceDiff && !stockDiff {
			continue
		}
		if priceDiff {
			priceDiffCount++
		}
		if stockDiff {
			stockDiffCount++
		}
		priceStockDiffs = append(priceStockDiffs, PriceStockDiffRow{
			SpecValue:  st.SpecValue,
			StoreSpec:  st.SpecLabel,
			CoreSpec:   c.SpecLabel,
			StoreCode:  st.OuterCode,
			CoreCode:   c.SkuCode,
			StorePrice: st.Price,
			CorePrice:  c.Price,
			PriceDiff:  priceDiff,
			StoreStock: st.Stock,
			CoreStock:  c.Stock,
			StockDiff:  stockDiff,
		})
	}

	sort.Slice(specDiffs, func(i, j int) bool {
		if specDiffs[i].Kind != specDiffs[j].Kind {
			return specKindOrder(specDiffs[i].Kind) < specKindOrder(specDiffs[j].Kind)
		}
		return specDiffs[i].SpecValue < specDiffs[j].SpecValue
	})
	sort.Slice(priceStockDiffs, func(i, j int) bool {
		return priceStockDiffs[i].SpecValue < priceStockDiffs[j].SpecValue
	})

	return &ProductCompareResult{
		Summary: ProductCompareSummary{
			StoreTitle:      item.Title,
			StoreItemID:     item.ItemID,
			CoreProductID:   core.ID,
			CoreProductName: core.Name,
			StoreSkuCount:   storeSkuCount,
			CoreSkuCount:    coreSkuCount,
			MatchedCount:    matched,
			SpecDiffCount:   len(specDiffs),
			PriceDiffCount:  priceDiffCount,
			StockDiffCount:  stockDiffCount,
		},
		SpecDiffs:       specDiffs,
		PriceStockDiffs: priceStockDiffs,
	}, nil
}

// extractSpecValues 从店铺 propertiesName 提取规格值。
// 「商品规格:HG400-9速飞轮 11-28T」→ ["HG400-9速飞轮 11-28T"]
// 「颜色:红;尺码:L」→ ["红","L"]（排序前）
func extractSpecValues(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, "；", ";")
	raw = strings.ReplaceAll(raw, "，", ";")
	raw = strings.ReplaceAll(raw, ",", ";")
	raw = strings.ReplaceAll(raw, "｜", ";")
	raw = strings.ReplaceAll(raw, "|", ";")
	raw = strings.ReplaceAll(raw, "/", ";")
	parts := strings.Split(raw, ";")
	var values []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = strings.ReplaceAll(p, "：", ":")
		if i := strings.Index(p, ":"); i >= 0 {
			v := strings.TrimSpace(p[i+1:])
			if v != "" {
				values = append(values, v)
			}
			continue
		}
		values = append(values, p)
	}
	return values
}

func extractSpecValuesFromMap(specs map[string]string) []string {
	if len(specs) == 0 {
		return nil
	}
	var values []string
	for _, v := range specs {
		v = strings.TrimSpace(v)
		if v != "" {
			values = append(values, v)
		}
	}
	return values
}

func specValuesKey(values []string) string {
	if len(values) == 0 {
		return ""
	}
	norm := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		v = strings.ReplaceAll(v, " ", "")
		if v == "" {
			continue
		}
		norm = append(norm, v)
	}
	if len(norm) == 0 {
		return ""
	}
	sort.Strings(norm)
	return strings.Join(norm, "|")
}

func firstNonEmptyCode(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func almostEqualPrice(a, b float64) bool {
	return math.Abs(a-b) < 0.005
}

func specKindOrder(kind string) int {
	switch kind {
	case "store_only":
		return 1
	case "core_only":
		return 2
	default:
		return 9
	}
}
