package service

import (
	"fmt"
	"strings"

	"storesyncagent/internal/config"
	"storesyncagent/internal/model"
	"storesyncagent/internal/repo"
)

type KdzsAccountInput struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
	Enabled  *bool  `json:"enabled"`
	SortOrder *int  `json:"sortOrder"`
}

type KdzsAccountDetail struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	RoleLabel   string `json:"roleLabel"`
	Mobile      string `json:"mobile"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sortOrder"`
	PasswordSet bool   `json:"passwordSet"`
	Active      bool   `json:"active"`
	IsDefault   bool   `json:"isDefault"`
}

func (s *SyncService) resolveAccounts() ([]config.KdzsAccount, error) {
	records, err := s.kdzsRepo.ListAccounts(s.tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]config.KdzsAccount, 0, len(records))
	for _, rec := range records {
		out = append(out, recordToConfigAccount(rec))
	}
	return out, nil
}

func (s *SyncService) accountByCode(code string) (config.KdzsAccount, bool) {
	rec, err := s.kdzsRepo.GetAccount(s.tenantID, code)
	if err != nil || rec == nil || !rec.Enabled {
		return config.KdzsAccount{}, false
	}
	return recordToConfigAccount(*rec), true
}

func recordToConfigAccount(rec model.KdzsAccount) config.KdzsAccount {
	return config.KdzsAccount{
		ID:       rec.Code,
		Name:     rec.Name,
		Role:     rec.Role,
		Mobile:   rec.Mobile,
		Password: rec.Password,
	}
}

func (s *SyncService) kdzsBaseURL() string {
	if s.settings != nil && s.settings.BaseURL != "" {
		return s.settings.BaseURL
	}
	if s.globalBaseURL != "" {
		return s.globalBaseURL
	}
	return "https://df.kdzs.com"
}

func (s *SyncService) loadSettings() error {
	settings, err := s.kdzsRepo.GetOrCreateSettings(s.tenantID, s.globalBaseURL)
	if err != nil {
		return err
	}
	s.settings = settings
	if settings.ActiveAccountCode != "" {
		s.activeAccountID = settings.ActiveAccountCode
	} else if settings.DefaultAccountCode != "" {
		s.activeAccountID = settings.DefaultAccountCode
	}
	return nil
}

func (s *SyncService) ensureDefaultActiveAccount() error {
	if s.activeAccountID != "" {
		if _, ok := s.accountByCode(s.activeAccountID); ok {
			return nil
		}
	}
	accounts, err := s.resolveAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return fmt.Errorf("当前租户未配置快递助手账号，请先在「账号管理」中添加")
	}
	if s.settings != nil && s.settings.DefaultAccountCode != "" {
		if _, ok := s.accountByCode(s.settings.DefaultAccountCode); ok {
			s.activeAccountID = s.settings.DefaultAccountCode
			return nil
		}
	}
	s.activeAccountID = accounts[0].ID
	return nil
}

func (s *SyncService) ListAccountDetails() ([]KdzsAccountDetail, error) {
	if err := s.loadSettings(); err != nil {
		return nil, err
	}
	records, err := s.kdzsRepo.ListAllAccounts(s.tenantID)
	if err != nil {
		return nil, err
	}
	items := make([]KdzsAccountDetail, 0, len(records))
	for _, rec := range records {
		items = append(items, KdzsAccountDetail{
			Code:        rec.Code,
			Name:        rec.Name,
			Role:        rec.Role,
			RoleLabel:   accountRoleLabel(rec.Role),
			Mobile:      rec.Mobile,
			Enabled:     rec.Enabled,
			SortOrder:   rec.SortOrder,
			PasswordSet: rec.Password != "",
			Active:      rec.Code == s.activeAccountID,
			IsDefault:   s.settings != nil && rec.Code == s.settings.DefaultAccountCode,
		})
	}
	return items, nil
}

