package kdzs

import "context"

// Platform names used by 快递助手.
const (
	PlatformDouyin  = "FXG"    // 抖店
	PlatformTaobao  = "TB"     // 淘宝
	PlatformXHS     = "XHS"    // 小红书
	PlatformManual  = "DFHAND" // 手工单
)

// ListBindShops returns all e-commerce shops bound to the account.
func (c *Client) ListBindShops(ctx context.Context) ([]BindShop, error) {
	var resp APIResponse[[]BindShop]
	if err := c.get(ctx, "/factory/management/getBindShopList", &resp); err != nil {
		return nil, err
	}
	return checkResult(&resp)
}

// ListActiveShops returns shops that are bound and not deleted.
func (c *Client) ListActiveShops(ctx context.Context) ([]BindShop, error) {
	shops, err := c.ListBindShops(ctx)
	if err != nil {
		return nil, err
	}
	active := make([]BindShop, 0, len(shops))
	for _, shop := range shops {
		if shop.IsDelete == 0 && shop.BindStatus == 1 {
			active = append(active, shop)
		}
	}
	return active, nil
}

// ListEcommerceShops returns active shops excluding manual-order pseudo shop.
func (c *Client) ListEcommerceShops(ctx context.Context) ([]BindShop, error) {
	shops, err := c.ListActiveShops(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BindShop, 0, len(shops))
	for _, shop := range shops {
		if IsEcommercePlatform(shop.Platform) {
			out = append(out, shop)
		}
	}
	return out, nil
}
