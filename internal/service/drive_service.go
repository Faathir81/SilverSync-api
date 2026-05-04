package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"silversync-api/internal/config"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const googleTokenFile = ".google_token"

// QuotaInfo is our custom quota response with accurate SilverSync folder size
type QuotaInfo struct {
	Limit           int64 `json:"limit"`
	Usage           int64 `json:"usage"`
	UsageInDrive    int64 `json:"usageInDrive"`
	SilversyncBytes int64 `json:"silversyncBytes"`
}

// FileMeta holds metadata about a Drive file needed for streaming
type FileMeta struct {
	Name     string
	MimeType string
	Size     int64
}

type DriveService interface {
	UploadFile(ctx context.Context, filePath string, originalFileName string) (string, error)
	GetStorageQuota(ctx context.Context) (*QuotaInfo, error)
	// GetFileMeta fetches name, mimeType, size from Drive
	GetFileMeta(ctx context.Context, fileID string) (*FileMeta, error)
	// GetFileStream proxies the Drive file stream; rangeHeader forwards the HTTP Range header for seek support
	GetFileStream(ctx context.Context, fileID string, rangeHeader string) (io.ReadCloser, string, int64, int, error)
	DeleteFile(ctx context.Context, fileID string) error
	IsAuthenticated() bool
	SetToken(token *oauth2.Token)
	GetOAuthConfig() *oauth2.Config
}

type driveService struct {
	mu       sync.Mutex
	oauthCfg *oauth2.Config
	client   *drive.Service
	token    *oauth2.Token
	folderID string
}

func NewDriveService() (DriveService, error) {
	clientID := os.Getenv("GDRIVE_CLIENT_ID")
	clientSecret := os.Getenv("GDRIVE_CLIENT_SECRET")
	redirectURI := os.Getenv("GDRIVE_REDIRECT_URI")
	folderID := os.Getenv("GDRIVE_FOLDER_ID")

	if clientID == "" || clientSecret == "" || folderID == "" {
		return nil, fmt.Errorf("GDRIVE_CLIENT_ID, GDRIVE_CLIENT_SECRET, or GDRIVE_FOLDER_ID is missing")
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{"https://www.googleapis.com/auth/drive.file"},
		Endpoint:     google.Endpoint,
	}

	svc := &driveService{
		oauthCfg: cfg,
		folderID: folderID,
	}

	// Try to load saved token from disk
	if token, err := loadGoogleToken(); err == nil {
		svc.token = token
		if err := svc.initClient(); err != nil {
			config.Logger.Warnf("[Drive] Failed to init client from saved token: %v", err)
		} else {
			config.Logger.Info("[Drive] ✅ Loaded saved token. Google Drive is ready.")
		}
	} else {
		config.Logger.Warn("[Drive] ⚠️ No saved token. Visit /auth/google/login to authenticate.")
	}

	return svc, nil
}

func (s *driveService) initClient() error {
	ctx := context.Background()
	httpClient := s.oauthCfg.Client(ctx, s.token)
	client, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return err
	}
	s.client = client
	return nil
}

func (s *driveService) IsAuthenticated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client != nil
}

func (s *driveService) GetOAuthConfig() *oauth2.Config {
	return s.oauthCfg
}

func (s *driveService) SetToken(token *oauth2.Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
	if err := saveGoogleToken(token); err != nil {
		config.Logger.Warnf("[Drive] Failed to save token: %v", err)
	}
	if err := s.initClient(); err != nil {
		config.Logger.Warnf("[Drive] Failed to init client after token set: %v", err)
	} else {
		config.Logger.Info("[Drive] ✅ Token saved and Google Drive client initialized.")
	}
}

func (s *driveService) UploadFile(ctx context.Context, filePath string, originalFileName string) (string, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()

	if client == nil {
		return "", fmt.Errorf("google drive not authenticated. please visit /auth/google/login")
	}

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

	res, err := client.Files.Create(driveFile).Media(file).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %v", err)
	}

	config.Logger.Infof("[Drive] Successfully uploaded %s (File ID: %s)", originalFileName, res.Id)
	return res.Id, nil
}

