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

type SyncedItemListQuery struct {
	Platform       string
	ShopIDs        []string
	Type           string
	Title          string
	ShortTitle     string
	ItemIDs        []string
	OuterID        string
	ProductNumLike string
	SkuTexts       []string
	PageNo         int
	PageSize       int
}

func (r *SyncedItemRepo) List(tenantID uint64, accountCode string, q SyncedItemListQuery) ([]model.SyncedShopItem, int64, error) {
	pageNo := q.PageNo
	if pageNo < 1 {
		pageNo = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	dbq := r.db.Model(&model.SyncedShopItem{}).
		Where("tenant_id = ? AND account_code = ?", tenantID, accountCode)
	if platform := strings.ToUpper(strings.TrimSpace(q.Platform)); platform != "" {
		dbq = dbq.Where("platform = ?", platform)
	}
	if len(q.ShopIDs) > 0 {
		dbq = dbq.Where("shop_id IN ?", q.ShopIDs)
	}
	if len(q.ItemIDs) > 0 {
		dbq = dbq.Where("item_id IN ?", q.ItemIDs)
	}
	dbq = applyApproveType(dbq, q.Type)
	dbq = applyLike(dbq, "title", q.Title)
	dbq = applyLike(dbq, "short_title", q.ShortTitle)
	dbq = applyLike(dbq, "outer_id", q.OuterID)
	dbq = applyLike(dbq, "product_num", q.ProductNumLike)
	for _, text := range q.SkuTexts {
		dbq = applyLike(dbq, "sku_search_text", text)
	}

	var total int64
	if err := dbq.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.SyncedShopItem
	err := dbq.Session(&gorm.Session{}).
		Order("shop_id, item_id").
		Offset((pageNo - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error
	return rows, total, err
}

func applyLike(dbq *gorm.DB, column, value string) *gorm.DB {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return dbq
	}
	return dbq.Where("LOWER("+column+") LIKE ? ESCAPE '\\'", "%"+escapeLike(value)+"%")
}

func applyApproveType(dbq *gorm.DB, typ string) *gorm.DB {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "onsale":
		return dbq.Where("LOWER(REPLACE(approve_status, '-', '_')) IN ?", []string{"onsale", "on_sale"})
	case "instock":
		return dbq.Where("LOWER(REPLACE(approve_status, '-', '_')) IN ?", []string{"instock", "in_stock"})
	default:
		return dbq
	}
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
