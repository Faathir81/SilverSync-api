package routes

import (
	"log"
	"net/http"
	"silversync-api/internal/config"
	"silversync-api/internal/handler"
	"silversync-api/internal/repository"
	"silversync-api/internal/service"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Initialize Repositories
	trackRepo := repository.NewTrackRepository(config.DB)
	syncLogRepo := repository.NewSyncLogRepository(config.DB)
	playlistRepo := repository.NewPlaylistRepository(config.DB)
	watchRepo := repository.NewWatchedPlaylistRepository(config.DB)

	// Initialize Services
	spotifyService, err := service.NewSpotifyService()
	if err != nil {
		log.Fatalf("Failed to initialize Spotify Service: %v", err)
	}
	driveService, err := service.NewDriveService()
	if err != nil {
		log.Fatalf("Failed to initialize Google Drive Service: %v", err)
	}
	downloaderService := service.NewDownloaderService()

	// Initialize Handlers
	syncHandler := handler.NewSyncHandler(spotifyService, downloaderService, driveService, trackRepo, syncLogRepo, watchRepo)
	trackHandler := handler.NewTrackHandler(trackRepo, driveService)
	playlistHandler := handler.NewPlaylistHandler(playlistRepo)

	// Health check / Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
			"status":  "SilverSync API is running",
		})
	})

	// API v1 Group
	v1 := r.Group("/api/v1")
	{
		// Sync & Smart Watcher Endpoints
		v1.POST("/sync", syncHandler.Sync)
		v1.GET("/sync/status/:id", syncHandler.Status)
		v1.GET("/sync/quota", syncHandler.GetDriveQuota)
		v1.POST("/sync/watch", syncHandler.AddWatch)
		v1.GET("/sync/watch", syncHandler.ListWatch)

		// Track Endpoints
		v1.GET("/tracks", trackHandler.GetTracks)
		v1.PATCH("/tracks/:id", trackHandler.UpdateTrack)
		v1.DELETE("/tracks/:id", trackHandler.DeleteTrack)
		v1.PATCH("/tracks/:id/favorite", trackHandler.ToggleFavorite)

		// Playlist Endpoints
		v1.POST("/playlists", playlistHandler.Create)
		v1.GET("/playlists", playlistHandler.GetAll)
		v1.GET("/playlists/:id", playlistHandler.GetByID)
		v1.POST("/playlists/:id/tracks/:trackId", playlistHandler.AddTrack)
		v1.DELETE("/playlists/:id/tracks/:trackId", playlistHandler.RemoveTrack)
	}

	return r
}
