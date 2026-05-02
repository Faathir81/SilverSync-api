package handler

import (
	"net/http"
	"silversync-api/internal/models"
	"silversync-api/internal/repository"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PlaylistHandler struct {
	repo repository.PlaylistRepository
}

func NewPlaylistHandler(repo repository.PlaylistRepository) *PlaylistHandler {
	return &PlaylistHandler{repo: repo}
}

func (h *PlaylistHandler) Create(c *gin.Context) {
	var playlist models.Playlist
	if err := c.ShouldBindJSON(&playlist); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(&playlist); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create playlist"})
		return
	}

	c.JSON(http.StatusCreated, playlist)
}

func (h *PlaylistHandler) GetAll(c *gin.Context) {
	playlists, err := h.repo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlists"})
		return
	}
	c.JSON(http.StatusOK, playlists)
}

func (h *PlaylistHandler) GetByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	playlist, err := h.repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}
	c.JSON(http.StatusOK, playlist)
}

func (h *PlaylistHandler) AddTrack(c *gin.Context) {
	playlistID, _ := strconv.Atoi(c.Param("id"))
	trackID, _ := strconv.Atoi(c.Param("trackId"))

	if err := h.repo.AddTrack(uint(playlistID), uint(trackID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add track to playlist"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Track added to playlist"})
}

func (h *PlaylistHandler) RemoveTrack(c *gin.Context) {
	playlistID, _ := strconv.Atoi(c.Param("id"))
	trackID, _ := strconv.Atoi(c.Param("trackId"))

	if err := h.repo.RemoveTrack(uint(playlistID), uint(trackID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove track from playlist"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Track removed from playlist"})
}
