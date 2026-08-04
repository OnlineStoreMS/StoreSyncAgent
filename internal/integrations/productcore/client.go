package productcore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

type ProductListItem struct {
	ID       uint64  `json:"id"`
	Name     string  `json:"name"`
	Pic      string  `json:"pic"`
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
	SkuCount int     `json:"skuCount"`
}

type SkuItem struct {
	ID      uint64            `json:"id"`
	SkuCode string            `json:"skuCode"`
	Specs   map[string]string `json:"specs"`
	Price   float64           `json:"price"`
	Stock   int               `json:"stock"`
	Pic     string            `json:"pic"`
}

type SkuSpecValue struct {
	Value string `json:"value"`
	Pic   string `json:"pic"`
}

type SkuSpec struct {
	Name   string         `json:"name"`
	Values []SkuSpecValue `json:"values"`
}

type ProductSkus struct {
	ID       uint64    `json:"id"`
	Name     string    `json:"name"`
	Pic      string    `json:"pic"`
	SkuSpecs []SkuSpec `json:"skuSpecs"`
	Skus     []SkuItem `json:"skus"`
}

type pagePayload[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

type apiBody struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) ListProducts(ctx context.Context, bearerToken, keyword string, page, pageSize int) ([]ProductListItem, int64, error) {
	if !c.Enabled() {
		return nil, 0, fmt.Errorf("productcore_api_url 未配置")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))
	q.Set("publishStatus", "1")
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		q.Set("keyword", keyword)
	}
	var pageData pagePayload[ProductListItem]
	if err := c.get(ctx, bearerToken, "/api/v1/admin/products?"+q.Encode(), &pageData); err != nil {
		return nil, 0, err
	}
	if pageData.List == nil {
		pageData.List = []ProductListItem{}
	}
	return pageData.List, pageData.Total, nil
}

func (c *Client) GetProductSkus(ctx context.Context, bearerToken string, productID uint64) (*ProductSkus, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("productcore_api_url 未配置")
	}
	if productID == 0 {
		return nil, fmt.Errorf("product id required")
	}
	var item ProductSkus
	path := fmt.Sprintf("/api/v1/admin/products/%d/skus", productID)
	if err := c.get(ctx, bearerToken, path, &item); err != nil {
		return nil, err
	}
	if item.Skus == nil {
		item.Skus = []SkuItem{}
	}
	return &item, nil
}

// LoadAllPublishedSkus 拉取已上架商品的全部 SKU（带分页上限）。
func (c *Client) LoadAllPublishedSkus(ctx context.Context, bearerToken string, maxProducts int) ([]ProductSkus, error) {
	if maxProducts <= 0 {
		maxProducts = 500
	}
	var out []ProductSkus
	page := 1
	const pageSize = 50
	for {
		list, total, err := c.ListProducts(ctx, bearerToken, "", page, pageSize)
		if err != nil {
			return nil, err
		}
		for _, p := range list {
			detail, err := c.GetProductSkus(ctx, bearerToken, p.ID)
			if err != nil {
				return nil, fmt.Errorf("拉取商品 SKU(%d): %w", p.ID, err)
			}
			if detail.Name == "" {
				detail.Name = p.Name
			}
			if detail.Pic == "" {
				detail.Pic = p.Pic
			}
			out = append(out, *detail)
			if len(out) >= maxProducts {
				return out, nil
			}
		}
		if int64(page*pageSize) >= total || len(list) == 0 {
			break
		}
		page++
		if page > 40 {
			break
		}
	}
	return out, nil
}

func (c *Client) get(ctx context.Context, bearerToken, path string, dest any) error {
	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	token := strings.TrimSpace(bearerToken)
	if token != "" {
		if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = "Bearer " + token
		}
		req.Header.Set("Authorization", token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("productcore request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		msg := string(body)
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return fmt.Errorf("productcore http %d: %s", resp.StatusCode, msg)
	}
	var wrapped apiBody
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return fmt.Errorf("productcore decode: %w", err)
	}
	if wrapped.Code != 200 {
		msg := wrapped.Message
		if msg == "" {
			msg = "productcore error"
		}
		return fmt.Errorf("%s", msg)
	}
	if err := json.Unmarshal(wrapped.Data, dest); err != nil {
		return fmt.Errorf("productcore data decode: %w", err)
	}
	return nil
}
