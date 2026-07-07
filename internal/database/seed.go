package database

import (
	"log"
	"strconv"

	"storesyncagent/internal/config"
	"storesyncagent/internal/model"
	"storesyncagent/internal/repo"

	"gorm.io/gorm"
)

// SeedLegacyAccounts imports kdzs.accounts from config when DB has no rows for that tenant.
func SeedLegacyAccounts(db *gorm.DB, cfg *config.Config) {
	r := repo.NewKdzs(db)
	seedTenant := func(tenantID uint64, kdzsCfg config.KdzsConfig) {
		count, err := r.CountAccounts(tenantID)
		if err != nil || count > 0 {
			return
		}
		accounts := kdzsCfg.ResolveAccounts()
		if len(accounts) == 0 {
			return
		}
		settings, err := r.GetOrCreateSettings(tenantID, kdzsCfg.BaseURL)
		if err != nil {
			log.Printf("[seed] tenant %d settings: %v", tenantID, err)
			return
		}
		if kdzsCfg.DefaultAccountID != "" {
			settings.DefaultAccountCode = kdzsCfg.DefaultAccountID
		} else if len(accounts) > 0 {
			settings.DefaultAccountCode = accounts[0].ID
		}
		settings.ActiveAccountCode = settings.DefaultAccountCode
		if err := r.SaveSettings(settings); err != nil {
			log.Printf("[seed] tenant %d save settings: %v", tenantID, err)
			return
		}
		for i, acc := range accounts {
			rec := &model.KdzsAccount{
				TenantID:  tenantID,
				Code:      acc.ID,
				Name:      acc.Name,
				Role:      acc.Role,
				Mobile:    acc.Mobile,
				Password:  acc.Password,
				SortOrder: i,
				Enabled:   true,
			}
			if rec.Role == "" {
				rec.Role = "merchant"
			}
			if rec.Name == "" {
				rec.Name = acc.Mobile
			}
			if err := r.CreateAccount(rec); err != nil {
				log.Printf("[seed] tenant %d account %s: %v", tenantID, acc.ID, err)
			}
		}
		log.Printf("[seed] imported %d kdzs account(s) for tenant %d from config", len(accounts), tenantID)
	}

	if len(cfg.Kdzs.ResolveAccounts()) > 0 {
		seedTenant(1, cfg.Kdzs)
	}
	for key, tenantCfg := range cfg.Tenants {
		if tenantCfg.Kdzs == nil {
			continue
		}
		id, err := strconv.ParseUint(key, 10, 64)
		if err != nil || id == 0 {
			continue
		}
		kdzs := *tenantCfg.Kdzs
		if kdzs.BaseURL == "" {
			kdzs.BaseURL = cfg.Kdzs.BaseURL
		}
		seedTenant(id, kdzs)
	}
}
