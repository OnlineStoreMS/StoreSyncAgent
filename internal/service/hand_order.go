package service

import (
	"context"
	"fmt"
	"strings"

	"storesyncagent/internal/config"
	"storesyncagent/internal/kdzs"
)

// ensureLoginDefaultAccount 使用租户「默认」快递助手账号登录（手工建单同步约定）。
// 优先 settings.DefaultAccountCode；未配置时回退到启用列表第一个。
func (s *SyncService) ensureLoginDefaultAccount(ctx context.Context) error {
	return s.ensureLoginAccount(ctx, "")
}

// ensureLoginAccount 登录指定账号；accountID 为空时走租户默认账号。
func (s *SyncService) ensureLoginAccount(ctx context.Context, accountID string) error {
	if err := s.loadSettings(); err != nil {
		return err
	}
	accounts, err := s.resolveAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return fmt.Errorf("当前租户未配置快递助手账号，请先在「账号管理」中添加")
	}
	var acc config.KdzsAccount
	var ok bool
	code := strings.TrimSpace(accountID)
	if code != "" {
		acc, ok = s.accountByCode(code)
		if !ok {
			return fmt.Errorf("快递助手账号 %s 不可用或已停用", code)
		}
	} else if s.settings != nil && strings.TrimSpace(s.settings.DefaultAccountCode) != "" {
		acc, ok = s.accountByCode(s.settings.DefaultAccountCode)
		if !ok {
			return fmt.Errorf("默认快递助手账号不可用或已停用，请在「账号管理」中重新设置默认账号")
		}
	} else {
		acc = accounts[0]
	}
	if strings.TrimSpace(acc.Mobile) == "" || strings.TrimSpace(acc.Password) == "" {
		return fmt.Errorf("快递助手账号缺少手机号或密码")
	}
	return s.session.SwitchAccount(ctx, acc.ID, acc.Name, acc.Role, acc.Mobile, acc.Password)
}

type ParseAddressRequest struct {
	RawAddress string `json:"rawAddress"`
	Batch      bool   `json:"batch"`
}

func (s *SyncService) ParseHandAddress(ctx context.Context, req ParseAddressRequest) (any, error) {
	if err := s.ensureLoginDefaultAccount(ctx); err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(req.RawAddress)
	if raw == "" {
		return nil, fmt.Errorf("rawAddress is required")
	}
	if req.Batch {
		return s.session.AddressBatchParse(ctx, raw)
	}
	return s.session.AddressParse(ctx, raw)
}

type CreateHandOrderRequest struct {
	Recipient      string              `json:"recipient"`
	Phone          string              `json:"phone"`
	Tel            string              `json:"tel"`
	Province       string              `json:"province"`
	City           string              `json:"city"`
	County         string              `json:"county"`
	ReceiveAddress string              `json:"receiveaddress"`
	SaveRecipient  bool                `json:"saveRecipient"`
	SkuList        []kdzs.HandOrderSku `json:"skuList"`
	Remark         string              `json:"remark"`
	SellerFlag     *int                `json:"sellerFlag"`
	SendInfo       string              `json:"sendInfo"`
	OrderCode      string              `json:"orderCode"`
	Type           string              `json:"type"` // 默认 2=待推单
	// AccountID 指定登录账号（如发货中心默认账号 code）；空则用 SSA 租户默认账号
	AccountID string `json:"accountId"`
}

type CreateHandOrderResult struct {
	AccountCode string                    `json:"accountCode"`
	AccountName string                    `json:"accountName"`
	KDZS        *kdzs.HandOrderCreateResult `json:"kdzs"`
	SysTid      string                    `json:"sysTid"`
	Tid         string                    `json:"tid"`
}

