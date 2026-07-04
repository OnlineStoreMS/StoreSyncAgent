package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"storesyncagent/internal/kdzs"
	"storesyncagent/internal/store"
)

var notificationScenarioLabels = map[string]string{
	"urgent":            "时效紧迫",
	"pickup_pending":    "驿站待取件",
	"return_signed":     "退回已签收",
	"confirm_receive":   "待确认收货",
	"wait_agree":        "待卖家同意",
	"refund_only":       "仅退款提醒",
	"exchange":          "换货待处理",
	"wait_send_exchange": "待发出换货商品",
}

var supportedNotificationScenarios = []string{
	"urgent",
	"pickup_pending",
	"return_signed",
	"confirm_receive",
	"wait_agree",
	"refund_only",
	"exchange",
	"wait_send_exchange",
}

type NotificationView struct {
	Config   NotificationConfigView `json:"config"`
	State    store.NotificationState `json:"state"`
	Scenarios []ScenarioOption       `json:"scenarios"`
}

type ScenarioOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type NotificationConfigView struct {
	Enabled             bool     `json:"enabled"`
	WebhookURL          string   `json:"webhookUrl"`
	Secret              string   `json:"secret,omitempty"`
	SecretSet           bool     `json:"secretSet"`
	Platform            string   `json:"platform"`
	PollIntervalMinutes int      `json:"pollIntervalMinutes"`
	DateRangeDays       int      `json:"dateRangeDays"`
	Scenarios           []string `json:"scenarios"`
}

type NotificationRunResult struct {
	Sent   int    `json:"sent"`
	Skipped int   `json:"skipped"`
	Error  string `json:"error,omitempty"`
}

func (s *SyncService) SupportedNotificationScenarios() []ScenarioOption {
	out := make([]ScenarioOption, 0, len(supportedNotificationScenarios))
	for _, key := range supportedNotificationScenarios {
		out = append(out, ScenarioOption{Key: key, Label: notificationScenarioLabels[key]})
	}
	return out
}

func (s *SyncService) GetNotificationView() (*NotificationView, error) {
	data, err := s.notificationStore.Load()
	if err != nil {
		return nil, err
	}
	return &NotificationView{
		Config:    toNotificationConfigView(data.Config),
		State:     data.State,
		Scenarios: s.SupportedNotificationScenarios(),
	}, nil
}

func toNotificationConfigView(cfg store.NotificationConfig) NotificationConfigView {
	view := NotificationConfigView{
		Enabled:             cfg.Enabled,
		WebhookURL:          cfg.WebhookURL,
		Platform:            cfg.Platform,
		PollIntervalMinutes: cfg.PollIntervalMinutes,
		DateRangeDays:       cfg.DateRangeDays,
		Scenarios:           append([]string(nil), cfg.Scenarios...),
		SecretSet:           cfg.Secret != "",
	}
	return view
}

func (s *SyncService) SaveNotificationConfig(in store.NotificationConfig) (*NotificationView, error) {
	for _, sc := range in.Scenarios {
		if notificationScenarioLabels[sc] == "" {
			return nil, fmt.Errorf("unsupported scenario: %s", sc)
		}
	}
	data, err := s.notificationStore.SaveConfig(in)
	if err != nil {
		return nil, err
	}
	return &NotificationView{
		Config:    toNotificationConfigView(data.Config),
		State:     data.State,
		Scenarios: s.SupportedNotificationScenarios(),
	}, nil
}

func (s *SyncService) TestNotification(ctx context.Context, text string) error {
	data, err := s.notificationStore.Load()
	if err != nil {
		return err
	}
	cfg := data.Config
	if cfg.WebhookURL == "" {
		return fmt.Errorf("请先配置 Webhook 地址")
	}
	if text == "" {
		text = "【StoreSyncAgent】飞书通知测试消息"
	}
	return s.feishuClient.SendText(ctx, cfg.WebhookURL, cfg.Secret, text)
}

