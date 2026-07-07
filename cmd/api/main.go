package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"storesyncagent/internal/config"
	"storesyncagent/internal/database"
	"storesyncagent/internal/handler"
	"storesyncagent/internal/repo"
	"storesyncagent/internal/router"
	"storesyncagent/internal/scheduler"
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

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatal(err)
	}
	if cfg.Database.SeedFromConfig {
		database.SeedLegacyAccounts(db, cfg)
	}
	log.Printf("database connected: driver=%s", cfg.Database.Driver)

	kdzsRepo := repo.NewKdzs(db)
	mgr := service.NewManager(cfg, kdzsRepo)
	h := handler.New(mgr)
	notifyScheduler := scheduler.NewNotificationScheduler(mgr)
	notifyScheduler.Start()
	engine := router.Setup(h, cfg, *webDist)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("StoreSyncAgent API listening on http://localhost%s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatal(err)
	}
}
