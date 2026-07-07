package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"storesyncagent/internal/config"
	"storesyncagent/internal/repo"
)

type Manager struct {
	baseCfg  *config.Config
	kdzsRepo *repo.KdzsRepo
	mu       sync.Mutex
	services map[uint64]*SyncService
}

func NewManager(baseCfg *config.Config, kdzsRepo *repo.KdzsRepo) *Manager {
	return &Manager{
		baseCfg:  baseCfg,
		kdzsRepo: kdzsRepo,
		services: make(map[uint64]*SyncService),
	}
}

func (m *Manager) KdzsRepo() *repo.KdzsRepo {
	return m.kdzsRepo
}

func (m *Manager) InvalidateTenant(tenantID uint64) {
	m.mu.Lock()
	delete(m.services, tenantID)
	m.mu.Unlock()
}

func (m *Manager) ForTenant(tenantID uint64) (*SyncService, error) {
	if tenantID == 0 {
		return nil, fmt.Errorf("tenant required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if svc, ok := m.services[tenantID]; ok {
		return svc, nil
	}
	svc, err := NewSyncService(m.baseCfg, tenantID, m.kdzsRepo)
	if err != nil {
		return nil, err
	}
	m.services[tenantID] = svc
	return svc, nil
}

func (m *Manager) ListTenantIDs() []uint64 {
	seen := make(map[uint64]struct{})
	add := func(id uint64) {
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
	}

	if ids, err := m.kdzsRepo.ListTenantIDs(); err == nil {
		for _, id := range ids {
			add(id)
		}
	}

	base := m.baseCfg.Storage.DataDir
	if base == "" {
		base = "data"
	}
	tenantRoot := filepath.Join(base, "tenants")
	entries, err := os.ReadDir(tenantRoot)
	if err == nil {
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			id, err := strconv.ParseUint(ent.Name(), 10, 64)
			if err != nil {
				continue
			}
			add(id)
		}
	}

	if len(seen) == 0 {
		add(1)
	}

	ids := make([]uint64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}

func (m *Manager) NotificationEnabled() bool {
	for _, tid := range m.ListTenantIDs() {
		svc, err := m.ForTenant(tid)
		if err != nil {
			continue
		}
		if svc.NotificationEnabled() {
			return true
		}
	}
	return false
}

func (m *Manager) RunNotificationPollForAll(ctx context.Context) (sent, skipped int, lastErr error) {
	for _, tid := range m.ListTenantIDs() {
		svc, err := m.ForTenant(tid)
		if err != nil || !svc.NotificationEnabled() {
			continue
		}
		result, err := svc.RunNotificationPoll(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		if result != nil {
			sent += result.Sent
			skipped += result.Skipped
		}
	}
	return sent, skipped, lastErr
}

func (m *Manager) NotificationPollInterval() time.Duration {
	min := 15 * time.Minute
	found := false
	for _, tid := range m.ListTenantIDs() {
		svc, err := m.ForTenant(tid)
		if err != nil {
			continue
		}
		iv := svc.NotificationPollInterval()
		if !found || iv < min {
			min = iv
			found = true
		}
	}
	return min
}
