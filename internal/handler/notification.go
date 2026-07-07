package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"storesyncagent/internal/pkg/response"
	"storesyncagent/internal/store"
)

func (h *Handler) GetNotification(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, err := svc.GetNotificationView()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, view)
}

func (h *Handler) SaveNotification(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var req store.NotificationConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, err := svc.SaveNotificationConfig(req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	response.OK(c, view)
}

func (h *Handler) TestNotification(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := svc.TestNotification(c.Request.Context(), req.Text); err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *Handler) TestBarcodeNotification(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := svc.TestBarcodeNotification(c.Request.Context()); err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *Handler) RunNotification(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := svc.RunNotificationPoll(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusBadGateway, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) ResetNotificationState(c *gin.Context) {
	svc, err := h.svc(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	view, cleared, err := svc.ResetNotificationState()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(c, gin.H{"cleared": cleared, "view": view})
}
