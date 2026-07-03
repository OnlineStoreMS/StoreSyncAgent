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
