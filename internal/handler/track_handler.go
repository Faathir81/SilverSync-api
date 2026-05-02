package handler

import (
	"net/http"
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
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
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
