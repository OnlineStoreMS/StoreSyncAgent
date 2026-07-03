package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type ReturnExchangeGoods struct {
	PicURL  string `json:"picUrl,omitempty"`
	SkuName string `json:"skuName,omitempty"`
}

type ReturnExchangeRecord struct {
	ID                 string `json:"id"`
	SeqNo              int    `json:"seqNo,omitempty"`
	BuyerNick          string  `json:"buyerNick,omitempty"`          // 客户昵称（手填）
	AfterSaleType      string  `json:"afterSaleType,omitempty"`
	ReturnTrackingNo   string  `json:"returnTrackingNo,omitempty"`
	Spec               string  `json:"spec,omitempty"`
	FeedbackTime       string  `json:"feedbackTime,omitempty"`
	SubmitTime         string  `json:"submitTime,omitempty"`
	OrderNo            string  `json:"orderNo,omitempty"`
	RecipientInfo         string                `json:"recipientInfo,omitempty"`      // 顾客提供的新收件地址（手填）
	ParsedRecipientInfo   string                `json:"parsedRecipientInfo,omitempty"` // 解析后的格式化地址
	OutboundTrackingNo string  `json:"outboundTrackingNo,omitempty"`
	Remark             string  `json:"remark,omitempty"`
	Platform           string  `json:"platform,omitempty"`
	SysTid             string  `json:"sysTid,omitempty"`
	ShopName              string                `json:"shopName,omitempty"`
	Goods                 []ReturnExchangeGoods `json:"goods,omitempty"`
	GoodsTitle            string                `json:"goodsTitle,omitempty"`
	OriginalRecipientInfo string                `json:"originalRecipientInfo,omitempty"`
	Payment            float64 `json:"payment,omitempty"`
	PayTime            string  `json:"payTime,omitempty"`
	StatusText         string  `json:"statusText,omitempty"`
	CreatedAt          string `json:"createdAt,omitempty"`
	UpdatedAt          string `json:"updatedAt,omitempty"`
}

type ReturnExchangeStore struct {
	path    string
	seedPath string
	mu      sync.Mutex
}

func NewReturnExchangeStore(path, seedPath string) (*ReturnExchangeStore, error) {
	if path == "" {
		return nil, fmt.Errorf("return exchange store path is empty")
	}
	s := &ReturnExchangeStore{path: path, seedPath: seedPath}
	if err := s.ensureFile(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ReturnExchangeStore) ensureFile() error {
	if _, err := os.Stat(s.path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if s.seedPath != "" {
		if raw, err := os.ReadFile(s.seedPath); err == nil && len(raw) > 0 {
			return os.WriteFile(s.path, raw, 0o644)
		}
	}
	return os.WriteFile(s.path, []byte("[]"), 0o644)
}

func (s *ReturnExchangeStore) List() ([]ReturnExchangeRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readLocked()
}

func (s *ReturnExchangeStore) Create(in ReturnExchangeRecord) (ReturnExchangeRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.readLocked()
	if err != nil {
		return ReturnExchangeRecord{}, err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	rec := in
	if rec.ID == "" {
		rec.ID = newID()
	}
	if rec.SeqNo == 0 {
		rec.SeqNo = nextSeqNo(items)
	}
	rec.CreatedAt = now
	rec.UpdatedAt = now
	items = append(items, rec)
	if err := s.writeLocked(items); err != nil {
		return ReturnExchangeRecord{}, err
	}
	return rec, nil
}

func (s *ReturnExchangeStore) Update(id string, in ReturnExchangeRecord) (ReturnExchangeRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.readLocked()
	if err != nil {
		return ReturnExchangeRecord{}, err
	}
	for i := range items {
		if items[i].ID != id {
			continue
		}
		updated := in
		updated.ID = id
		if updated.SeqNo == 0 {
			updated.SeqNo = items[i].SeqNo
		}
		updated.CreatedAt = items[i].CreatedAt
		updated.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
		items[i] = updated
		if err := s.writeLocked(items); err != nil {
			return ReturnExchangeRecord{}, err
		}
		return updated, nil
	}
	return ReturnExchangeRecord{}, fmt.Errorf("record %s not found", id)
}

func (s *ReturnExchangeStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.readLocked()
	if err != nil {
		return err
	}
	out := make([]ReturnExchangeRecord, 0, len(items))
	found := false
	for _, item := range items {
		if item.ID == id {
			found = true
			continue
		}
		out = append(out, item)
	}
	if !found {
		return fmt.Errorf("record %s not found", id)
	}
	return s.writeLocked(out)
}

func (s *ReturnExchangeStore) readLocked() ([]ReturnExchangeRecord, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []ReturnExchangeRecord{}, nil
	}
	var items []ReturnExchangeRecord
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SeqNo != items[j].SeqNo {
			return items[i].SeqNo < items[j].SeqNo
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return items, nil
}

func (s *ReturnExchangeStore) writeLocked(items []ReturnExchangeRecord) error {
	raw, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func nextSeqNo(items []ReturnExchangeRecord) int {
	max := 0
	for _, item := range items {
		if item.SeqNo > max {
			max = item.SeqNo
		}
	}
	return max + 1
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
