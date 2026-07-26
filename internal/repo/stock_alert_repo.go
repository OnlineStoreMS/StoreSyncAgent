package repo

import (
	"encoding/json"
	"errors"
	"time"

	"storesyncagent/internal/model"
	"storesyncagent/internal/store"

	"gorm.io/gorm"
)

type StockAlertRepo struct {
	db *gorm.DB
}

func NewStockAlert(db *gorm.DB) *StockAlertRepo {
	return &StockAlertRepo{db: db}
}

func (r *StockAlertRepo) Load(tenantID uint64) (store.StockAlertData, error) {
	row, err := r.getOrCreate(tenantID)
	if err != nil {
		return store.StockAlertData{}, err
	}
	return rowToStockAlertData(*row), nil
}

func (r *StockAlertRepo) SaveConfig(tenantID uint64, cfg store.StockAlertConfig) (store.StockAlertData, error) {
	row, err := r.getOrCreate(tenantID)
	if err != nil {
		return store.StockAlertData{}, err
	}
	data := rowToStockAlertData(*row)
	if cfg.Secret == "" {
		cfg.Secret = data.Config.Secret
	}
	store.NormalizeStockAlertConfig(&cfg)
	if err := r.applyConfig(row, cfg); err != nil {
		return store.StockAlertData{}, err
	}
	if err := r.db.Save(row).Error; err != nil {
		return store.StockAlertData{}, err
	}
	return rowToStockAlertData(*row), nil
}

func (r *StockAlertRepo) UpdateState(tenantID uint64, fn func(*store.StockAlertState) error) error {
	row, err := r.getOrCreate(tenantID)
	if err != nil {
		return err
	}
	data := rowToStockAlertData(*row)
	if data.State.Notified == nil {
		data.State.Notified = map[string]string{}
	}
	if err := fn(&data.State); err != nil {
		return err
	}
	store.PruneNotified(data.State.Notified, 60)
	return r.applyState(row, data.State)
}

func (r *StockAlertRepo) ResetState(tenantID uint64) (int, error) {
	row, err := r.getOrCreate(tenantID)
	if err != nil {
		return 0, err
	}
	data := rowToStockAlertData(*row)
	cleared := len(data.State.Notified)
	state := store.StockAlertState{Notified: map[string]string{}}
	if err := r.applyState(row, state); err != nil {
		return 0, err
	}
	return cleared, nil
}

func (r *StockAlertRepo) ListTenantIDs() ([]uint64, error) {
	var ids []uint64
	err := r.db.Model(&model.TenantStockAlert{}).Distinct("tenant_id").Pluck("tenant_id", &ids).Error
	return ids, err
}

func (r *StockAlertRepo) getOrCreate(tenantID uint64) (*model.TenantStockAlert, error) {
	var row model.TenantStockAlert
	err := r.db.Where("tenant_id = ?", tenantID).First(&row).Error
	if err == nil {
		return &row, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	row = model.TenantStockAlert{
		TenantID:            tenantID,
		Platform:            "FXG",
		StockThreshold:      10,
		CheckLevel:          "sku",
		OnlyOnsale:          true,
		PollIntervalMinutes: 60,
		NotifiedJSON:        "{}",
	}
	if err := r.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *StockAlertRepo) applyConfig(row *model.TenantStockAlert, cfg store.StockAlertConfig) error {
	shopIDsJSON, err := json.Marshal(cfg.ShopIDs)
	if err != nil {
		return err
	}
	row.Enabled = cfg.Enabled
	row.WebhookURL = cfg.WebhookURL
	row.Secret = cfg.Secret
	row.Platform = cfg.Platform
	row.ShopIDsJSON = string(shopIDsJSON)
	row.StockThreshold = cfg.StockThreshold
	row.CheckLevel = cfg.CheckLevel
	row.OnlyOnsale = cfg.OnlyOnsale
	row.PollIntervalMinutes = cfg.PollIntervalMinutes
	return nil
}

func (r *StockAlertRepo) applyState(row *model.TenantStockAlert, state store.StockAlertState) error {
	if state.Notified == nil {
		state.Notified = map[string]string{}
	}
	notifiedJSON, err := json.Marshal(state.Notified)
	if err != nil {
		return err
	}
	row.LastRunOK = state.LastRunOK
	row.LastError = state.LastError
	row.LastSentCount = state.LastSentCount
	row.LastAlertCount = state.LastAlertCount
	row.NotifiedJSON = string(notifiedJSON)
	if state.LastRunAt != "" {
		if t, ok := store.ParseStoreTime(state.LastRunAt); ok {
			row.LastRunAt = &t
		} else {
			row.LastRunAt = nil
		}
	} else {
		row.LastRunAt = nil
	}
	return r.db.Save(row).Error
}

func rowToStockAlertData(row model.TenantStockAlert) store.StockAlertData {
	cfg := store.StockAlertConfig{
		Enabled:             row.Enabled,
		WebhookURL:          row.WebhookURL,
		Secret:              row.Secret,
		Platform:            row.Platform,
		StockThreshold:      row.StockThreshold,
		CheckLevel:          row.CheckLevel,
		OnlyOnsale:          row.OnlyOnsale,
		PollIntervalMinutes: row.PollIntervalMinutes,
	}
	if row.ShopIDsJSON != "" {
		_ = json.Unmarshal([]byte(row.ShopIDsJSON), &cfg.ShopIDs)
	}
	store.NormalizeStockAlertConfig(&cfg)

	state := store.StockAlertState{
		LastRunOK:      row.LastRunOK,
		LastError:      row.LastError,
		LastSentCount:  row.LastSentCount,
		LastAlertCount: row.LastAlertCount,
		Notified:       map[string]string{},
	}
	if row.LastRunAt != nil {
		state.LastRunAt = row.LastRunAt.Format(time.RFC3339)
	}
	if row.NotifiedJSON != "" {
		_ = json.Unmarshal([]byte(row.NotifiedJSON), &state.Notified)
	}
	if state.Notified == nil {
		state.Notified = map[string]string{}
	}
	return store.StockAlertData{Config: cfg, State: state}
}
