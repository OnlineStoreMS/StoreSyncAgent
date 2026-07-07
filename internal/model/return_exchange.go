package model

import "time"

type ReturnExchange struct {
	ID                    uint64    `gorm:"primaryKey"`
	TenantID              uint64    `gorm:"not null;uniqueIndex:idx_rex_tenant_record,priority:1"`
	RecordID              string    `gorm:"size:64;not null;uniqueIndex:idx_rex_tenant_record,priority:2"`
	SeqNo                 int       `gorm:"not null;default:0"`
	BuyerNick             string    `gorm:"size:128"`
	AfterSaleType         string    `gorm:"size:64"`
	ReturnTrackingNo      string    `gorm:"size:64"`
	Spec                  string    `gorm:"size:256"`
	FeedbackTime          string    `gorm:"size:32"`
	SubmitTime            string    `gorm:"size:32"`
	OrderNo               string    `gorm:"size:64"`
	RecipientInfo         string    `gorm:"type:text"`
	ParsedRecipientInfo   string    `gorm:"type:text"`
	OutboundTrackingNo    string    `gorm:"size:64"`
	Remark                string    `gorm:"type:text"`
	Platform              string    `gorm:"size:16"`
	SysTid                string    `gorm:"size:64"`
	ShopName              string    `gorm:"size:128"`
	GoodsJSON             string    `gorm:"type:text"`
	GoodsTitle            string    `gorm:"size:512"`
	OriginalRecipientInfo string    `gorm:"type:text"`
	Payment               float64   `gorm:"default:0"`
	PayTime               string    `gorm:"size:32"`
	StatusText            string    `gorm:"size:128"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
