package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) TriggerSyncHandler(c *gin.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in manual sync trigger: %v", r)
			}
		}()
		h.Worker.SyncAll()
	}()

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Sync triggered successfully",
	})
}
