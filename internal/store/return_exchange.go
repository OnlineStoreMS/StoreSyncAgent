package store

type ReturnExchangeGoods struct {
	PicURL  string `json:"picUrl,omitempty"`
	SkuName string `json:"skuName,omitempty"`
}

type ReturnExchangeRecord struct {
	ID                    string                `json:"id"`
	SeqNo                 int                   `json:"seqNo,omitempty"`
	BuyerNick             string                `json:"buyerNick,omitempty"`
	AfterSaleType         string                `json:"afterSaleType,omitempty"`
	ReturnTrackingNo      string                `json:"returnTrackingNo,omitempty"`
	Spec                  string                `json:"spec,omitempty"`
	FeedbackTime          string                `json:"feedbackTime,omitempty"`
	SubmitTime            string                `json:"submitTime,omitempty"`
	OrderNo               string                `json:"orderNo,omitempty"`
	RecipientInfo         string                `json:"recipientInfo,omitempty"`
	ParsedRecipientInfo   string                `json:"parsedRecipientInfo,omitempty"`
	OutboundTrackingNo    string                `json:"outboundTrackingNo,omitempty"`
	Remark                string                `json:"remark,omitempty"`
	Platform              string                `json:"platform,omitempty"`
	SysTid                string                `json:"sysTid,omitempty"`
	ShopName              string                `json:"shopName,omitempty"`
	Goods                 []ReturnExchangeGoods `json:"goods,omitempty"`
	GoodsTitle            string                `json:"goodsTitle,omitempty"`
	OriginalRecipientInfo string                `json:"originalRecipientInfo,omitempty"`
	Payment               float64               `json:"payment,omitempty"`
	PayTime               string                `json:"payTime,omitempty"`
	StatusText            string                `json:"statusText,omitempty"`
	CreatedAt             string                `json:"createdAt,omitempty"`
	UpdatedAt             string                `json:"updatedAt,omitempty"`
}
