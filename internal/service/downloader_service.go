package service

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type DownloaderService interface {
	Download(spotifyURL string)
}

type downloaderService struct {
	driveService DriveService
}

func NewDownloaderService(ds DriveService) DownloaderService {
	return &downloaderService{
		driveService: ds,
	}
}

func (s *downloaderService) Download(spotifyURL string) {
	// 1. Ensure downloads directory exists
	downloadDir := "downloads"
	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		_ = os.Mkdir(downloadDir, 0755)
	}

	// 2. Prepare the spotDL command arguments
	args := []string{spotifyURL, "--output", downloadDir, "--format", "mp3"}

	// Add Spotify credentials if available in .env
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID != "" && clientSecret != "" {
		args = append(args, "--client-id", clientID, "--client-secret", clientSecret)
		log.Println("[Downloader] Using Spotify API credentials for this request.")
	}

	cmd := exec.Command("spotdl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("[Downloader] Starting download for: %s\n", spotifyURL)

	// 3. Execute the command
	err := cmd.Run()
	if err != nil {
		log.Printf("[Downloader] Error executing spotDL: %v\n", err)
		return
	}

	// 4. Find the downloaded file (spotDL names it based on metadata)
	files, err := filepath.Glob(filepath.Join(downloadDir, "*.mp3"))
	if err != nil || len(files) == 0 {
		log.Printf("[Downloader] Could not find downloaded file in %s\n", downloadDir)
		return
	}

	// For simplicity, we take the first .mp3 found in the folder
	// In production, we might want a more precise matching logic
	localFilePath := files[0]
	fileName := filepath.Base(localFilePath)

	// 5. Cleanup: Delete local file after this function finishes
	defer func() {
		log.Printf("[Cleanup] Removing temporary file: %s\n", localFilePath)
		_ = os.Remove(localFilePath)
	}()

	// 6. Upload to Google Drive
	driveID, err := s.driveService.UploadFile(localFilePath, fileName)
	if err != nil {
		log.Printf("[Downloader] Failed to upload to Drive: %v\n", err)
		return
	}

	log.Printf("[Downloader] Process completed. Drive ID: %s\n", driveID)
}
