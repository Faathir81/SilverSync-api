package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"silversync-api/internal/config"
	"strings"

	"github.com/bogem/id3v2"
)

type DownloaderService interface {
	DownloadAudio(ctx context.Context, track *TrackMetadata) (string, error)
}

type downloaderService struct{}

func NewDownloaderService() DownloaderService {
	return &downloaderService{}
}

func sanitizeFileName(name string) string {
	// Simple sanitizer to remove invalid characters for windows/linux
	invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, char, "_")
	}
	return name
}

func (s *downloaderService) DownloadAudio(ctx context.Context, track *TrackMetadata) (string, error) {
	downloadDir := "downloads"
	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		err := os.Mkdir(downloadDir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create downloads directory: %v", err)
		}
	}

	safeArtist := sanitizeFileName(track.Artist)
	safeTitle := sanitizeFileName(track.Title)
	fileName := fmt.Sprintf("%s - %s.mp3", safeArtist, safeTitle)
	outputPath := filepath.Join(downloadDir, fileName)

	// Using yt-dlp to search and download the best audio format, converting to mp3
	searchQuery := fmt.Sprintf("ytsearch1:%s %s audio", track.Title, track.Artist)

	args := []string{
		searchQuery,
		"-x", // extract audio
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"-o", outputPath,
		"--sleep-interval", "3",
		"--max-sleep-interval", "8",
		"--no-playlist",
		"--extractor-arg", "youtube:skip=hls,dash",
	}

	// Check if cookies.txt exists in root directory to avoid Rate Limit 429
	if _, err := os.Stat("cookies.txt"); err == nil {
		args = append(args, "--cookies", "cookies.txt")
		log.Println("[Downloader] Using cookies.txt for authentication")
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	
	config.Logger.Infof("[Downloader] Executing yt-dlp for track: %s", track.Title)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("yt-dlp error: %v, output: %s", err, string(out))
	}

	config.Logger.Infof("[Downloader] Successfully downloaded: %s", outputPath)

	// Inject ID3 Tags
	config.Logger.Infof("[Downloader] Injecting ID3 tags for: %s", track.Title)
	if err := s.injectID3Tags(outputPath, track); err != nil {
		config.Logger.Warnf("[Downloader] Failed to inject ID3 tags: %v", err)
	}

	return outputPath, nil
}

func (s *downloaderService) injectID3Tags(filePath string, track *TrackMetadata) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("error opening mp3 file for tagging: %v", err)
	}
	defer tag.Close()

	tag.SetTitle(track.Title)
	tag.SetArtist(track.Artist)

	if track.AlbumArtURL != "" {
		resp, err := http.Get(track.AlbumArtURL)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				imgData, err := io.ReadAll(resp.Body)
				if err == nil {
					pic := id3v2.PictureFrame{
						Encoding:    id3v2.EncodingUTF8,
						MimeType:    "image/jpeg",
						PictureType: id3v2.PTFrontCover,
						Description: "Front cover",
						Picture:     imgData,
					}
					tag.AddAttachedPicture(pic)
				}
			}
		} else {
			config.Logger.Warnf("[Downloader] Failed to download album art: %v", err)
		}
	}

	if err = tag.Save(); err != nil {
		return fmt.Errorf("error saving id3 tags: %v", err)
	}
	return nil
}
