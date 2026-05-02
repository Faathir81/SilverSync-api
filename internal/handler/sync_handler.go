package handler

import (
	"context"
	"log"
	"net/http"
	"silversync-api/internal/service"

	"github.com/gin-gonic/gin"
)

type SyncHandler struct {
	spotifyService    *service.SpotifyService
	downloaderService service.DownloaderService
}

func NewSyncHandler(ss *service.SpotifyService, ds service.DownloaderService) *SyncHandler {
	return &SyncHandler{
		spotifyService:    ss,
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
	go func(url string) {
		// Temporary basic flow (will be improved with error handling & worker pool in later phases)
		importCtx := context.Background()
		id, isPlaylist, err := service.ExtractSpotifyID(url)
		if err != nil {
			log.Printf("Invalid Spotify URL: %v\n", err)
			return
		}

		if !isPlaylist {
			trackMeta, err := h.spotifyService.FetchTrackMetadata(importCtx, id)
			if err != nil {
				log.Printf("Failed to fetch metadata: %v\n", err)
				return
			}
			_, err = h.downloaderService.DownloadAudio(importCtx, trackMeta)
			if err != nil {
				log.Printf("Failed to download audio: %v\n", err)
			}
		} else {
			log.Println("Playlist download not fully implemented in this phase yet")
		}
	}(req.URL)

	// Return immediate response to prevent timeout
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Syncing started in background",
		"url":     req.URL,
		"status":  "PENDING",
	})
}
