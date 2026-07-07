package database

import (
	"fmt"

	"storesyncagent/internal/config"
	"storesyncagent/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	if cfg.Driver != "postgres" {
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
	if cfg.PostgresDSN == "" {
		return nil, fmt.Errorf("postgres_dsn is required")
	}
	return gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.KdzsAccount{},
		&model.TenantKdzsSetting{},
		&model.ReturnExchange{},
		&model.TenantNotification{},
	)
}
