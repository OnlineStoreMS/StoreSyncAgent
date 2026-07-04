package feishu

import "testing"

func TestInteractiveCardFields(t *testing.T) {
	card := InteractiveCard{
		Title:    "售后通知 · 时效紧迫",
		Template: "red",
		Markdown: "**店铺：** <font color='blue'>测试店</font>",
	}
	if card.Title == "" || card.Template != "red" {
		t.Fatal("unexpected card")
	}
}
