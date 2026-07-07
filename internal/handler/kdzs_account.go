package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"storesyncagent/internal/pkg/authcontext"
	"storesyncagent/internal/pkg/response"
	"storesyncagent/internal/service"
)

func (h *Handler) ListKdzsAccountDetails(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := svc.ListAccountDetails()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) CreateKdzsAccount(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var req service.KdzsAccountInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := svc.CreateKdzsAccount(req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.mgr.InvalidateTenant(authcontext.TenantID(c))
	response.Created(c, item)
}

func (h *Handler) UpdateKdzsAccount(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	code := c.Param("id")
	var req service.KdzsAccountInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := svc.UpdateKdzsAccount(code, req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.mgr.InvalidateTenant(authcontext.TenantID(c))
	response.OK(c, item)
}

func (h *Handler) DeleteKdzsAccount(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	code := c.Param("id")
	if err := svc.DeleteKdzsAccount(code); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.mgr.InvalidateTenant(authcontext.TenantID(c))
	response.OK(c, gin.H{"ok": true})
}

func (h *Handler) SetDefaultKdzsAccount(c *gin.Context) {
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
	if err := svc.SetDefaultKdzsAccount(req.AccountID); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.mgr.InvalidateTenant(authcontext.TenantID(c))
	response.OK(c, gin.H{"ok": true})
}
