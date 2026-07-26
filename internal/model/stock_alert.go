package model

import "time"

type TenantStockAlert struct {
	ID                  uint64     `gorm:"primaryKey"`
	TenantID            uint64     `gorm:"uniqueIndex;not null"`
	Enabled             bool       `gorm:"not null;default:false"`
	WebhookURL          string     `gorm:"type:text"`
	Secret              string     `gorm:"size:256"`
	Platform            string     `gorm:"size:16;default:FXG"`
	ShopIDsJSON         string     `gorm:"type:text"`
	StockThreshold      int        `gorm:"not null;default:10"`
	CheckLevel          string     `gorm:"size:16;default:sku"` // sku | spu | both
	OnlyOnsale          bool       `gorm:"not null;default:true"`
	PollIntervalMinutes int        `gorm:"not null;default:60"`
	LastRunAt           *time.Time
	LastRunOK           bool       `gorm:"not null;default:false"`
	LastError           string     `gorm:"type:text"`
	LastSentCount       int        `gorm:"not null;default:0"`
	LastAlertCount      int        `gorm:"not null;default:0"`
	NotifiedJSON        string     `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
