package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"storesyncagent/internal/config"
	"storesyncagent/internal/kdzs"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	mobile := flag.String("mobile", "", "KDZS mobile (required)")
	password := flag.String("password", "", "KDZS password (required)")
	activeOnly := flag.Bool("active-only", true, "only show active bound shops")
	flag.Parse()

	if *mobile == "" || *password == "" {
		log.Fatal("usage: kdzs-test -mobile MOBILE -password PASSWORD [-config configs/config.yaml]")
	}

	absConfig, err := filepath.Abs(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := config.Load(absConfig)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	client := kdzs.NewClient(cfg.Kdzs.BaseURL)

	fmt.Println("=== 快递助手登录测试 ===")
	fmt.Printf("账号: %s\n", *mobile)

	loginData, err := client.LoginWithPassword(ctx, *mobile, *password)
	if err != nil {
		log.Fatalf("login failed: %v", err)
	}
	fmt.Printf("登录成功 userId=%s mobile=%s\n", loginData.UserID, loginData.Mobile)
	fmt.Printf("token: %s...\n", loginData.Token[:min(24, len(loginData.Token))])

	fmt.Println("\n=== 电商店铺列表 ===")
	var shops []kdzs.BindShop
	if *activeOnly {
		shops, err = client.ListActiveShops(ctx)
	} else {
		shops, err = client.ListBindShops(ctx)
	}
	if err != nil {
		log.Fatalf("list shops failed: %v", err)
	}

	type shopView struct {
		ID           int64  `json:"id"`
		Platform     string `json:"platform"`
		PlatformName string `json:"platformName"`
		MallUserID   string `json:"mallUserId"`
		MallUserName string `json:"mallUserName"`
		BindTime     string `json:"bindTime"`
		ExpireTime   string `json:"expireTime"`
		TokenValid   bool   `json:"tokenValid"`
	}
	views := make([]shopView, 0, len(shops))
	for _, shop := range shops {
		views = append(views, shopView{
			ID:           shop.ID,
			Platform:     shop.Platform,
			PlatformName: platformName(shop.Platform),
			MallUserID:   shop.MallUserID,
			MallUserName: shop.MallUserName,
			BindTime:     shop.BindTime,
			ExpireTime:   shop.ExpireTime,
			TokenValid:   shop.TokenValid,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(views); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n共 %d 个店铺\n", len(views))
}

func platformName(code string) string {
	switch code {
	case kdzs.PlatformDouyin:
		return "抖店"
	case kdzs.PlatformTaobao:
		return "淘宝"
	case kdzs.PlatformXHS:
		return "小红书"
	case kdzs.PlatformManual:
		return "手工单"
	default:
		return code
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
