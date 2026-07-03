package kdzs

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

// LoginWithPassword performs password login (loginType=1).
// The web client sends MD5(password) in uppercase hex.
func (c *Client) LoginWithPassword(ctx context.Context, mobile, password string) (*LoginData, error) {
	mobile = strings.TrimSpace(mobile)
	password = strings.TrimSpace(password)
	if mobile == "" || password == "" {
		return nil, fmt.Errorf("mobile and password are required")
	}

	req := LoginRequest{
		Mobile:    mobile,
		Password:  hashPassword(password),
		LoginType: LoginTypePassword,
	}

	var resp APIResponse[LoginData]
	if err := c.post(ctx, "/factory/login/login/", req, &resp); err != nil {
		return nil, err
	}
	data, err := checkResult(&resp)
	if err != nil {
		return nil, err
	}
	if data.Token == "" {
		return nil, fmt.Errorf("login succeeded but token is empty")
	}
	c.token = data.Token
	return &data, nil
}

func hashPassword(password string) string {
	sum := md5.Sum([]byte(password))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}
