package model

import "time"

type TenantNotification struct {
	ID                  uint64     `gorm:"primaryKey"`
	TenantID            uint64     `gorm:"uniqueIndex;not null"`
	Enabled             bool       `gorm:"not null;default:false"`
	WebhookURL          string     `gorm:"type:text"`
	Secret              string     `gorm:"size:256"`
	Platform            string     `gorm:"size:16;default:FXG"`
	PollIntervalMinutes int        `gorm:"not null;default:15"`
	DateRangeDays       int        `gorm:"not null;default:30"`
	ScenariosJSON       string     `gorm:"type:text"`
	AccountIDsJSON      string     `gorm:"type:text"`
	AppID               string     `gorm:"size:128"`
	AppSecret           string     `gorm:"size:256"`
	LastRunAt           *time.Time
	LastRunOK           bool       `gorm:"not null;default:false"`
	LastError           string     `gorm:"type:text"`
	LastSentCount       int        `gorm:"not null;default:0"`
	LastBarcodeError    string     `gorm:"type:text"`
	NotifiedJSON        string     `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
