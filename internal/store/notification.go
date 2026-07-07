package store

import (
	"strings"
	"time"
)

type NotificationConfig struct {
	Enabled             bool     `json:"enabled"`
	WebhookURL          string   `json:"webhookUrl"`
	Secret              string   `json:"secret,omitempty"`
	Platform            string   `json:"platform,omitempty"`
	PollIntervalMinutes int      `json:"pollIntervalMinutes,omitempty"`
	DateRangeDays       int      `json:"dateRangeDays,omitempty"`
	Scenarios           []string `json:"scenarios,omitempty"`
	AccountIDs          []string `json:"accountIds,omitempty"` // 空=当前租户已启用的全部 KDZS 账号
	AppID               string   `json:"appId,omitempty"`
	AppSecret           string   `json:"appSecret,omitempty"`
}

type NotificationState struct {
	LastRunAt        string            `json:"lastRunAt,omitempty"`
	LastRunOK        bool              `json:"lastRunOk"`
	LastError        string            `json:"lastError,omitempty"`
	LastSentCount    int               `json:"lastSentCount,omitempty"`
	LastBarcodeError string            `json:"lastBarcodeError,omitempty"`
	Notified         map[string]string `json:"notified,omitempty"`
}

type NotificationData struct {
	Config NotificationConfig `json:"config"`
	State  NotificationState  `json:"state"`
}

func NormalizeNotificationConfig(cfg *NotificationConfig) {
	if cfg.Platform == "" {
		cfg.Platform = "FXG"
	}
	if cfg.PollIntervalMinutes <= 0 {
		cfg.PollIntervalMinutes = 15
	}
	if cfg.PollIntervalMinutes < 5 {
		cfg.PollIntervalMinutes = 5
	}
	if cfg.DateRangeDays <= 0 {
		cfg.DateRangeDays = 30
	}
}

func PruneNotified(notified map[string]string, keepDays int) {
	if len(notified) == 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	for key, at := range notified {
		t, ok := ParseStoreTime(at)
		if !ok || t.Before(cutoff) {
			delete(notified, key)
		}
	}
	if len(notified) > 10000 {
		for key := range notified {
			if len(notified) <= 8000 {
				break
			}
			delete(notified, key)
		}
	}
}

func ParseStoreTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
