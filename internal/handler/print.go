package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"storesyncagent/internal/pkg/response"
	"storesyncagent/internal/service"
)

func (h *Handler) ListElecAuth(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := svc.ListElecAuth(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ListExpressTemplates(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := svc.ListExpressTemplates(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) GetBatchPrintURL(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	platform := c.Query("platform")
	if platform == "" {
		response.Fail(c, http.StatusBadRequest, "platform is required")
		return
	}
	url, err := svc.GetBatchPrintURL(c.Request.Context(), platform)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"url": url, "platform": platform})
}

func (h *Handler) QueryPrintWaybills(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var req service.PrintWaybillQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := svc.QueryPrintWaybills(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) ListSharedExpressAccounts(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := svc.ListSharedExpressAccounts(c.Request.Context(), c.Query("platform"))
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}
