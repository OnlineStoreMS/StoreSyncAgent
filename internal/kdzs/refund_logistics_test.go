package kdzs

import "testing"

func TestIsPickupPendingTrace(t *testing.T) {
	outbound := "收件员从【丰巢智能快递柜】取出快件(收件员:黄军,电话：19823479788)"
	if isPickupPendingTrace("TRANSPORT", "TRANSPORT", outbound) {
		t.Fatal("outbound locker pickup should not be pickup pending")
	}

	cases := []struct {
		desc string
		want bool
	}{
		{"您的快件已放入丰巢智能快递柜，请凭取件码领取", true},
		{"快件到达菜鸟驿站，请凭取件码及时领取", true},
		{"【代收点】您的快件已暂存至驿站，请尽快领取", true},
		{"快件在【XX店】完成分拣，准备发往 【YY转运中心】", false},
		{"顺丰速运 已收取快件", false},
		{"收件员从【丰巢智能快递柜】取出快件", false},
	}
	for _, c := range cases {
		got := isPickupPendingTrace("", "", c.desc)
		if got != c.want {
			t.Fatalf("desc=%q got=%v want=%v", c.desc, got, c.want)
		}
	}
}

func TestIsDescSigned(t *testing.T) {
	cases := []struct {
		desc string
		want bool
	}{
		{"快件正在派送中，请耐心等待，保持电话畅通，准备签收", false},
		{"您的快件已签收，签收人：本人", true},
		{"待签收", false},
		{"已代收，请尽快领取", false},
	}
	for _, c := range cases {
		got := isDescSigned(c.desc)
		if got != c.want {
			t.Fatalf("desc=%q got=%v want=%v", c.desc, got, c.want)
		}
	}
}

func TestParseLogisticsDetailChinaPostAgentPickup(t *testing.T) {
	resp := &logisticsDetailResponse{}
	resp.Data.LogisticsStatus = "DELIVERING"
	resp.Data.LogisticsStatusDesc = "派件中"
	resp.Data.LogisticsTraceDetailList = []struct {
		Time               string `json:"time"`
		Desc               string `json:"desc"`
		LogisticsStatus    string `json:"logisticsStatus"`
		SubLogisticsStatus string `json:"subLogisticsStatus"`
	}{
		{Time: "2026-07-02 10:46:21", Desc: "中国邮政已收取快件", LogisticsStatus: "ACCEPT", SubLogisticsStatus: "ACCEPT"},
		{
			Time: "2026-07-04 08:46:16",
			Desc: "快件正在派送中，请耐心等待，保持电话畅通，准备签收，如有疑问请电联快递员",
			LogisticsStatus: "DELIVERING", SubLogisticsStatus: "DELIVERING",
		},
		{
			Time: "2026-07-04 10:15:15",
			Desc: "您的快件已代收，如有疑问请电联快递员",
			LogisticsStatus: "DELIVERING", SubLogisticsStatus: "STA_INBOUND",
		},
	}
	detail := parseLogisticsDetail(resp)
	if detail.IsSigned {
		t.Fatal("准备签收+已代收不应判定为已签收")
	}
	if !detail.IsPickupPending {
		t.Fatal("已代收应识别为驿站待取件")
	}
}

func TestParseLogisticsDetailPickupDirection(t *testing.T) {
	resp := &logisticsDetailResponse{}
	resp.Data.LogisticsStatus = "TRANSPORT"
	resp.Data.LogisticsStatusDesc = "运输中"
	resp.Data.LogisticsTraceDetailList = []struct {
		Time               string `json:"time"`
		Desc               string `json:"desc"`
		LogisticsStatus    string `json:"logisticsStatus"`
		SubLogisticsStatus string `json:"subLogisticsStatus"`
	}{
		{Time: "2026-07-01 10:00:00", Desc: "收件员从【丰巢智能快递柜】取出快件(收件员:黄军)", LogisticsStatus: "ACCEPT"},
		{Time: "2026-07-02 12:00:00", Desc: "快件离开 【北京转运中心】", LogisticsStatus: "TRANSPORT"},
	}
	detail := parseLogisticsDetail(resp)
	if detail.IsPickupPending {
		t.Fatal("should not mark outbound locker pickup as pending")
	}
}