// GetStorageQuota returns quota info including the real SilverSync folder usage
func (s *driveService) GetStorageQuota(ctx context.Context) (*QuotaInfo, error) {
	s.mu.Lock()
	client := s.client
	folderID := s.folderID
	s.mu.Unlock()

	if client == nil {
		return nil, fmt.Errorf("google drive not authenticated")
	}

	// 1. Get overall account quota (limit, usage, usageInDrive)
	about, err := client.About.Get().Fields("storageQuota").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account quota: %v", err)
	}
	q := about.StorageQuota

	// 2. Calculate actual SilverSync folder size by summing all files inside it
	var silversyncBytes int64
	pageToken := ""
	for {
		call := client.Files.List().
			Q(fmt.Sprintf("'%s' in parents and trashed = false", folderID)).
			Fields("nextPageToken, files(size)").
			PageSize(1000).
			Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		fileList, err := call.Do()
		if err != nil {
			// If we can't list files, fall back to 0
			config.Logger.Warnf("[Drive] Could not list SilverSync folder files: %v", err)
			break
		}
		for _, f := range fileList.Files {
			silversyncBytes += f.Size
		}
		if fileList.NextPageToken == "" {
			break
		}
		pageToken = fileList.NextPageToken
	}

	return &QuotaInfo{
		Limit:           q.Limit,
		Usage:           q.Usage,
		UsageInDrive:    q.UsageInDrive,
		SilversyncBytes: silversyncBytes,
	}, nil
}

func (s *driveService) DeleteFile(ctx context.Context, fileID string) error {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()

	if client == nil {
		return fmt.Errorf("google drive not authenticated")
	}

	config.Logger.Infof("[Drive] Deleting file from Drive: %s", fileID)
	return client.Files.Delete(fileID).Context(ctx).Do()
}

// GetFileMeta returns metadata (name, mimeType, size) for a Drive file
func (s *driveService) GetFileMeta(ctx context.Context, fileID string) (*FileMeta, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()

	if client == nil {
		return nil, fmt.Errorf("google drive not authenticated")
	}

	file, err := client.Files.Get(fileID).Fields("name, mimeType, size").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %v", err)
	}

	mimeType := file.MimeType
	if mimeType == "" {
		mimeType = "audio/mpeg"
	}

	return &FileMeta{
		Name:     file.Name,
		MimeType: mimeType,
		Size:     file.Size,
	}, nil
}

// GetFileStream proxies the Drive file stream with optional Range support.
// rangeHeader: value of the client's Range header (e.g. "bytes=0-"), empty for full download.
// Returns: body, mimeType, contentLength (-1 if unknown), http status code, error
func (s *driveService) GetFileStream(ctx context.Context, fileID string, rangeHeader string) (io.ReadCloser, string, int64, int, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()

	if client == nil {
		return nil, "", 0, 0, fmt.Errorf("google drive not authenticated")
	}

	meta, err := s.GetFileMeta(ctx, fileID)
	if err != nil {
		return nil, "", 0, 0, err
	}

	// Build download call; inject Range header when present so Drive returns 206 Partial Content
	call := client.Files.Get(fileID)
	if rangeHeader != "" {
		call.Header().Set("Range", rangeHeader)
	}

	resp, err := call.Download()
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("failed to download: %v", err)
	}

	statusCode := resp.StatusCode
	contentLength := resp.ContentLength // -1 if unknown
	if contentLength <= 0 && rangeHeader == "" {
		contentLength = meta.Size
	}

	config.Logger.Infof("[Drive] Streaming %s (%s) status=%d range=%q", meta.Name, meta.MimeType, statusCode, rangeHeader)
	return resp.Body, meta.MimeType, contentLength, statusCode, nil
}

func saveGoogleToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(googleTokenFile, data, 0600)
}

func loadGoogleToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(googleTokenFile)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}
