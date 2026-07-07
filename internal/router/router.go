package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	adminmw "storesyncagent/admin/middleware"
	"storesyncagent/internal/config"
	"storesyncagent/internal/handler"
	jwtmgr "storesyncagent/internal/pkg/jwt"
)

func Setup(h *handler.Handler, cfg *config.Config, webDist string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(corsMiddleware(cfg))

	r.GET("/health", h.Health)

	v1 := r.Group("/api/v1")
	v1.GET("/health", h.Health)

	adminGroup := v1.Group("/admin")
	jwtMgr := jwtmgr.NewManager(cfg.Auth.JWTSecret)
	adminGroup.Use(adminmw.AdminAuth(&cfg.Auth, jwtMgr))
	{
		adminGroup.GET("/kdzs/status", h.LoginStatus)
		adminGroup.GET("/kdzs/accounts", h.ListAccounts)
		adminGroup.GET("/kdzs/account-details", h.ListKdzsAccountDetails)
		adminGroup.POST("/kdzs/accounts", h.CreateKdzsAccount)
		adminGroup.PUT("/kdzs/accounts/:id", h.UpdateKdzsAccount)
		adminGroup.DELETE("/kdzs/accounts/:id", h.DeleteKdzsAccount)
		adminGroup.POST("/kdzs/accounts/default", h.SetDefaultKdzsAccount)
		adminGroup.POST("/kdzs/accounts/switch", h.SwitchAccount)
		adminGroup.GET("/shops", h.ListShops)
		adminGroup.GET("/factories", h.ListFactories)
		adminGroup.GET("/orders", h.ListOrders)
		adminGroup.POST("/orders/decrypt", h.DecryptOrders)
		adminGroup.POST("/orders/agent-type", h.SetOrderAgentType)
		adminGroup.GET("/refunds", h.ListRefunds)
		adminGroup.GET("/refunds/stats", h.RefundStats)
		adminGroup.GET("/refunds/logistics", h.RefundLogistics)
		adminGroup.GET("/orders/lookup", h.LookupOrder)
		adminGroup.POST("/orders/lookup-tracking", h.LookupOrdersByTracking)
		adminGroup.GET("/return-exchanges", h.ListReturnExchanges)
		adminGroup.POST("/return-exchanges", h.CreateReturnExchange)
		adminGroup.PUT("/return-exchanges/:id", h.UpdateReturnExchange)
		adminGroup.DELETE("/return-exchanges/:id", h.DeleteReturnExchange)
		adminGroup.GET("/notifications", h.GetNotification)
		adminGroup.PUT("/notifications", h.SaveNotification)
		adminGroup.POST("/notifications/test", h.TestNotification)
		adminGroup.POST("/notifications/test-barcode", h.TestBarcodeNotification)
		adminGroup.POST("/notifications/run", h.RunNotification)
		adminGroup.POST("/notifications/reset-state", h.ResetNotificationState)
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
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
			return
		}
		c.File(indexHTML)
	})
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORS.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin == ""
		for _, o := range origins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
