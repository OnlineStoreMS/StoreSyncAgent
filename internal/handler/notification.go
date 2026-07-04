package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"storesyncagent/internal/store"
)

func (h *Handler) GetNotification(c *gin.Context) {
	view, err := h.svc.GetNotificationView()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) SaveNotification(c *gin.Context) {
	var req store.NotificationConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	view, err := h.svc.SaveNotificationConfig(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) TestNotification(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.TestNotification(c.Request.Context(), req.Text); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RunNotification(c *gin.Context) {
	result, err := h.svc.RunNotificationPoll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "result": result})
		return
	}
	c.JSON(http.StatusOK, result)
}
