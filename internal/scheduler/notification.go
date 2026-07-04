package scheduler

import (
	"context"
	"log"
	"time"

	"storesyncagent/internal/service"
)

type NotificationScheduler struct {
	svc    *service.SyncService
	stopCh chan struct{}
}

func NewNotificationScheduler(svc *service.SyncService) *NotificationScheduler {
	return &NotificationScheduler{
		svc:    svc,
		stopCh: make(chan struct{}),
	}
}

func (s *NotificationScheduler) Start() {
	go s.loop()
}

func (s *NotificationScheduler) Stop() {
	close(s.stopCh)
}

func (s *NotificationScheduler) loop() {
	// 启动后延迟 30 秒首次检查，避免与 API 启动争抢资源。
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-timer.C:
			s.runOnce()
			timer.Reset(s.svc.NotificationPollInterval())
		}
	}
}

func (s *NotificationScheduler) runOnce() {
	if !s.svc.NotificationEnabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	result, err := s.svc.RunNotificationPoll(ctx)
	if err != nil {
		log.Printf("[notification] poll failed: %v", err)
		return
	}
	if result.Sent > 0 {
		log.Printf("[notification] sent %d message(s), skipped %d", result.Sent, result.Skipped)
	}
}
