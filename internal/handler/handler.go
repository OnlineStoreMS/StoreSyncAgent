package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"storesyncagent/internal/service"
)

type Handler struct {
	svc *service.SyncService
}

func New(svc *service.SyncService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) LoginStatus(c *gin.Context) {
	data, err := h.svc.LoginStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) ListShops(c *gin.Context) {
	shops, err := h.svc.ListShops(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": shops, "total": len(shops)})
}

func (h *Handler) ListOrders(c *gin.Context) {
	var q service.OrderQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.ListOrders(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) DecryptOrders(c *gin.Context) {
	var req service.DecryptOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.DecryptOrders(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListAccounts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.svc.ListAccounts()})
}

func (h *Handler) SwitchAccount(c *gin.Context) {
	var req struct {
		AccountID string `json:"accountId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.SwitchAccount(c.Request.Context(), req.AccountID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListFactories(c *gin.Context) {
	var q service.FactoryQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.ListFactories(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) SetOrderAgentType(c *gin.Context) {
	var req service.SetOrderAgentTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.SetOrderAgentType(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListRefunds(c *gin.Context) {
	var q service.RefundQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if v := c.Query("enrichLogistics"); v == "false" {
		q.EnrichLogistics = false
	} else {
		q.EnrichLogistics = true
	}
	result, err := h.svc.ListRefunds(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) RefundStats(c *gin.Context) {
	var q service.RefundQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.GetRefundStats(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) RefundLogistics(c *gin.Context) {
	platform := c.Query("platform")
	sid := c.Query("sid")
	sidCode := c.Query("sidCode")
	if sid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sid is required"})
		return
	}
	result, err := h.svc.GetRefundLogistics(c.Request.Context(), platform, sid, sidCode)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
