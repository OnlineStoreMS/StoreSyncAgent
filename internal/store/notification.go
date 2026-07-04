package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	AccountIDs          []string `json:"accountIds,omitempty"` // 空=全部 accounts
	AppID               string   `json:"appId,omitempty"`
	AppSecret           string   `json:"appSecret,omitempty"`
}

type NotificationState struct {
	LastRunAt     string            `json:"lastRunAt,omitempty"`
	LastRunOK     bool              `json:"lastRunOk"`
	LastError     string            `json:"lastError,omitempty"`
	LastSentCount int               `json:"lastSentCount,omitempty"`
	Notified      map[string]string `json:"notified,omitempty"`
}

type NotificationData struct {
	Config NotificationConfig `json:"config"`
	State  NotificationState  `json:"state"`
}

type NotificationStore struct {
	path string
	mu   sync.Mutex
}

func NewNotificationStore(path string) (*NotificationStore, error) {
	if path == "" {
		return nil, fmt.Errorf("notification store path is empty")
	}
	s := &NotificationStore{path: path}
	if err := s.ensureFile(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *NotificationStore) ensureFile() error {
	if _, err := os.Stat(s.path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data := NotificationData{
		Config: NotificationConfig{
			Platform:            "FXG",
			PollIntervalMinutes: 15,
			DateRangeDays:       30,
		},
		State: NotificationState{Notified: map[string]string{}},
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}

func (s *NotificationStore) Load() (NotificationData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *NotificationStore) loadLocked() (NotificationData, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return NotificationData{}, err
	}
	var data NotificationData
	if err := json.Unmarshal(raw, &data); err != nil {
		return NotificationData{}, err
	}
	if data.State.Notified == nil {
		data.State.Notified = map[string]string{}
	}
	normalizeNotificationConfig(&data.Config)
	return data, nil
}

func normalizeNotificationConfig(cfg *NotificationConfig) {
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

func (s *NotificationStore) SaveConfig(cfg NotificationConfig) (NotificationData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return NotificationData{}, err
	}
	if cfg.Secret == "" {
		cfg.Secret = data.Config.Secret
	}
	if cfg.AppSecret == "" {
		cfg.AppSecret = data.Config.AppSecret
	}
	normalizeNotificationConfig(&cfg)
	data.Config = cfg
	if err := s.saveLocked(data); err != nil {
		return NotificationData{}, err
	}
	return data, nil
}

func (s *NotificationStore) SaveState(state NotificationState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return err
	}
	if state.Notified == nil {
		state.Notified = map[string]string{}
	}
	pruneNotified(state.Notified, 60)
	data.State = state
	return s.saveLocked(data)
}

func (s *NotificationStore) UpdateState(fn func(*NotificationState) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return err
	}
	if data.State.Notified == nil {
		data.State.Notified = map[string]string{}
	}
	if err := fn(&data.State); err != nil {
		return err
	}
	pruneNotified(data.State.Notified, 60)
	return s.saveLocked(data)
}

func (s *NotificationStore) ResetState() (cleared int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return 0, err
	}
	cleared = len(data.State.Notified)
	data.State = NotificationState{Notified: map[string]string{}}
	if err := s.saveLocked(data); err != nil {
		return 0, err
	}
	return cleared, nil
}

func pruneNotified(notified map[string]string, keepDays int) {
	if len(notified) == 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	for key, at := range notified {
		t, ok := parseStoreTime(at)
		if !ok || t.Before(cutoff) {
			delete(notified, key)
		}
	}
	if len(notified) > 10000 {
		type kv struct {
			key string
			at  time.Time
		}
		list := make([]kv, 0, len(notified))
		for key, at := range notified {
			t, ok := parseStoreTime(at)
			if !ok {
				delete(notified, key)
				continue
			}
			list = append(list, kv{key: key, at: t})
		}
		// drop oldest beyond cap
		if len(list) > 8000 {
			// simple trim: delete arbitrary oldest half
			for key := range notified {
				if len(notified) <= 8000 {
					break
				}
				delete(notified, key)
			}
		}
	}
}

func parseStoreTime(s string) (time.Time, bool) {
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

func (s *NotificationStore) saveLocked(data NotificationData) error {
	if data.State.Notified == nil {
		data.State.Notified = map[string]string{}
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
