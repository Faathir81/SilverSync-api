package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const tokenFile = ".spotify_token"

// SpotifyService is responsible for interacting with the Spotify Web API.
type SpotifyService struct {
	Client        *spotify.Client
	Authenticator *spotifyauth.Authenticator
	token         *oauth2.Token
	mu            sync.Mutex
}

// TrackMetadata holds the required track information.
type TrackMetadata struct {
	SpotifyID   string
	Title       string
	Artist      string
	AlbumArtURL string
}

// NewSpotifyService initializes the Spotify authenticator with OAuth2.
// The Spotify client will be nil until the user completes OAuth at GET /auth/login.
func NewSpotifyService() (*SpotifyService, error) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_ID or SPOTIFY_CLIENT_SECRET is not set")
	}
	if redirectURI == "" {
		redirectURI = "http://localhost:8080/auth/callback"
	}

	auth := spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
			spotifyauth.ScopeUserLibraryRead,
		),
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
	)

	svc := &SpotifyService{
		Authenticator: auth,
	}

	// Try to restore a saved token from a previous session
	if token, err := loadToken(); err == nil {
		ctx := context.Background()
		svc.Client = spotify.New(auth.Client(ctx, token))
		svc.token = token
		fmt.Println("[Spotify] ✅ Loaded saved token. Service is ready.")
	} else {
		fmt.Println("[Spotify] ⚠️  No saved token found.")
		fmt.Println("[Spotify]    Please open your browser and go to: GET http://localhost:8080/auth/login")
	}

	return svc, nil
}

// IsAuthenticated returns true if the service has a valid Spotify client.
func (s *SpotifyService) IsAuthenticated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Client != nil
}

// SetToken sets the OAuth token and creates the Spotify client. Called after successful OAuth callback.
func (s *SpotifyService) SetToken(token *oauth2.Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx := context.Background()
	s.token = token
	s.Client = spotify.New(s.Authenticator.Client(ctx, token))
	if err := saveToken(token); err != nil {
		fmt.Printf("[Spotify] Warning: could not save token: %v\n", err)
	}
	fmt.Println("[Spotify] ✅ Token saved and Spotify client initialized.")
}

// ExtractSpotifyID parses a Spotify URL and returns the ID and its type.
func ExtractSpotifyID(url string) (id string, isPlaylist bool, err error) {
	trackRe := regexp.MustCompile(`track/([a-zA-Z0-9]+)`)
	playlistRe := regexp.MustCompile(`playlist/([a-zA-Z0-9]+)`)

	if matches := trackRe.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1], false, nil
	}
	if matches := playlistRe.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1], true, nil
	}
	return "", false, fmt.Errorf("invalid or unsupported Spotify URL")
}

// FetchTrackMetadata retrieves metadata for a single track by its Spotify ID.
func (s *SpotifyService) FetchTrackMetadata(ctx context.Context, trackID string) (*TrackMetadata, error) {
	track, err := s.Client.GetTrack(ctx, spotify.ID(trackID))
	if err != nil {
		return nil, fmt.Errorf("failed to get track metadata: %v", err)
	}
	meta := s.mapTrack(track)
	return &meta, nil
}

// FetchPlaylistTracks retrieves all tracks from a playlist by its Spotify ID using a raw HTTP request.
// This bypasses the zmb3/spotify library which has a bug calling the deprecated /tracks endpoint.
func (s *SpotifyService) FetchPlaylistTracks(ctx context.Context, playlistID string) ([]TrackMetadata, error) {
	s.mu.Lock()
	authStatus := s.Client != nil
	token := s.token
	s.mu.Unlock()

	fmt.Printf("[Spotify] Fetching tracks for playlist: %s (Authenticated: %v)\n", playlistID, authStatus)

	if !authStatus || token == nil {
		return nil, fmt.Errorf("spotify client is not authenticated. please visit /auth/login")
	}

	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/items", playlistID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spotify API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// DEBUG: Read raw body to inspect structure
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	// Print first 600 chars to see the actual JSON structure
	preview := string(bodyBytes)
	if len(preview) > 600 {
		preview = preview[:600]
	}
	fmt.Printf("[Spotify DEBUG] Raw response preview:\n%s\n...\n", preview)

	// CONFIRMED structure from debug:
	// Each playlist item has an "item" wrapper.
	// Inside "item": "track" = boolean flag (true/false), NOT an object.
	// All actual track data (id, name, artists, album) lives directly on "item".
	type spotifyImage struct {
		URL string `json:"url"`
	}
	type spotifyArtist struct {
		Name string `json:"name"`
	}
	type spotifyAlbum struct {
		Images []spotifyImage `json:"images"`
	}
	type spotifyItemData struct {
		IsTrack bool            `json:"track"`   // boolean flag
		IsEpisode bool          `json:"episode"` // boolean flag
		Type    string          `json:"type"`    // "track" or "episode"
		ID      string          `json:"id"`
		Name    string          `json:"name"`
		Artists []spotifyArtist `json:"artists"`
		Album   spotifyAlbum    `json:"album"`
	}
	type spotifyItem struct {
		IsLocal bool            `json:"is_local"`
		Item    spotifyItemData `json:"item"`
	}
	var page struct {
		Items []spotifyItem `json:"items"`
		Total int           `json:"total"`
	}

	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&page); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	fmt.Printf("[Spotify] Successfully parsed %d items, total in playlist: %d\n", len(page.Items), page.Total)

	var tracks []TrackMetadata
	for _, item := range page.Items {
		d := item.Item
		// Skip episodes, local files, or items without a valid ID
		if item.IsLocal || !d.IsTrack || d.ID == "" {
			continue
		}

		var artistNames []string
		for _, a := range d.Artists {
			artistNames = append(artistNames, a.Name)
		}

		var coverArt string
		if len(d.Album.Images) > 0 {
			coverArt = d.Album.Images[0].URL
		}

		tracks = append(tracks, TrackMetadata{
			SpotifyID:   d.ID,
			Title:       d.Name,
			Artist:      strings.Join(artistNames, ", "),
			AlbumArtURL: coverArt,
		})
	}

	fmt.Printf("[Spotify] Mapped %d valid tracks\n", len(tracks))
	return tracks, nil
}

// mapTrack converts a spotify.FullTrack to TrackMetadata.
func (s *SpotifyService) mapTrack(track *spotify.FullTrack) TrackMetadata {
	var artistNames []string
	for _, artist := range track.Artists {
		artistNames = append(artistNames, artist.Name)
	}

	var coverArt string
	if len(track.Album.Images) > 0 {
		coverArt = track.Album.Images[0].URL
	}

	return TrackMetadata{
		SpotifyID:   string(track.ID),
		Title:       track.Name,
		Artist:      strings.Join(artistNames, ", "),
		AlbumArtURL: coverArt,
	}
}

// saveToken persists the OAuth token to a local file.
func saveToken(token *oauth2.Token) error {
	lines := []string{
		token.AccessToken,
		token.RefreshToken,
		token.TokenType,
		token.Expiry.Format(time.RFC3339),
	}
	return os.WriteFile(tokenFile, []byte(strings.Join(lines, "\n")), 0600)
}

// loadToken reads the persisted OAuth token from file.
func loadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 4 {
		return nil, fmt.Errorf("invalid token file format")
	}

	expiry, err := time.Parse(time.RFC3339, lines[3])
	if err != nil {
		return nil, fmt.Errorf("failed to parse token expiry: %v", err)
	}

	return &oauth2.Token{
		AccessToken:  lines[0],
		RefreshToken: lines[1],
		TokenType:    lines[2],
		Expiry:       expiry,
	}, nil
}