func (s *SyncService) RunNotificationPoll(ctx context.Context) (*NotificationRunResult, error) {
	data, err := s.notificationStore.Load()
	if err != nil {
		return nil, err
	}
	cfg := data.Config
	result := &NotificationRunResult{}
	now := time.Now()
	runAt := now.Format("2006-01-02 15:04:05")

	updateState := func(ok bool, sent int, errMsg string) {
		_ = s.notificationStore.UpdateState(func(st *store.NotificationState) error {
			st.LastRunAt = runAt
			st.LastRunOK = ok
			st.LastError = errMsg
			st.LastSentCount = sent
			return nil
		})
	}

	if !cfg.Enabled {
		updateState(true, 0, "")
		return result, nil
	}
	if cfg.WebhookURL == "" {
		err := fmt.Errorf("webhook url is empty")
		updateState(false, 0, err.Error())
		return nil, err
	}
	if len(cfg.Scenarios) == 0 {
		updateState(true, 0, "")
		return result, nil
	}

	if err := s.ensureLogin(ctx); err != nil {
		updateState(false, 0, err.Error())
		return nil, err
	}

	base := kdzs.RefundQuery{
		Platform: cfg.Platform,
		DateType: 4,
	}
	start, end := refundDateRange(cfg.DateRangeDays)
	base.StartDateTime = start
	base.EndDateTime = end

	sent := 0
	skipped := 0
	var sendErr error
	notified := data.State.Notified
	if notified == nil {
		notified = map[string]string{}
	}

	for _, scenario := range cfg.Scenarios {
		items, err := s.collectScenarioRefunds(ctx, cfg.Platform, scenario, base)
		if err != nil {
			sendErr = err
			break
		}
		for _, item := range items {
			key := notificationKey(item, scenario)
			if notified[key] != "" {
				skipped++
				continue
			}
			text := formatRefundNotification(scenario, item)
			if err := s.feishuClient.SendText(ctx, cfg.WebhookURL, cfg.Secret, text); err != nil {
				sendErr = err
				break
			}
			sent++
			notifyAt := time.Now().Format(time.RFC3339)
			notified[key] = notifyAt
			if err := s.notificationStore.UpdateState(func(st *store.NotificationState) error {
				if st.Notified == nil {
					st.Notified = map[string]string{}
				}
				st.Notified[key] = notifyAt
				return nil
			}); err != nil {
				sendErr = err
				break
			}
		}
		if sendErr != nil {
			break
		}
	}

	result.Sent = sent
	result.Skipped = skipped
	if sendErr != nil {
		updateState(false, sent, sendErr.Error())
		return result, sendErr
	}
	updateState(true, sent, "")
	return result, nil
}

func refundDateRange(days int) (string, string) {
	if days <= 0 {
		days = 30
	}
	now := time.Now()
	end := now.Format("2006-01-02 15:04:05")
	start := now.AddDate(0, 0, -(days - 1)).Truncate(24 * time.Hour).Format("2006-01-02 15:04:05")
	return start, end
}

func notificationKey(item kdzs.RefundItem, scenario string) string {
	if scenario == "urgent" && item.SLA != nil && item.SLA.Urgency != "" {
		return item.RefundID + ":urgent:" + item.SLA.Urgency
	}
	return item.RefundID + ":" + scenario
}

func formatRefundNotification(scenario string, item kdzs.RefundItem) string {
	label := notificationScenarioLabels[scenario]
	if label == "" {
		label = scenario
	}
	var b strings.Builder
	fmt.Fprintf(&b, "【售后通知 · %s】\n", label)
	if item.ShopName != "" {
		fmt.Fprintf(&b, "店铺：%s\n", item.ShopName)
	}
	if item.Tid != "" {
		fmt.Fprintf(&b, "订单号：%s\n", item.Tid)
	}
	if item.RefundID != "" {
		fmt.Fprintf(&b, "售后单：%s\n", item.RefundID)
	}
	if item.AfterSaleTypeText != "" {
		fmt.Fprintf(&b, "类型：%s\n", item.AfterSaleTypeText)
	}
	if item.AfterSaleStatusText != "" {
		fmt.Fprintf(&b, "状态：%s\n", item.AfterSaleStatusText)
	}
	if item.BuyerNick != "" {
		fmt.Fprintf(&b, "买家：%s\n", item.BuyerNick)
	}
	if item.Sid != "" {
		fmt.Fprintf(&b, "退货物流：%s\n", item.Sid)
	}
	if g := firstRefundGoods(item); g != nil {
		if g.Title != "" {
			fmt.Fprintf(&b, "商品：%s\n", g.Title)
		}
		if g.SkuName != "" {
			fmt.Fprintf(&b, "规格：%s\n", g.SkuName)
		}
	}
	if item.SLA != nil {
		if item.SLA.RemainingText != "" {
			fmt.Fprintf(&b, "时效：%s\n", item.SLA.RemainingText)
		}
		if item.SLA.Hint != "" {
			fmt.Fprintf(&b, "说明：%s\n", item.SLA.Hint)
		}
		if item.SLA.PickupHint != "" {
			fmt.Fprintf(&b, "物流：%s\n", truncateText(item.SLA.PickupHint, 120))
		}
	}
	return strings.TrimSpace(b.String())
}

func firstRefundGoods(item kdzs.RefundItem) *kdzs.RefundGoods {
	if len(item.Goods) == 0 {
		return nil
	}
	return &item.Goods[0]
}

func truncateText(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

func (s *SyncService) NotificationPollInterval() time.Duration {
	data, err := s.notificationStore.Load()
	if err != nil || !data.Config.Enabled {
		return 15 * time.Minute
	}
	mins := data.Config.PollIntervalMinutes
	if mins < 5 {
		mins = 5
	}
	return time.Duration(mins) * time.Minute
}

func (s *SyncService) NotificationEnabled() bool {
	data, err := s.notificationStore.Load()
	if err != nil {
		return false
	}
	return data.Config.Enabled && data.Config.WebhookURL != ""
}
