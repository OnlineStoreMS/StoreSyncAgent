package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"storesyncagent/internal/handler"
)

func Setup(h *handler.Handler, webDist string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(corsMiddleware())

	api := r.Group("/api/v1")
	{
		api.GET("/health", h.Health)
		api.GET("/kdzs/status", h.LoginStatus)
		api.GET("/kdzs/accounts", h.ListAccounts)
		api.POST("/kdzs/accounts/switch", h.SwitchAccount)
		api.GET("/shops", h.ListShops)
		api.GET("/factories", h.ListFactories)
		api.GET("/orders", h.ListOrders)
		api.POST("/orders/decrypt", h.DecryptOrders)
		api.POST("/orders/agent-type", h.SetOrderAgentType)
		api.GET("/refunds", h.ListRefunds)
		api.GET("/refunds/stats", h.RefundStats)
		api.GET("/refunds/logistics", h.RefundLogistics)
		api.GET("/orders/lookup", h.LookupOrder)
		api.GET("/return-exchanges", h.ListReturnExchanges)
		api.POST("/return-exchanges", h.CreateReturnExchange)
		api.PUT("/return-exchanges/:id", h.UpdateReturnExchange)
		api.DELETE("/return-exchanges/:id", h.DeleteReturnExchange)
		api.GET("/notifications", h.GetNotification)
		api.PUT("/notifications", h.SaveNotification)
		api.POST("/notifications/test", h.TestNotification)
		api.POST("/notifications/test-barcode", h.TestBarcodeNotification)
		api.POST("/notifications/run", h.RunNotification)
		api.POST("/notifications/reset-state", h.ResetNotificationState)
	}

	mountWebUI(r, webDist)
	return r
}

func mountWebUI(r *gin.Engine, webDist string) {
	webDist = strings.TrimSpace(webDist)
	if webDist == "" {
		return
	}
	indexHTML := filepath.Join(webDist, "index.html")
	if _, err := os.Stat(indexHTML); err != nil {
		return
	}
	assetsDir := filepath.Join(webDist, "assets")
	if info, err := os.Stat(assetsDir); err == nil && info.IsDir() {
		r.Static("/assets", assetsDir)
	}
	r.GET("/", func(c *gin.Context) {
		c.File(indexHTML)
	})
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.File(indexHTML)
	})
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
