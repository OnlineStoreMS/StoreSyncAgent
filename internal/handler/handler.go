package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"storesyncagent/internal/pkg/authcontext"
	"storesyncagent/internal/pkg/response"
	"storesyncagent/internal/service"
)

type Handler struct {
	mgr *service.Manager
}

func New(mgr *service.Manager) *Handler {
	return &Handler{mgr: mgr}
}

func (h *Handler) svc(c *gin.Context) (*service.SyncService, error) {
	return h.mgr.ForTenant(authcontext.TenantID(c))
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "storesyncagent"})
}

func (h *Handler) LoginStatus(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := svc.LoginStatus(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, data)
}

func (h *Handler) ListShops(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	shops, err := svc.ListShops(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"items": shops, "total": len(shops)})
}

func (h *Handler) ListOrders(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var q service.OrderQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := svc.ListOrders(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) DecryptOrders(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var req service.DecryptOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := svc.DecryptOrders(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListAccounts(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{"items": svc.ListAccounts()})
}

func (h *Handler) SwitchAccount(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		AccountID string `json:"accountId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := svc.SwitchAccount(c.Request.Context(), req.AccountID)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListFactories(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var q service.FactoryQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := svc.ListFactories(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) SetOrderAgentType(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var req service.SetOrderAgentTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := svc.SetOrderAgentType(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListRefunds(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var q service.RefundQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if v := c.Query("enrichLogistics"); v == "false" {
		q.EnrichLogistics = false
	} else {
		q.EnrichLogistics = true
	}
	result, err := svc.ListRefunds(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) RefundStats(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var q service.RefundQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := svc.GetRefundStats(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) RefundLogistics(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	platform := c.Query("platform")
	sid := c.Query("sid")
	sidCode := c.Query("sidCode")
	if sid == "" {
		response.Fail(c, http.StatusBadRequest, "sid is required")
		return
	}
	result, err := svc.GetRefundLogistics(c.Request.Context(), platform, sid, sidCode)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}
