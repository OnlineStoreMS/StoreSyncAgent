package kdzs

import (
	"context"
	"fmt"
	"strings"
)

// HandOrderType 快递助手手工单创建动作。
const (
	HandOrderTypePush     = "1" // 创建并推单
	HandOrderTypeWaitPush = "2" // 创建到待推单
	HandOrderTypePrint    = "3" // 创建并打印
	HandOrderTypeWaitSend = "4" // 创建到待发货
)

// HandOrderSku 手工单商品行（对齐 hand.kdzs.com handProductDataAll）。
type HandOrderSku struct {
	ItemID     string `json:"itemId,omitempty"`
	ItemCode   string `json:"itemCode,omitempty"`
	ItemName   string `json:"itemName,omitempty"`
	ItemPic    string `json:"itemPic,omitempty"`
	SkuID      string `json:"skuId,omitempty"`
	SkuCode    string `json:"skuCode,omitempty"`
	SkuName    string `json:"skuName,omitempty"`
	SkuPic     string `json:"skuPic,omitempty"`
	Num        string `json:"num,omitempty"`
	SkuSpec    string `json:"skuSpec,omitempty"`
	ShortName  string `json:"shortName,omitempty"`
	PicPath    string `json:"picPath,omitempty"`
	OuterID    string `json:"outerId,omitempty"`
	SkuOuterID string `json:"skuOuterId,omitempty"`
	Platform   string `json:"platform,omitempty"`
	ShopID     string `json:"shopId,omitempty"`
	ShopName   string `json:"shopName,omitempty"`
}

// HandOrderCreateRequest 单个手工建单请求。
type HandOrderCreateRequest struct {
	Recipient      string         `json:"recipient"`
	Phone          string         `json:"phone"`
	Tel            string         `json:"tel,omitempty"`
	Province       string         `json:"province"`
	City           string         `json:"city"`
	County         string         `json:"county"`
	ReceiveAddress string         `json:"receiveaddress"`
	SaveRecipient  bool           `json:"saveRecipient"`
	SkuList        []HandOrderSku `json:"skuList"`
	Remark         string         `json:"remark,omitempty"`
	SellerFlag     *int           `json:"sellerFlag"`
	HandSupplyType any            `json:"handSupplyType"`
	SupplyID       string         `json:"supplyId,omitempty"`
	Type           string         `json:"type"` // 1推单 2待推单 3打印 4待发货
	SendInfo       string         `json:"sendInfo,omitempty"`
	// ShipInfo 列表「发货内容」列读取该字段；与 sendInfo 同值一并提交，避免只写 sendInfo 列表不显示
	ShipInfo       string         `json:"shipInfo,omitempty"`
	OrderCode      string         `json:"orderCode,omitempty"`
}

// HandOrderReceiver 批量建单中的收件人。
type HandOrderReceiver struct {
	Recipient      string `json:"recipient"`
	Phone          string `json:"phone"`
	Tel            string `json:"tel,omitempty"`
	Province       string `json:"province"`
	City           string `json:"city"`
	County         string `json:"county"`
	ReceiveAddress string `json:"receiveaddress"`
}

// HandOrderBatchCreateRequest 批量手工建单。
type HandOrderBatchCreateRequest struct {
	HandOrderCreateRequest
	ReceiverInfoList []HandOrderReceiver `json:"receiverInfoList"`
}

// HandOrderCreateResult 建单结果。
type HandOrderCreateResult struct {
	AllFail           bool              `json:"allFail"`
	AllSuccess        bool              `json:"allSuccess"`
	FailList          []string          `json:"failList"`
	FailMessageMap    map[string]string `json:"failMessageMap"`
	SuccessList       []string          `json:"successList"`
	SuccessNotItemList []string         `json:"successNotItemList"`
	SuccessRealList   []string          `json:"successRealList"`
	Message           string            `json:"message"`
	Result            int               `json:"result"`
}

// AddressParseResult 一键填充地址识别结果。
type AddressParseResult struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Tel   string `json:"tel"`
	Address struct {
		Province string `json:"province"`
		City     string `json:"city"`
		District string `json:"district"`
		Detail   string `json:"detail"`
		Str      string `json:"str"`
	} `json:"address"`
	ShipContent string `json:"shipContent"`
}

