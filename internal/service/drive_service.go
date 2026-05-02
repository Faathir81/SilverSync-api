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
	UploadFile(ctx context.Context, filePath string, originalFileName string) (string, error)
	GetStorageQuota(ctx context.Context) (*drive.AboutStorageQuota, error)
	DeleteFile(ctx context.Context, fileID string) error
}

type driveService struct {
	client   *drive.Service
	folderID string
}

func NewDriveService() (DriveService, error) {
	ctx := context.Background()

	credFile := os.Getenv("GDRIVE_CREDENTIALS_FILE")
	folderID := os.Getenv("GDRIVE_FOLDER_ID")

	if credFile == "" || folderID == "" {
		return nil, fmt.Errorf("GDRIVE_CREDENTIALS_FILE or GDRIVE_FOLDER_ID is missing")
	}

	client, err := drive.NewService(ctx, option.WithCredentialsFile(credFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %v", err)
	}

	return &driveService{
		client:   client,
		folderID: folderID,
	}, nil
}

func (s *driveService) UploadFile(ctx context.Context, filePath string, originalFileName string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for upload: %v", err)
	}
	defer file.Close()

	driveFile := &drive.File{
		Name:    originalFileName,
		Parents: []string{s.folderID},
	}

	log.Printf("[Drive] Uploading %s to Google Drive...\n", originalFileName)

	res, err := s.client.Files.Create(driveFile).Media(file).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %v", err)
	}

	log.Printf("[Drive] Successfully uploaded %s (File ID: %s)\n", originalFileName, res.Id)
	return res.Id, nil
}

func (s *driveService) GetStorageQuota(ctx context.Context) (*drive.AboutStorageQuota, error) {
	about, err := s.client.About.Get().Fields("storageQuota").Do()
	if err != nil {
		return nil, err
	}
	return about.StorageQuota, nil
}

func (s *driveService) DeleteFile(ctx context.Context, fileID string) error {
	log.Printf("[Drive] Deleting file from Drive: %s\n", fileID)
	return s.client.Files.Delete(fileID).Context(ctx).Do()
}
