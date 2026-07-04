package feishu

import (
	"context"
)

// InteractiveCard 飞书消息卡片（schema 2.0）。
type InteractiveCard struct {
	Title    string
	Template string // blue | green | orange | red | purple | wathet | ...
	Markdown string
}

func (c *Client) SendInteractiveCard(ctx context.Context, webhookURL, secret string, card InteractiveCard) error {
	if card.Title == "" {
		card.Title = "通知"
	}
	if card.Template == "" {
		card.Template = "blue"
	}
	return c.postSignedJSON(ctx, webhookURL, secret, map[string]any{
		"msg_type": "interactive",
		"card": map[string]any{
			"schema": "2.0",
			"config": map[string]any{
				"update_multi": true,
			},
			"header": map[string]any{
				"title": map[string]any{
					"tag":     "plain_text",
					"content": card.Title,
				},
				"template": card.Template,
			},
			"body": map[string]any{
				"direction": "vertical",
				"elements": []any{
					map[string]any{
						"tag":        "markdown",
						"content":    card.Markdown,
						"text_align": "left",
					},
				},
			},
		},
	})
}
