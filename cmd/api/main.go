package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"storesyncagent/internal/config"
	"storesyncagent/internal/handler"
	"storesyncagent/internal/router"
	"storesyncagent/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	webDist := flag.String("web-dist", "", "static web root for local dev (empty = API only)")
	flag.Parse()

	absConfig, err := filepath.Abs(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := config.Load(absConfig)
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Kdzs.Validate(); err != nil {
		log.Fatal(err)
	}

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	svc, err := service.NewSyncService(cfg)
	if err != nil {
		log.Fatal(err)
	}
	h := handler.New(svc)
	engine := router.Setup(h, *webDist)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("StoreSyncAgent API listening on http://localhost%s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatal(err)
	}
}
