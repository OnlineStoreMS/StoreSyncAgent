package scheduler

import (
	"context"
	"log"
	"time"

	"storesyncagent/internal/service"
)

type StockAlertScheduler struct {
	mgr    *service.Manager
	stopCh chan struct{}
}

func NewStockAlertScheduler(mgr *service.Manager) *StockAlertScheduler {
	return &StockAlertScheduler{
		mgr:    mgr,
		stopCh: make(chan struct{}),
	}
}

func (s *StockAlertScheduler) Start() {
	go s.loop()
}

func (s *StockAlertScheduler) Stop() {
	close(s.stopCh)
}

func (s *StockAlertScheduler) loop() {
	timer := time.NewTimer(45 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			s.runOnce()
			timer.Reset(s.mgr.StockAlertPollInterval())
		}
	}
}

func (s *StockAlertScheduler) runOnce() {
	if !s.mgr.StockAlertEnabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	sent, skipped, err := s.mgr.RunStockAlertPollForAll(ctx)
	if err != nil {
		log.Printf("[stock-alert] poll failed: %v", err)
		return
	}
	if sent > 0 {
		log.Printf("[stock-alert] sent %d message(s), skipped %d", sent, skipped)
	}
}
