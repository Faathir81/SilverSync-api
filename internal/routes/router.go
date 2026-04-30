package routes

import (
	"net/http"
	"silversync-api/internal/handler"
	"silversync-api/internal/service"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Initialize Services & Handlers
	driveService := service.NewDriveService()
	downloaderService := service.NewDownloaderService(driveService)
	syncHandler := handler.NewSyncHandler(downloaderService)

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
		v1.POST("/sync", syncHandler.Sync)
		
		v1.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "API v1 is accessible",
			})
		})
	}

	return r
}
