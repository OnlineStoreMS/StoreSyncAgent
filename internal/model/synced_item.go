package model

import "time"

// SyncedShopItem 商品列表同步完成后落入本地的店铺商品，供同商品比对查询。
type SyncedShopItem struct {
	ID          uint64    `gorm:"primaryKey"`
	TenantID    uint64    `gorm:"not null;uniqueIndex:idx_synced_item,priority:1;index:idx_synced_item_shop,priority:1"`
	AccountCode string    `gorm:"size:64;not null;uniqueIndex:idx_synced_item,priority:2;index:idx_synced_item_shop,priority:2"`
	Platform    string    `gorm:"size:16;not null"`
	ShopID      string    `gorm:"size:64;not null;uniqueIndex:idx_synced_item,priority:3;index:idx_synced_item_shop,priority:3"`
	ShopName    string    `gorm:"size:128"`
	ItemID      string    `gorm:"size:64;not null;uniqueIndex:idx_synced_item,priority:4"`
	SearchText  string    `gorm:"type:text"`
	Payload     string    `gorm:"type:text"`
	SyncedAt    time.Time `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
