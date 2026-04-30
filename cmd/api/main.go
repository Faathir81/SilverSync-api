package main

import (
	"log"
	"os"
	"silversync-api/internal/config"
	"silversync-api/internal/routes"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
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
	log.Printf("Server starting on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
