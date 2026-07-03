package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Kdzs    KdzsConfig    `mapstructure:"kdzs"`
	Storage StorageConfig `mapstructure:"storage"`
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
	applyEnvSecrets(&cfg)
	return &cfg, nil
}

func applyEnvSecrets(cfg *Config) {
	if pwd := os.Getenv("KDZS_PASSWORD"); pwd != "" {
		cfg.Kdzs.Password = pwd
	}
	if mobile := os.Getenv("KDZS_MOBILE"); mobile != "" {
		cfg.Kdzs.Mobile = mobile
	}
	for i := range cfg.Kdzs.Accounts {
		acc := &cfg.Kdzs.Accounts[i]
		if acc.Password != "" {
			continue
		}
		if pwd := os.Getenv(accountPasswordEnvKey(acc.ID)); pwd != "" {
			acc.Password = pwd
		} else if cfg.Kdzs.Password != "" {
			acc.Password = cfg.Kdzs.Password
		}
	}
}

func accountPasswordEnvKey(accountID string) string {
	norm := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(accountID))
	return "KDZS_ACCOUNT_" + norm + "_PASSWORD"
}
