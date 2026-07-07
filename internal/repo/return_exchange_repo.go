package repo

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"storesyncagent/internal/model"
	"storesyncagent/internal/store"

	"gorm.io/gorm"
)

type ReturnExchangeRepo struct {
	db *gorm.DB
}

func NewReturnExchange(db *gorm.DB) *ReturnExchangeRepo {
	return &ReturnExchangeRepo{db: db}
}

func (r *ReturnExchangeRepo) List(tenantID uint64) ([]store.ReturnExchangeRecord, error) {
	var rows []model.ReturnExchange
	err := r.db.Where("tenant_id = ?", tenantID).
		Order("seq_no ASC, created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]store.ReturnExchangeRecord, 0, len(rows))
	for _, row := range rows {
		items = append(items, rowToRecord(row))
	}
	return items, nil
}

func (r *ReturnExchangeRepo) Create(tenantID uint64, in store.ReturnExchangeRecord) (store.ReturnExchangeRecord, error) {
	items, err := r.List(tenantID)
	if err != nil {
		return store.ReturnExchangeRecord{}, err
	}
	now := time.Now()
	rec := in
	if rec.ID == "" {
		rec.ID = newReturnExchangeID()
	}
	if rec.SeqNo == 0 {
		rec.SeqNo = nextReturnExchangeSeqNo(items)
	}
	rec.CreatedAt = now.Format("2006-01-02 15:04:05")
	rec.UpdatedAt = rec.CreatedAt
	row, err := recordToRow(tenantID, rec)
	if err != nil {
		return store.ReturnExchangeRecord{}, err
	}
	if err := r.db.Create(&row).Error; err != nil {
		return store.ReturnExchangeRecord{}, err
	}
	return rowToRecord(row), nil
}

func (r *ReturnExchangeRepo) Update(tenantID uint64, id string, in store.ReturnExchangeRecord) (store.ReturnExchangeRecord, error) {
	var row model.ReturnExchange
	err := r.db.Where("tenant_id = ? AND record_id = ?", tenantID, id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return store.ReturnExchangeRecord{}, fmt.Errorf("record %s not found", id)
	}
	if err != nil {
		return store.ReturnExchangeRecord{}, err
	}
	updated := in
	updated.ID = id
	if updated.SeqNo == 0 {
		updated.SeqNo = row.SeqNo
	}
	updated.CreatedAt = row.CreatedAt.Format("2006-01-02 15:04:05")
	updated.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	next, err := recordToRow(tenantID, updated)
	if err != nil {
		return store.ReturnExchangeRecord{}, err
	}
	next.ID = row.ID
	if err := r.db.Save(&next).Error; err != nil {
		return store.ReturnExchangeRecord{}, err
	}
	return rowToRecord(next), nil
}

func (r *ReturnExchangeRepo) Delete(tenantID uint64, id string) error {
	res := r.db.Where("tenant_id = ? AND record_id = ?", tenantID, id).Delete(&model.ReturnExchange{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("record %s not found", id)
	}
	return nil
}

func (r *ReturnExchangeRepo) Count(tenantID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.ReturnExchange{}).Where("tenant_id = ?", tenantID).Count(&count).Error
	return count, err
}

func (r *ReturnExchangeRepo) ListTenantIDs() ([]uint64, error) {
	var ids []uint64
	err := r.db.Model(&model.ReturnExchange{}).Distinct("tenant_id").Pluck("tenant_id", &ids).Error
	return ids, err
}

func recordToRow(tenantID uint64, rec store.ReturnExchangeRecord) (model.ReturnExchange, error) {
	goodsJSON, err := json.Marshal(rec.Goods)
	if err != nil {
		return model.ReturnExchange{}, err
	}
	row := model.ReturnExchange{
		TenantID:              tenantID,
		RecordID:              rec.ID,
		SeqNo:                 rec.SeqNo,
		BuyerNick:             rec.BuyerNick,
		AfterSaleType:         rec.AfterSaleType,
		ReturnTrackingNo:      rec.ReturnTrackingNo,
		Spec:                  rec.Spec,
		FeedbackTime:          rec.FeedbackTime,
		SubmitTime:            rec.SubmitTime,
		OrderNo:               rec.OrderNo,
		RecipientInfo:         rec.RecipientInfo,
		ParsedRecipientInfo:   rec.ParsedRecipientInfo,
		OutboundTrackingNo:    rec.OutboundTrackingNo,
		Remark:                rec.Remark,
		Platform:              rec.Platform,
		SysTid:                rec.SysTid,
		ShopName:              rec.ShopName,
		GoodsJSON:             string(goodsJSON),
		GoodsTitle:            rec.GoodsTitle,
		OriginalRecipientInfo: rec.OriginalRecipientInfo,
		Payment:               rec.Payment,
		PayTime:               rec.PayTime,
		StatusText:            rec.StatusText,
	}
	if rec.CreatedAt != "" {
		if t, ok := store.ParseStoreTime(rec.CreatedAt); ok {
			row.CreatedAt = t
		}
	}
	if rec.UpdatedAt != "" {
		if t, ok := store.ParseStoreTime(rec.UpdatedAt); ok {
			row.UpdatedAt = t
		}
	}
	return row, nil
}

func rowToRecord(row model.ReturnExchange) store.ReturnExchangeRecord {
	rec := store.ReturnExchangeRecord{
		ID:                    row.RecordID,
		SeqNo:                 row.SeqNo,
		BuyerNick:             row.BuyerNick,
		AfterSaleType:         row.AfterSaleType,
		ReturnTrackingNo:      row.ReturnTrackingNo,
		Spec:                  row.Spec,
		FeedbackTime:          row.FeedbackTime,
		SubmitTime:            row.SubmitTime,
		OrderNo:               row.OrderNo,
		RecipientInfo:         row.RecipientInfo,
		ParsedRecipientInfo:   row.ParsedRecipientInfo,
		OutboundTrackingNo:    row.OutboundTrackingNo,
		Remark:                row.Remark,
		Platform:              row.Platform,
		SysTid:                row.SysTid,
		ShopName:              row.ShopName,
		GoodsTitle:            row.GoodsTitle,
		OriginalRecipientInfo: row.OriginalRecipientInfo,
		Payment:               row.Payment,
		PayTime:               row.PayTime,
		StatusText:            row.StatusText,
		CreatedAt:             row.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:             row.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if row.GoodsJSON != "" {
		_ = json.Unmarshal([]byte(row.GoodsJSON), &rec.Goods)
	}
	return rec
}

func nextReturnExchangeSeqNo(items []store.ReturnExchangeRecord) int {
	max := 0
	for _, item := range items {
		if item.SeqNo > max {
			max = item.SeqNo
		}
	}
	return max + 1
}

func newReturnExchangeID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
