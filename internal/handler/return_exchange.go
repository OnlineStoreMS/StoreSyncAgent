package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"storesyncagent/internal/store"
)

func (h *Handler) LookupOrder(c *gin.Context) {
	tid := c.Query("orderNo")
	if tid == "" {
		tid = c.Query("tid")
	}
	platform := c.Query("platform")
	result, err := h.svc.LookupOrderByTid(c.Request.Context(), platform, tid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListReturnExchanges(c *gin.Context) {
	items, err := h.svc.ListReturnExchanges()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) CreateReturnExchange(c *gin.Context) {
	var req store.ReturnExchangeRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.CreateReturnExchange(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) UpdateReturnExchange(c *gin.Context) {
	id := c.Param("id")
	var req store.ReturnExchangeRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.UpdateReturnExchange(id, req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) DeleteReturnExchange(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteReturnExchange(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