// AddressParse 识别单条收件人文本。
func (s *Session) AddressParse(ctx context.Context, rawAddress string) (*AddressParseResult, error) {
	ps, err := s.PlatformSession(ctx, PlatformManual)
	if err != nil {
		return nil, err
	}
	var resp APIResponse[AddressParseResult]
	if err := s.client.PostPlatform(ctx, ps, "/handOrder/addressParse", map[string]any{
		"rawAddress": strings.TrimSpace(rawAddress),
	}, &resp); err != nil {
		return nil, err
	}
	data, err := checkResult(&resp)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// AddressBatchParse 批量识别收件人文本（换行分隔）。
func (s *Session) AddressBatchParse(ctx context.Context, rawAddress string) ([]AddressParseResult, error) {
	ps, err := s.PlatformSession(ctx, PlatformManual)
	if err != nil {
		return nil, err
	}
	var resp APIResponse[[]AddressParseResult]
	if err := s.client.PostPlatform(ctx, ps, "/handOrder/addressBatchParse", map[string]any{
		"rawAddress": strings.TrimSpace(rawAddress),
	}, &resp); err != nil {
		return nil, err
	}
	data, err := checkResult(&resp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// CreateHandOrder 在快递助手创建单个手工单（默认到待推单）。
func (s *Session) CreateHandOrder(ctx context.Context, req HandOrderCreateRequest) (*HandOrderCreateResult, error) {
	ps, err := s.PlatformSession(ctx, PlatformManual)
	if err != nil {
		return nil, err
	}
	normalizeHandOrder(&req)
	var resp APIResponse[HandOrderCreateResult]
	if err := s.client.PostPlatform(ctx, ps, "/handOrder/add", req, &resp); err != nil {
		return nil, err
	}
	data, err := checkResult(&resp)
	if err != nil {
		return nil, err
	}
	if data.AllFail {
		msg := data.Message
		if msg == "" && len(data.FailMessageMap) > 0 {
			for _, v := range data.FailMessageMap {
				msg = v
				break
			}
		}
		if msg == "" {
			msg = "快递助手建单失败"
		}
		return &data, fmt.Errorf("%s", msg)
	}
	return &data, nil
}

// BatchCreateHandOrder 批量创建手工单。
func (s *Session) BatchCreateHandOrder(ctx context.Context, req HandOrderBatchCreateRequest) (*HandOrderCreateResult, error) {
	ps, err := s.PlatformSession(ctx, PlatformManual)
	if err != nil {
		return nil, err
	}
	normalizeHandOrder(&req.HandOrderCreateRequest)
	if len(req.ReceiverInfoList) == 0 {
		return nil, fmt.Errorf("receiverInfoList is empty")
	}
	var resp APIResponse[HandOrderCreateResult]
	if err := s.client.PostPlatform(ctx, ps, "/handOrder/batchAdd", req, &resp); err != nil {
		return nil, err
	}
	data, err := checkResult(&resp)
	if err != nil {
		return nil, err
	}
	if data.AllFail {
		msg := data.Message
		if msg == "" {
			msg = "快递助手批量建单失败"
		}
		return &data, fmt.Errorf("%s", msg)
	}
	return &data, nil
}

func normalizeHandOrder(req *HandOrderCreateRequest) {
	if req.Type == "" {
		req.Type = HandOrderTypeWaitPush
	}
	if req.SkuList == nil {
		req.SkuList = []HandOrderSku{}
	}
	send := strings.TrimSpace(req.SendInfo)
	ship := strings.TrimSpace(req.ShipInfo)
	if send == "" {
		send = ship
	}
	if ship == "" {
		ship = send
	}
	req.SendInfo = send
	req.ShipInfo = ship
	for i := range req.SkuList {
		if strings.TrimSpace(req.SkuList[i].Num) == "" {
			req.SkuList[i].Num = "1"
		}
		if req.SkuList[i].PicPath == "" {
			req.SkuList[i].PicPath = req.SkuList[i].ItemPic
		}
		if req.SkuList[i].PicPath == "" {
			req.SkuList[i].PicPath = req.SkuList[i].SkuPic
		}
	}
}
