package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"silversync-api/internal/config"
	"silversync-api/internal/models"
	"silversync-api/internal/repository"
	"silversync-api/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SyncHandler struct {
	spotifyService    *service.SpotifyService
	downloaderService service.DownloaderService
	driveService      service.DriveService
	trackRepo         repository.TrackRepository
	syncLogRepo       repository.SyncLogRepository
	watchRepo         repository.WatchedPlaylistRepository
	workerPool        service.WorkerPool
}

func NewSyncHandler(ss *service.SpotifyService, ds service.DownloaderService, dr service.DriveService, tr repository.TrackRepository, slr repository.SyncLogRepository, wr repository.WatchedPlaylistRepository, wp service.WorkerPool) *SyncHandler {
	return &SyncHandler{
		spotifyService:    ss,
		downloaderService: ds,
		driveService:      dr,
		trackRepo:         tr,
		syncLogRepo:       slr,
		watchRepo:         wr,
		workerPool:        wp,
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

	// 1. Create SyncLog
	syncLog := models.SyncLog{
		SpotifyURL: req.URL,
		Status:     "PENDING",
		Message:    "Sync process initiated",
	}
	if err := h.syncLogRepo.Create(&syncLog); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate sync log"})
		return
	}

	// Submit to WorkerPool instead of naked goroutine
	h.workerPool.Submit(func(ctx context.Context) {
		id, isPlaylist, err := service.ExtractSpotifyID(req.URL)
		if err != nil {
			h.updateLog(syncLog.ID, "FAILED", fmt.Sprintf("Invalid Spotify URL: %v", err))
			return
		}

		if !isPlaylist {
			h.syncSingleTrack(ctx, id, syncLog.ID)
		} else {
			h.updateLog(syncLog.ID, "DOWNLOADING", "Fetching playlist items...")
			tracks, err := h.spotifyService.FetchPlaylistTracks(ctx, id)
			if err != nil {
				h.updateLog(syncLog.ID, "FAILED", fmt.Sprintf("Failed to fetch playlist tracks: %v", err))
				return
			}

			total := len(tracks)
			h.updateLog(syncLog.ID, "DOWNLOADING", fmt.Sprintf("Found %d tracks. Starting bulk sync...", total))

			for i, t := range tracks {
				h.updateLog(syncLog.ID, "DOWNLOADING", fmt.Sprintf("[%d/%d] Syncing: %s - %s", i+1, total, t.Artist, t.Title))
				h.syncSingleTrack(ctx, t.SpotifyID, syncLog.ID)
			}

			h.updateLog(syncLog.ID, "SUCCESS", fmt.Sprintf("Bulk sync complete for %d tracks", total))
		}
	})

	// Return immediate response to prevent timeout
	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Sync task queued",
		"sync_log_id": syncLog.ID,
		"url":         req.URL,
		"status":      "PENDING",
	})
}

// syncSingleTrack is a helper to process one track with retry logic
func (h *SyncHandler) syncSingleTrack(ctx context.Context, spotifyID string, logID uint) {
	// Check Duplicate
	existingTrack, err := h.trackRepo.FindBySpotifyID(spotifyID)
	if err == nil && existingTrack != nil {
		h.updateLog(logID, "SUCCESS", fmt.Sprintf("Track already exists. Drive ID: %s", existingTrack.DriveFileID))
		return
	}

	trackMeta, err := h.spotifyService.FetchTrackMetadata(ctx, spotifyID)
	if err != nil {
		h.updateLog(logID, "FAILED", fmt.Sprintf("Failed to fetch metadata for %s: %v", spotifyID, err))
		return
	}

	const maxRetries = 2
	var outputPath string
	for i := 0; i <= maxRetries; i++ {
		attemptMsg := ""
		if i > 0 {
			attemptMsg = fmt.Sprintf(" (Retry %d/%d)", i, maxRetries)
		}
		h.updateLog(logID, "DOWNLOADING", fmt.Sprintf("Downloading: %s - %s%s", trackMeta.Artist, trackMeta.Title, attemptMsg))

		outputPath, err = h.downloaderService.DownloadAudio(ctx, trackMeta)
		if err == nil {
			break
		}
		config.Logger.Errorf("[Sync] Download failed for %s: %v. Retrying...", trackMeta.Title, err)
	}

	if err != nil {
		h.updateLog(logID, "FAILED", fmt.Sprintf("Failed to download %s after retries: %v", trackMeta.Title, err))
		return
	}

	// Clean up local file
	defer os.Remove(outputPath)

	h.updateLog(logID, "UPLOADING", fmt.Sprintf("Uploading %s to Drive...", trackMeta.Title))
	originalFileName := filepath.Base(outputPath)
	fileID, err := h.driveService.UploadFile(ctx, outputPath, originalFileName)
	if err != nil {
		h.updateLog(logID, "FAILED", fmt.Sprintf("Failed to upload %s to Drive: %v", trackMeta.Title, err))
		return
	}

	// Save to Database
	newTrack := models.Track{
		SpotifyID:   trackMeta.SpotifyID,
		Title:       trackMeta.Title,
		Artist:      trackMeta.Artist,
		AlbumArtURL: trackMeta.AlbumArtURL,
		DriveFileID: fileID,
	}
	if err := h.trackRepo.Save(&newTrack); err != nil {
		h.updateLog(logID, "FAILED", fmt.Sprintf("Failed to save %s to DB: %v", trackMeta.Title, err))
		return
	}

	h.updateLog(logID, "SUCCESS", fmt.Sprintf("Sync complete for %s. Drive ID: %s", trackMeta.Title, fileID))
}

// Helper method
func (h *SyncHandler) updateLog(id uint, status string, message string) {
	config.Logger.Infof("[SyncLog %d] %s: %s", id, status, message)
	syncLog, err := h.syncLogRepo.FindByID(id)
	if err == nil {
		syncLog.Status = status
		syncLog.Message = message
		_ = h.syncLogRepo.Update(syncLog)
	}
}

func (h *SyncHandler) Status(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sync log ID"})
		return
	}

	syncLog, err := h.syncLogRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sync log not found"})
		return
	}

	c.JSON(http.StatusOK, syncLog)
}

func (h *SyncHandler) GetDriveQuota(c *gin.Context) {
	quota, err := h.driveService.GetStorageQuota(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch drive quota"})
		return
	}
	c.JSON(http.StatusOK, quota)
}

func (h *SyncHandler) AddWatch(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required,url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	id, isPlaylist, err := service.ExtractSpotifyID(req.URL)
	if err != nil || !isPlaylist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only Spotify Playlist URLs are supported for Smart Watcher"})
		return
	}

	wp := models.WatchedPlaylist{
		SpotifyID: id,
		Name:      "Checking...", // Will be updated by watcher later
	}

	if err := h.watchRepo.Create(&wp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to watch list"})
		return
	}

	c.JSON(http.StatusCreated, wp)
}

func (h *SyncHandler) ListWatch(c *gin.Context) {
	watches, err := h.watchRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watch list"})
		return
	}
	c.JSON(http.StatusOK, watches)
}
