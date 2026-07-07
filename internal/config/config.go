package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig            `mapstructure:"server"`
	Database DatabaseConfig          `mapstructure:"database"`
	Kdzs     KdzsConfig              `mapstructure:"kdzs"`
	Storage  StorageConfig           `mapstructure:"storage"`
	Auth     AuthConfig              `mapstructure:"auth"`
	CORS     CORSConfig              `mapstructure:"cors"`
	Tenants  map[string]TenantConfig `mapstructure:"tenants"`
}

type DatabaseConfig struct {
	Driver         string `mapstructure:"driver"`
	SQLitePath     string `mapstructure:"sqlite_path"`
	PostgresDSN    string `mapstructure:"postgres_dsn"`
	SeedFromConfig bool   `mapstructure:"seed_from_config"`
}

type AuthConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	JWTSecret string `mapstructure:"jwt_secret"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

type TenantConfig struct {
	Kdzs *KdzsConfig `mapstructure:"kdzs"`
}

type StorageConfig struct {
	DataDir string `mapstructure:"data_dir"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type KdzsConfig struct {
	BaseURL         string         `mapstructure:"base_url"`
	Mobile          string         `mapstructure:"mobile"`
	Password        string         `mapstructure:"password"`
	DefaultAccountID string        `mapstructure:"default_account_id"`
	Accounts        []KdzsAccount  `mapstructure:"accounts"`
}

type KdzsAccount struct {
	ID       string `mapstructure:"id"`
	Name     string `mapstructure:"name"`
	Role     string `mapstructure:"role"` // merchant | factory
	Mobile   string `mapstructure:"mobile"`
	Password string `mapstructure:"password"`
}

func (c *KdzsConfig) ResolveAccounts() []KdzsAccount {
	if len(c.Accounts) > 0 {
		return c.Accounts
	}
	if c.Mobile != "" {
		return []KdzsAccount{{
			ID:       "merchant",
			Name:     "商家版",
			Role:     "merchant",
			Mobile:   c.Mobile,
			Password: c.Password,
		}}
	}
	return nil
}

func (c *KdzsConfig) ActiveAccountID() string {
	if c.DefaultAccountID != "" {
		return c.DefaultAccountID
	}
	accounts := c.ResolveAccounts()
	if len(accounts) > 0 {
		return accounts[0].ID
	}
	return "merchant"
}

func (c *KdzsConfig) AccountByID(id string) (KdzsAccount, bool) {
	for _, acc := range c.ResolveAccounts() {
		if acc.ID == id {
			return acc, true
		}
	}
	return KdzsAccount{}, false
}

func (c *KdzsConfig) ActiveAccount() (KdzsAccount, error) {
	acc, ok := c.AccountByID(c.ActiveAccountID())
	if !ok {
		return KdzsAccount{}, fmt.Errorf("account %s not found", c.ActiveAccountID())
	}
	if acc.Password == "" {
		acc.Password = c.Password
	}
	if acc.Mobile == "" {
		return KdzsAccount{}, fmt.Errorf("account %s mobile is empty", acc.ID)
	}
	if acc.Password == "" {
		return KdzsAccount{}, fmt.Errorf("account %s password is empty", acc.ID)
	}
	return acc, nil
}

func (c *Config) TenantDataDir(tenantID uint64) string {
	base := c.Storage.DataDir
	if base == "" {
		base = "data"
	}
	return filepath.Join(base, "tenants", strconv.FormatUint(tenantID, 10))
}

func (c *Config) KdzsForTenant(tenantID uint64) KdzsConfig {
	key := strconv.FormatUint(tenantID, 10)
	if t, ok := c.Tenants[key]; ok && t.Kdzs != nil && len(t.Kdzs.ResolveAccounts()) > 0 {
		kdzs := *t.Kdzs
		if kdzs.BaseURL == "" {
			kdzs.BaseURL = c.Kdzs.BaseURL
		}
		applyEnvSecretsToKdzs(&kdzs, c.Kdzs.Password)
		return kdzs
	}
	return c.Kdzs
}

func (c *Config) ConfiguredTenantIDs() []uint64 {
	seen := make(map[uint64]struct{})
	var ids []uint64
	for key := range c.Tenants {
		id, err := strconv.ParseUint(key, 10, 64)
		if err != nil || id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 && len(c.Kdzs.ResolveAccounts()) > 0 {
		ids = append(ids, 1)
	}
	return ids
}

func (c *KdzsConfig) Validate() error {
	accounts := c.ResolveAccounts()
	if len(accounts) == 0 {
		return fmt.Errorf("kdzs accounts required (configure kdzs.accounts or kdzs.mobile/password)")
	}
	for _, acc := range accounts {
		pwd := acc.Password
		if pwd == "" {
			pwd = c.Password
		}
		if acc.Mobile == "" {
			return fmt.Errorf("account %s mobile is empty", acc.ID)
		}
		if pwd == "" {
			return fmt.Errorf("account %s password is empty", acc.ID)
		}
	}
	if _, err := c.ActiveAccount(); err != nil {
		return err
	}
	return nil
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
		cfg.Server.Mode = "debug"
	}
	if cfg.Kdzs.BaseURL == "" {
		cfg.Kdzs.BaseURL = "https://df.kdzs.com"
	}
	if cfg.Storage.DataDir == "" {
		cfg.Storage.DataDir = "data"
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.SQLitePath == "" {
		cfg.Database.SQLitePath = filepath.Join(cfg.Storage.DataDir, "storesyncagent.db")
	}
	if len(cfg.CORS.AllowOrigins) == 0 {
		cfg.CORS.AllowOrigins = []string{
			"http://localhost:5178",
			"http://127.0.0.1:5178",
			"http://localhost:5174",
			"http://127.0.0.1:5174",
		}
	}
	applyEnvSecrets(&cfg)
	return &cfg, nil
}

func applyEnvSecrets(cfg *Config) {
	applyEnvSecretsToKdzs(&cfg.Kdzs, "")
	for key, tenant := range cfg.Tenants {
		if tenant.Kdzs == nil {
			continue
		}
		applyEnvSecretsToKdzs(tenant.Kdzs, cfg.Kdzs.Password)
		cfg.Tenants[key] = tenant
	}
}

func applyEnvSecretsToKdzs(kdzs *KdzsConfig, fallbackPassword string) {
	if pwd := os.Getenv("KDZS_PASSWORD"); pwd != "" {
		kdzs.Password = pwd
	} else if kdzs.Password == "" && fallbackPassword != "" {
		kdzs.Password = fallbackPassword
	}
	if mobile := os.Getenv("KDZS_MOBILE"); mobile != "" {
		kdzs.Mobile = mobile
	}
	for i := range kdzs.Accounts {
		acc := &kdzs.Accounts[i]
		if acc.Password != "" {
			continue
		}
		if pwd := os.Getenv(accountPasswordEnvKey(acc.ID)); pwd != "" {
			acc.Password = pwd
		} else if kdzs.Password != "" {
			acc.Password = kdzs.Password
		}
	}
}

func accountPasswordEnvKey(accountID string) string {
	norm := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(accountID))
	return "KDZS_ACCOUNT_" + norm + "_PASSWORD"
}
