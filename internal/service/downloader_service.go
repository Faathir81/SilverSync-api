package service

import (
	"log"
	"os"
	"os/exec"
)

type DownloaderService interface {
	Download(spotifyURL string)
}

type downloaderService struct{}

func NewDownloaderService() DownloaderService {
	return &downloaderService{}
}

func (s *downloaderService) Download(spotifyURL string) {
	// 1. Ensure downloads directory exists
	downloadDir := "downloads"
	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		err := os.Mkdir(downloadDir, 0755)
		if err != nil {
			log.Printf("[Downloader] Error creating directory: %v\n", err)
			return
		}
	}

	// 2. Prepare the spotDL command
	// Format: spotdl [url] --output [path] --format mp3
	cmd := exec.Command("spotdl", spotifyURL, "--output", downloadDir, "--format", "mp3")

	// 3. Capture output for logging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("[Downloader] Starting download for: %s\n", spotifyURL)

	// 4. Execute the command
	err := cmd.Run()
	if err != nil {
		log.Printf("[Downloader] Error executing spotDL: %v\n", err)
		return
	}

	log.Printf("[Downloader] Successfully downloaded: %s\n", spotifyURL)
}
