package repo

import (
	"errors"

	"storesyncagent/internal/model"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type KdzsRepo struct {
	db *gorm.DB
}

func NewKdzs(db *gorm.DB) *KdzsRepo {
	return &KdzsRepo{db: db}
}

func (r *KdzsRepo) ListAccounts(tenantID uint64) ([]model.KdzsAccount, error) {
	var items []model.KdzsAccount
	err := r.db.Where("tenant_id = ? AND enabled = ?", tenantID, true).
		Order("sort_order ASC, id ASC").
		Find(&items).Error
	return items, err
}

func (r *KdzsRepo) ListAllAccounts(tenantID uint64) ([]model.KdzsAccount, error) {
	var items []model.KdzsAccount
	err := r.db.Where("tenant_id = ?", tenantID).
		Order("sort_order ASC, id ASC").
		Find(&items).Error
	return items, err
}

func (r *KdzsRepo) GetAccount(tenantID uint64, code string) (*model.KdzsAccount, error) {
	var item model.KdzsAccount
	err := r.db.Where("tenant_id = ? AND code = ?", tenantID, code).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *KdzsRepo) CreateAccount(item *model.KdzsAccount) error {
	return r.db.Create(item).Error
}

func (r *KdzsRepo) SaveAccount(item *model.KdzsAccount) error {
	return r.db.Save(item).Error
}

func (r *KdzsRepo) DeleteAccount(tenantID uint64, code string) error {
	res := r.db.Where("tenant_id = ? AND code = ?", tenantID, code).Delete(&model.KdzsAccount{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *KdzsRepo) CountAccounts(tenantID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.KdzsAccount{}).Where("tenant_id = ?", tenantID).Count(&count).Error
	return count, err
}

func (r *KdzsRepo) GetOrCreateSettings(tenantID uint64, defaultBaseURL string) (*model.TenantKdzsSetting, error) {
	var item model.TenantKdzsSetting
	err := r.db.Where("tenant_id = ?", tenantID).First(&item).Error
	if err == nil {
		return &item, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if defaultBaseURL == "" {
		defaultBaseURL = "https://df.kdzs.com"
	}
	item = model.TenantKdzsSetting{
		TenantID: tenantID,
		BaseURL:  defaultBaseURL,
	}
	if err := r.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *KdzsRepo) SaveSettings(item *model.TenantKdzsSetting) error {
	return r.db.Save(item).Error
}

func (r *KdzsRepo) UpdateActiveAccount(tenantID uint64, code string) error {
	return r.db.Model(&model.TenantKdzsSetting{}).
		Where("tenant_id = ?", tenantID).
		Update("active_account_code", code).Error
}

func (r *KdzsRepo) ListTenantIDs() ([]uint64, error) {
	var fromAccounts []uint64
	if err := r.db.Model(&model.KdzsAccount{}).Distinct("tenant_id").Pluck("tenant_id", &fromAccounts).Error; err != nil {
		return nil, err
	}
	var fromSettings []uint64
	if err := r.db.Model(&model.TenantKdzsSetting{}).Distinct("tenant_id").Pluck("tenant_id", &fromSettings).Error; err != nil {
		return nil, err
	}
	seen := make(map[uint64]struct{})
	var ids []uint64
	for _, id := range append(fromAccounts, fromSettings...) {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}
