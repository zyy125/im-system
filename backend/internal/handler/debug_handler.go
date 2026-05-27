package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/zyy125/im-system/internal/ws"
)

type DebugHandler struct {
	hub *ws.Hub
}

func NewDebugHandler(hub *ws.Hub) *DebugHandler {
	return &DebugHandler{hub: hub}
}

func (h *DebugHandler) HubSnapshot(c *gin.Context) {
	if h == nil || h.hub == nil {
		respondOK(c, gin.H{})
		return
	}
	respondOK(c, h.hub.Snapshot())
}
