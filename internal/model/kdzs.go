package model

import "time"

type KdzsAccount struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"not null;uniqueIndex:idx_kdzs_acc_tenant_code,priority:1" json:"tenantId"`
	Code      string    `gorm:"size:64;not null;uniqueIndex:idx_kdzs_acc_tenant_code,priority:2" json:"code"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Role      string    `gorm:"size:32;not null;default:merchant" json:"role"`
	Mobile    string    `gorm:"size:32;not null" json:"mobile"`
	Password  string    `gorm:"size:256;not null" json:"-"`
	SortOrder int       `gorm:"not null;default:0" json:"sortOrder"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TenantKdzsSetting struct {
	ID                 uint64    `gorm:"primaryKey" json:"id"`
	TenantID           uint64    `gorm:"uniqueIndex;not null" json:"tenantId"`
	BaseURL            string    `gorm:"size:256;not null" json:"baseUrl"`
	DefaultAccountCode string    `gorm:"size:64" json:"defaultAccountCode"`
	ActiveAccountCode  string    `gorm:"size:64" json:"activeAccountCode"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}
