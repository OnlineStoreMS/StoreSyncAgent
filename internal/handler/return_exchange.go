package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"storesyncagent/internal/pkg/response"
	"storesyncagent/internal/store"
)

func (h *Handler) LookupOrder(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tid := c.Query("orderNo")
	if tid == "" {
		tid = c.Query("tid")
	}
	trackingNo := c.Query("trackingNo")
	platform := c.Query("platform")
	if trackingNo != "" {
		recordType := c.Query("recordType")
		result, err := svc.LookupOrderByTrackingNo(c.Request.Context(), trackingNo, recordType)
		if err != nil {
			response.Fail(c, http.StatusBadGateway, err.Error())
			return
		}
		response.OK(c, result)
		return
	}
	result, err := svc.LookupOrderByTid(c.Request.Context(), platform, tid)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) LookupOrdersByTracking(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		TrackingNos []string `json:"trackingNos" binding:"required"`
		RecordType  string   `json:"recordType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := svc.LookupOrdersByTrackingNos(c.Request.Context(), req.TrackingNos, req.RecordType)
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"items": result})
}

func (h *Handler) ListReturnExchanges(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := svc.ListReturnExchanges()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) CreateReturnExchange(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var req store.ReturnExchangeRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := svc.CreateReturnExchange(req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Created(c, item)
}

func (h *Handler) UpdateReturnExchange(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	id := c.Param("id")
	var req store.ReturnExchangeRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := svc.UpdateReturnExchange(id, req)
	if err != nil {
		response.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *Handler) DeleteReturnExchange(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	id := c.Param("id")
	if err := svc.DeleteReturnExchange(id); err != nil {
		response.Fail(c, http.StatusNotFound, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}