func (s *SyncService) CreateHandOrder(ctx context.Context, req CreateHandOrderRequest) (*CreateHandOrderResult, error) {
	if err := s.ensureLoginAccount(ctx, req.AccountID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Recipient) == "" {
		return nil, fmt.Errorf("recipient is required")
	}
	if strings.TrimSpace(req.Phone) == "" && strings.TrimSpace(req.Tel) == "" {
		return nil, fmt.Errorf("phone or tel is required")
	}
	if strings.TrimSpace(req.Province) == "" || strings.TrimSpace(req.City) == "" || strings.TrimSpace(req.County) == "" {
		return nil, fmt.Errorf("province/city/county are required")
	}
	if strings.TrimSpace(req.ReceiveAddress) == "" {
		return nil, fmt.Errorf("receiveaddress is required")
	}
	accID := s.session.AccountID()
	accName := s.session.AccountName()
	// 对齐快递助手：有 sku 时忽略发货内容；无 sku 时才写入 sendInfo/shipInfo
	sendInfo := ""
	if len(req.SkuList) == 0 {
		sendInfo = strings.TrimSpace(req.SendInfo)
	}
	result, err := s.session.CreateHandOrder(ctx, kdzs.HandOrderCreateRequest{
		Recipient:      req.Recipient,
		Phone:          req.Phone,
		Tel:            req.Tel,
		Province:       req.Province,
		City:           req.City,
		County:         req.County,
		ReceiveAddress: req.ReceiveAddress,
		SaveRecipient:  req.SaveRecipient,
		SkuList:        req.SkuList,
		Remark:         req.Remark,
		SellerFlag:     req.SellerFlag,
		HandSupplyType: nil,
		SendInfo:       sendInfo,
		ShipInfo:       sendInfo,
		OrderCode:      req.OrderCode,
		Type:           req.Type,
	})
	if err != nil {
		return nil, err
	}
	out := &CreateHandOrderResult{
		AccountCode: accID,
		AccountName: accName,
		KDZS:        result,
	}
	if result != nil {
		if len(result.SuccessRealList) > 0 {
			out.Tid = result.SuccessRealList[0]
		}
		if len(result.SuccessList) > 0 {
			out.SysTid = result.SuccessList[0]
		}
		if out.Tid == "" && len(result.SuccessNotItemList) > 0 {
			out.Tid = result.SuccessNotItemList[0]
		}
	}
	return out, nil
}

type BatchCreateHandOrderRequest struct {
	CreateHandOrderRequest
	Receivers []kdzs.HandOrderReceiver `json:"receivers"`
}

type BatchCreateHandOrderResult struct {
	AccountCode string                      `json:"accountCode"`
	AccountName string                      `json:"accountName"`
	KDZS        *kdzs.HandOrderCreateResult `json:"kdzs"`
}

func (s *SyncService) BatchCreateHandOrder(ctx context.Context, req BatchCreateHandOrderRequest) (*BatchCreateHandOrderResult, error) {
	if err := s.ensureLoginAccount(ctx, req.AccountID); err != nil {
		return nil, err
	}
	if len(req.Receivers) == 0 {
		return nil, fmt.Errorf("receivers is required")
	}
	accID := s.session.AccountID()
	accName := s.session.AccountName()
	base := req.CreateHandOrderRequest
	sendInfo := ""
	if len(base.SkuList) == 0 {
		sendInfo = strings.TrimSpace(base.SendInfo)
	}
	result, err := s.session.BatchCreateHandOrder(ctx, kdzs.HandOrderBatchCreateRequest{
		HandOrderCreateRequest: kdzs.HandOrderCreateRequest{
			Recipient:      base.Recipient,
			Phone:          base.Phone,
			Tel:            base.Tel,
			Province:       base.Province,
			City:           base.City,
			County:         base.County,
			ReceiveAddress: base.ReceiveAddress,
			SaveRecipient:  base.SaveRecipient,
			SkuList:        base.SkuList,
			Remark:         base.Remark,
			SellerFlag:     base.SellerFlag,
			HandSupplyType: nil,
			SendInfo:       sendInfo,
			ShipInfo:       sendInfo,
			OrderCode:      base.OrderCode,
			Type:           base.Type,
		},
		ReceiverInfoList: req.Receivers,
	})
	if err != nil {
		return nil, err
	}
	return &BatchCreateHandOrderResult{
		AccountCode: accID,
		AccountName: accName,
		KDZS:        result,
	}, nil
}
