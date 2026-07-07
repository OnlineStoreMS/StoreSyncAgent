package repo

import (
	"encoding/json"
	"errors"
	"time"

	"storesyncagent/internal/model"
	"storesyncagent/internal/store"

	"gorm.io/gorm"
)

type NotificationRepo struct {
	db *gorm.DB
}

func NewNotification(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Load(tenantID uint64) (store.NotificationData, error) {
	row, err := r.getOrCreate(tenantID)
	if err != nil {
		return store.NotificationData{}, err
	}
	return rowToNotificationData(*row), nil
}

func (r *NotificationRepo) SaveConfig(tenantID uint64, cfg store.NotificationConfig) (store.NotificationData, error) {
	row, err := r.getOrCreate(tenantID)
	if err != nil {
		return store.NotificationData{}, err
	}
	data := rowToNotificationData(*row)
	if cfg.Secret == "" {
		cfg.Secret = data.Config.Secret
	}
	if cfg.AppSecret == "" {
		cfg.AppSecret = data.Config.AppSecret
	}
	store.NormalizeNotificationConfig(&cfg)
	if err := r.applyConfig(row, cfg); err != nil {
		return store.NotificationData{}, err
	}
	if err := r.db.Save(row).Error; err != nil {
		return store.NotificationData{}, err
	}
	return rowToNotificationData(*row), nil
}

func (r *NotificationRepo) SaveState(tenantID uint64, state store.NotificationState) error {
	row, err := r.getOrCreate(tenantID)
	if err != nil {
		return err
	}
	if state.Notified == nil {
		state.Notified = map[string]string{}
	}
	store.PruneNotified(state.Notified, 60)
	return r.applyState(row, state)
}

func (r *NotificationRepo) UpdateState(tenantID uint64, fn func(*store.NotificationState) error) error {
	row, err := r.getOrCreate(tenantID)
	if err != nil {
		return err
	}
	data := rowToNotificationData(*row)
	if data.State.Notified == nil {
		data.State.Notified = map[string]string{}
	}
	if err := fn(&data.State); err != nil {
		return err
	}
	store.PruneNotified(data.State.Notified, 60)
	return r.applyState(row, data.State)
}

func (r *NotificationRepo) ResetState(tenantID uint64) (int, error) {
	row, err := r.getOrCreate(tenantID)
	if err != nil {
		return 0, err
	}
	data := rowToNotificationData(*row)
	cleared := len(data.State.Notified)
	state := store.NotificationState{Notified: map[string]string{}}
	if err := r.applyState(row, state); err != nil {
		return 0, err
	}
	return cleared, nil
}

func (r *NotificationRepo) HasRow(tenantID uint64) (bool, error) {
	var count int64
	err := r.db.Model(&model.TenantNotification{}).Where("tenant_id = ?", tenantID).Count(&count).Error
	return count > 0, err
}

func (r *NotificationRepo) ListTenantIDs() ([]uint64, error) {
	var ids []uint64
	err := r.db.Model(&model.TenantNotification{}).Distinct("tenant_id").Pluck("tenant_id", &ids).Error
	return ids, err
}

func (r *NotificationRepo) getOrCreate(tenantID uint64) (*model.TenantNotification, error) {
	var row model.TenantNotification
	err := r.db.Where("tenant_id = ?", tenantID).First(&row).Error
	if err == nil {
		return &row, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	row = model.TenantNotification{
		TenantID:            tenantID,
		Platform:            "FXG",
		PollIntervalMinutes: 15,
		DateRangeDays:       30,
		NotifiedJSON:        "{}",
	}
	if err := r.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *NotificationRepo) applyConfig(row *model.TenantNotification, cfg store.NotificationConfig) error {
	scenariosJSON, err := json.Marshal(cfg.Scenarios)
	if err != nil {
		return err
	}
	accountIDsJSON, err := json.Marshal(cfg.AccountIDs)
	if err != nil {
		return err
	}
	row.Enabled = cfg.Enabled
	row.WebhookURL = cfg.WebhookURL
	row.Secret = cfg.Secret
	row.Platform = cfg.Platform
	row.PollIntervalMinutes = cfg.PollIntervalMinutes
	row.DateRangeDays = cfg.DateRangeDays
	row.ScenariosJSON = string(scenariosJSON)
	row.AccountIDsJSON = string(accountIDsJSON)
	row.AppID = cfg.AppID
	row.AppSecret = cfg.AppSecret
	return nil
}

func (r *NotificationRepo) applyState(row *model.TenantNotification, state store.NotificationState) error {
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
	row.LastBarcodeError = state.LastBarcodeError
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

func rowToNotificationData(row model.TenantNotification) store.NotificationData {
	cfg := store.NotificationConfig{
		Enabled:             row.Enabled,
		WebhookURL:          row.WebhookURL,
		Secret:              row.Secret,
		Platform:            row.Platform,
		PollIntervalMinutes: row.PollIntervalMinutes,
		DateRangeDays:       row.DateRangeDays,
		AppID:               row.AppID,
		AppSecret:           row.AppSecret,
	}
	if row.ScenariosJSON != "" {
		_ = json.Unmarshal([]byte(row.ScenariosJSON), &cfg.Scenarios)
	}
	if row.AccountIDsJSON != "" {
		_ = json.Unmarshal([]byte(row.AccountIDsJSON), &cfg.AccountIDs)
	}
	store.NormalizeNotificationConfig(&cfg)

	state := store.NotificationState{
		LastRunOK:        row.LastRunOK,
		LastError:        row.LastError,
		LastSentCount:    row.LastSentCount,
		LastBarcodeError: row.LastBarcodeError,
		Notified:         map[string]string{},
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
	return store.NotificationData{Config: cfg, State: state}
}
