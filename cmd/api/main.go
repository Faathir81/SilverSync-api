package main

import (
	"os"
	"silversync-api/internal/config"
	"silversync-api/internal/routes"

	"github.com/joho/godotenv"
)

func main() {
	// Initialize Logger
	config.InitLogger()

	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		config.Logger.Warn("Warning: .env file not found, using system environment variables")
	}

	// Connect to Database
	config.ConnectDatabase()

	// Initialize Router
	r := routes.SetupRouter()

	// Get port from env
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	config.Logger.Infof("Server starting on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		config.Logger.Fatal("Failed to start server:", err)
	}
}
