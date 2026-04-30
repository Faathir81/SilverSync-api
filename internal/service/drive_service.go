package service

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type DriveService interface {
	UploadFile(filePath string, filename string) (string, error)
}

type driveService struct {
	service  *drive.Service
	folderID string
}

func NewDriveService() DriveService {
	ctx := context.Background()
	credentialsFile := os.Getenv("GDRIVE_CREDENTIALS_FILE")
	folderID := os.Getenv("GDRIVE_FOLDER_ID")

	if credentialsFile == "" || folderID == "" {
		log.Fatal("GDRIVE_CREDENTIALS_FILE or GDRIVE_FOLDER_ID is not set in environment variables")
	}

	srv, err := drive.NewService(ctx, 
		option.WithCredentialsFile(credentialsFile),
		option.WithScopes(drive.DriveFileScope),
	)
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	return &driveService{
		service:  srv,
		folderID: folderID,
	}
}

func (s *driveService) UploadFile(filePath string, filename string) (string, error) {
	// Open the local file
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("could not open file: %v", err)
	}
	defer f.Close()

	// Prepare Drive file metadata
	driveFile := &drive.File{
		Name:    filename,
		Parents: []string{s.folderID},
	}

	// Execute Upload
	log.Printf("[Drive] Uploading %s to Google Drive...\n", filename)
	res, err := s.service.Files.Create(driveFile).Media(f).Do()
	if err != nil {
		return "", fmt.Errorf("could not create file in drive: %v", err)
	}

	log.Printf("[Drive] Upload successful. File ID: %s\n", res.Id)
	return res.Id, nil
}
