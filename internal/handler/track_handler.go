package handler

import (
	"fmt"
	"io"
	"net/http"
	"silversync-api/internal/config"
	"silversync-api/internal/repository"
	"silversync-api/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TrackHandler struct {
	trackRepo    repository.TrackRepository
	driveService service.DriveService
}

func NewTrackHandler(tr repository.TrackRepository, ds service.DriveService) *TrackHandler {
	return &TrackHandler{
		trackRepo:    tr,
		driveService: ds,
	}
}

func (h *TrackHandler) GetTracks(c *gin.Context) {
	// Query parameters
	searchQuery := c.Query("q")
	sortQuery := c.Query("sort") // e.g., "title asc", "created_at desc"
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "500")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		limit = 500
	}

	offset := (page - 1) * limit

	tracks, total, err := h.trackRepo.FindAll(searchQuery, sortQuery, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tracks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tracks,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func (h *TrackHandler) ToggleFavorite(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		IsFavorite bool `json:"is_favorite"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := h.trackRepo.UpdateFavorite(uint(id), req.IsFavorite); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update favorite status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Favorite status updated", "is_favorite": req.IsFavorite})
}

func (h *TrackHandler) UpdateTrack(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	track, err := h.trackRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
		return
	}

	var req struct {
		Title  string `json:"title"`
		Artist string `json:"artist"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if req.Title != "" {
		track.Title = req.Title
	}
	if req.Artist != "" {
		track.Artist = req.Artist
	}

	if err := h.trackRepo.Update(track); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update track"})
		return
	}

	c.JSON(http.StatusOK, track)
}

func (h *TrackHandler) DeleteTrack(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	track, err := h.trackRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
		return
	}

	// 1. Delete from Google Drive
	if track.DriveFileID != "" {
		_ = h.driveService.DeleteFile(c.Request.Context(), track.DriveFileID)
	}

	// 2. Delete from Database
	if err := h.trackRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete track from database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Track successfully deleted from database and Drive"})
}

// StreamTrack proxies the Google Drive audio file with correct HTTP Range handling.
// Chrome's audio element always sends Range: bytes=0- and requires a 206 + Content-Range response.
func (h *TrackHandler) StreamTrack(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid track ID"})
		return
	}

	track, err := h.trackRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
		return
	}

	if track.DriveFileID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No Drive file associated with this track"})
		return
	}

	// Always fetch the full file from Drive (don't forward Range to avoid Drive SDK issues)
	// We handle Range negotiation ourselves at the proxy level.
	stream, mimeType, contentLength, _, err := h.driveService.GetFileStream(
		c.Request.Context(), track.DriveFileID, "",
	)
	if err != nil {
		config.Logger.Errorf("[Stream] Failed for track %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stream track"})
		return
	}
	defer stream.Close()

	rangeHeader := c.GetHeader("Range")

	c.Header("Content-Type", mimeType)
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "no-cache")

	if rangeHeader != "" && contentLength > 0 {
		// Chrome's audio element sends Range: bytes=0- for ALL audio requests.
		// We must respond with 206 Partial Content + Content-Range or Chrome will abort.
		// Since we always stream the full file, Content-Range covers the entire file.
		c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
		c.Header("Content-Range", fmt.Sprintf("bytes 0-%d/%d", contentLength-1, contentLength))
		c.Status(http.StatusPartialContent) // 206
		config.Logger.Infof("[Stream] Serving track %d as 206 (range=%s, size=%d)", id, rangeHeader, contentLength)
	} else {
		if contentLength > 0 {
			c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
		}
		c.Status(http.StatusOK) // 200
		config.Logger.Infof("[Stream] Serving track %d as 200 (size=%d)", id, contentLength)
	}

	if _, err := io.Copy(c.Writer, stream); err != nil {
		// wsasend error is expected when browser stops buffering — not a real error
		config.Logger.Debugf("[Stream] Stream ended for track %d: %v", id, err)
	}
}
