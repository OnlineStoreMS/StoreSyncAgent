package repo

import (
	"strings"
	"time"

	"storesyncagent/internal/model"

	"gorm.io/gorm"
)

type SyncedItemRepo struct {
	db *gorm.DB
}

func NewSyncedItem(db *gorm.DB) *SyncedItemRepo {
	return &SyncedItemRepo{db: db}
}

func (r *SyncedItemRepo) CountShop(tenantID uint64, accountCode, shopID string) (int64, error) {
	var n int64
	err := r.db.Model(&model.SyncedShopItem{}).
		Where("tenant_id = ? AND account_code = ? AND shop_id = ?", tenantID, accountCode, shopID).
		Count(&n).Error
	return n, err
}

func (r *SyncedItemRepo) Search(tenantID uint64, accountCode, shopID string, keywords []string, limit int) ([]model.SyncedShopItem, error) {
	q := r.db.Where("tenant_id = ? AND account_code = ? AND shop_id = ?", tenantID, accountCode, shopID)
	for _, kw := range keywords {
		kw = strings.TrimSpace(strings.ToLower(kw))
		if kw == "" {
			continue
		}
		q = q.Where("search_text LIKE ? ESCAPE '\\'", "%"+escapeLike(kw)+"%")
	}
	if limit <= 0 {
		limit = 500
	}
	var rows []model.SyncedShopItem
	err := q.Order("item_id").Limit(limit).Find(&rows).Error
	return rows, err
}

// ReplaceShop 用本次同步结果覆盖该店本地商品。空列表会清空该店。
func (r *SyncedItemRepo) ReplaceShop(tenantID uint64, accountCode, shopID string, rows []model.SyncedShopItem) error {
	now := time.Now()
	for i := range rows {
		rows[i].TenantID = tenantID
		rows[i].AccountCode = accountCode
		rows[i].ShopID = shopID
		if rows[i].SyncedAt.IsZero() {
			rows[i].SyncedAt = now
		}
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND account_code = ? AND shop_id = ?", tenantID, accountCode, shopID).
			Delete(&model.SyncedShopItem{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.CreateInBatches(rows, 100).Error
	})
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
