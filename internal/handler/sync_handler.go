package handler

import (
	"net/http"
	"silversync-api/internal/service"

	"github.com/gin-gonic/gin"
)

type SyncHandler struct {
	downloaderService service.DownloaderService
}

func NewSyncHandler(ds service.DownloaderService) *SyncHandler {
	return &SyncHandler{
		downloaderService: ds,
	}
}

type SyncRequest struct {
	URL string `json:"url" binding:"required,url"`
}

func (h *SyncHandler) Sync(c *gin.Context) {
	var req SyncRequest

	// Validate JSON request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request. 'url' is required and must be a valid Spotify URL.",
		})
		return
	}

	// Trigger download in a Goroutine (Asynchronous)
	go h.downloaderService.Download(req.URL)

	// Return immediate response to prevent timeout
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Syncing started in background",
		"url":     req.URL,
		"status":  "PENDING",
	})
}