func (s *SyncService) CreateKdzsAccount(in KdzsAccountInput) (*KdzsAccountDetail, error) {
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	in.Mobile = strings.TrimSpace(in.Mobile)
	in.Role = strings.TrimSpace(in.Role)
	if in.Code == "" {
		return nil, fmt.Errorf("账号 ID 不能为空")
	}
	if in.Mobile == "" {
		return nil, fmt.Errorf("手机号不能为空")
	}
	if in.Password == "" {
		return nil, fmt.Errorf("密码不能为空")
	}
	if in.Role == "" {
		in.Role = "merchant"
	}
	if in.Name == "" {
		in.Name = in.Mobile
	}
	if _, err := s.kdzsRepo.GetAccount(s.tenantID, in.Code); err == nil {
		return nil, fmt.Errorf("账号 %s 已存在", in.Code)
	} else if err != repo.ErrNotFound {
		return nil, err
	}
	rec := &model.KdzsAccount{
		TenantID:  s.tenantID,
		Code:      in.Code,
		Name:      in.Name,
		Role:      in.Role,
		Mobile:    in.Mobile,
		Password:  in.Password,
		Enabled:   true,
		SortOrder: 0,
	}
	if in.Enabled != nil {
		rec.Enabled = *in.Enabled
	}
	if in.SortOrder != nil {
		rec.SortOrder = *in.SortOrder
	}
	if err := s.kdzsRepo.CreateAccount(rec); err != nil {
		return nil, err
	}
	if err := s.loadSettings(); err != nil {
		return nil, err
	}
	count, _ := s.kdzsRepo.CountAccounts(s.tenantID)
	if count == 1 || s.settings.DefaultAccountCode == "" {
		s.settings.DefaultAccountCode = rec.Code
		s.settings.ActiveAccountCode = rec.Code
		s.activeAccountID = rec.Code
		_ = s.kdzsRepo.SaveSettings(s.settings)
	}
	items, err := s.ListAccountDetails()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Code == rec.Code {
			return &items[i], nil
		}
	}
	return nil, fmt.Errorf("account created but not found")
}

func (s *SyncService) UpdateKdzsAccount(code string, in KdzsAccountInput) (*KdzsAccountDetail, error) {
	rec, err := s.kdzsRepo.GetAccount(s.tenantID, code)
	if err != nil {
		return nil, err
	}
	if v := strings.TrimSpace(in.Name); v != "" {
		rec.Name = v
	}
	if v := strings.TrimSpace(in.Role); v != "" {
		rec.Role = v
	}
	if v := strings.TrimSpace(in.Mobile); v != "" {
		rec.Mobile = v
	}
	if in.Password != "" {
		rec.Password = in.Password
	}
	if in.Enabled != nil {
		rec.Enabled = *in.Enabled
	}
	if in.SortOrder != nil {
		rec.SortOrder = *in.SortOrder
	}
	if err := s.kdzsRepo.SaveAccount(rec); err != nil {
		return nil, err
	}
	items, err := s.ListAccountDetails()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Code == code {
			return &items[i], nil
		}
	}
	return nil, repo.ErrNotFound
}

func (s *SyncService) DeleteKdzsAccount(code string) error {
	if err := s.loadSettings(); err != nil {
		return err
	}
	if s.settings != nil && s.settings.DefaultAccountCode == code {
		return fmt.Errorf("不能删除默认账号，请先设置其他账号为默认")
	}
	if s.activeAccountID == code {
		return fmt.Errorf("不能删除当前使用中的账号，请先切换到其他账号")
	}
	return s.kdzsRepo.DeleteAccount(s.tenantID, code)
}

func (s *SyncService) SetDefaultKdzsAccount(code string) error {
	if _, ok := s.accountByCode(code); !ok {
		return fmt.Errorf("account %s not found", code)
	}
	settings, err := s.kdzsRepo.GetOrCreateSettings(s.tenantID, s.globalBaseURL)
	if err != nil {
		return err
	}
	settings.DefaultAccountCode = code
	s.settings = settings
	return s.kdzsRepo.SaveSettings(settings)
}
