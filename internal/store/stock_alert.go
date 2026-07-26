package store

type StockAlertConfig struct {
	Enabled             bool     `json:"enabled"`
	WebhookURL          string   `json:"webhookUrl"`
	Secret              string   `json:"secret,omitempty"`
	Platform            string   `json:"platform,omitempty"`
	ShopIDs             []string `json:"shopIds,omitempty"`
	StockThreshold      int      `json:"stockThreshold,omitempty"`
	CheckLevel          string   `json:"checkLevel,omitempty"` // sku | spu | both
	OnlyOnsale          bool     `json:"onlyOnsale"`
	PollIntervalMinutes int      `json:"pollIntervalMinutes,omitempty"`
}

type StockAlertState struct {
	LastRunAt     string            `json:"lastRunAt,omitempty"`
	LastRunOK     bool              `json:"lastRunOk"`
	LastError     string            `json:"lastError,omitempty"`
	LastSentCount int               `json:"lastSentCount,omitempty"`
	LastAlertCount int              `json:"lastAlertCount,omitempty"`
	Notified      map[string]string `json:"notified,omitempty"`
}

type StockAlertData struct {
	Config StockAlertConfig `json:"config"`
	State  StockAlertState  `json:"state"`
}

func NormalizeStockAlertConfig(cfg *StockAlertConfig) {
	if cfg.Platform == "" {
		cfg.Platform = "FXG"
	}
	switch cfg.CheckLevel {
	case "spu", "sku", "both":
	default:
		cfg.CheckLevel = "sku"
	}
	if cfg.StockThreshold < 0 {
		cfg.StockThreshold = 0
	}
	if cfg.PollIntervalMinutes <= 0 {
		cfg.PollIntervalMinutes = 60
	}
	if cfg.PollIntervalMinutes < 15 {
		cfg.PollIntervalMinutes = 15
	}
}
