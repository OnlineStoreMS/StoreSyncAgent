package kdzs

import (
	"context"
	"encoding/json"
)

func (s *Session) FetchDecryptMetaBySysTids(ctx context.Context, platform, tradeStatus string, sysTids []string) (map[string]TradeDecryptMeta, error) {
	pkgs, err := s.FetchTradeDetails(ctx, platform, tradeStatus, sysTids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]TradeDecryptMeta, len(pkgs))
	for _, raw := range pkgs {
		var pkg map[string]any
		if err := json.Unmarshal(raw, &pkg); err != nil {
			continue
		}
		meta := ParseDecryptMeta(pkg)
		if sysTid := asString(pkg["sysTid"]); sysTid != "" {
			out[sysTid] = meta
		}
	}
	return out, nil
}

func ApplyDecryptedReceiver(item *TradeListItem, decrypted *DecryptedReceiver) {
	if item == nil || decrypted == nil {
		return
	}
	item.ReceiverName = decrypted.ReceiverName
	item.ReceiverMobile = decrypted.ReceiverMobile
	item.ReceiverAddress = decrypted.ReceiverAddress
	item.FormattedReceiver = decrypted.FormattedText
	item.Decrypted = true
}
