package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Kdzs     KdzsConfig   `mapstructure:"kdzs"`
	Auth     AuthConfig   `mapstructure:"auth"`
	CORS     CORSConfig   `mapstructure:"cors"`
}

type DatabaseConfig struct {
	Driver      string `mapstructure:"driver"`
	PostgresDSN string `mapstructure:"postgres_dsn"`
}

type AuthConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	JWTSecret string `mapstructure:"jwt_secret"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type KdzsConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

// KdzsAccount 映射 kdzs_accounts 表记录，供业务层传递账号信息。
type KdzsAccount struct {
	ID       string
	Name     string
	Role     string
	Mobile   string
	Password string
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8097
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "release"
	}
	if cfg.Kdzs.BaseURL == "" {
		cfg.Kdzs.BaseURL = "https://df.kdzs.com"
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "postgres"
	}
	if cfg.Database.Driver != "postgres" {
		return nil, fmt.Errorf("unsupported database driver: %s (only postgres)", cfg.Database.Driver)
	}
	if cfg.Database.PostgresDSN == "" {
		return nil, fmt.Errorf("database.postgres_dsn is required")
	}
	if !cfg.Auth.Enabled {
		return nil, fmt.Errorf("auth.enabled must be true")
	}
	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("auth.jwt_secret is required")
	}
	if len(cfg.CORS.AllowOrigins) == 0 {
		return nil, fmt.Errorf("cors.allow_origins is required")
	}
	return &cfg, nil
}
