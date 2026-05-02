package service

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2/clientcredentials"
)

// SpotifyService is responsible for interacting with the Spotify Web API.
type SpotifyService struct {
	Client *spotify.Client
}

// TrackMetadata holds the required track information.
type TrackMetadata struct {
	SpotifyID   string
	Title       string
	Artist      string
	AlbumArtURL string
}

// NewSpotifyService initializes and authenticates the Spotify client.
func NewSpotifyService() (*SpotifyService, error) {
	ctx := context.Background()

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_ID or SPOTIFY_CLIENT_SECRET is not set in environment variables")
	}

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://accounts.spotify.com/api/token",
	}

	httpClient := config.Client(ctx)
	client := spotify.New(httpClient)

	return &SpotifyService{Client: client}, nil
}

// ExtractSpotifyID parses a Spotify URL and returns the ID and its type (track/playlist).
func ExtractSpotifyID(url string) (id string, isPlaylist bool, err error) {
	// e.g., https://open.spotify.com/track/4PTG3Z6ehGkBFwjybzWkR8?si=...
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

	var artistNames []string
	for _, artist := range track.Artists {
		artistNames = append(artistNames, artist.Name)
	}

	var coverArt string
	if len(track.Album.Images) > 0 {
		coverArt = track.Album.Images[0].URL // The first image is the highest resolution
	}

	return &TrackMetadata{
		SpotifyID:   string(track.ID),
		Title:       track.Name,
		Artist:      strings.Join(artistNames, ", "),
		AlbumArtURL: coverArt,
	}, nil
}

// FetchPlaylistTracks retrieves all tracks from a playlist by its Spotify ID.
func (s *SpotifyService) FetchPlaylistTracks(ctx context.Context, playlistID string) ([]TrackMetadata, error) {
	playlistItems, err := s.Client.GetPlaylistItems(ctx, spotify.ID(playlistID))
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist items: %v", err)
	}

	var tracks []TrackMetadata
	for _, item := range playlistItems.Items {
		if item.Track.Track != nil {
			var artistNames []string
			for _, artist := range item.Track.Track.Artists {
				artistNames = append(artistNames, artist.Name)
			}

			var coverArt string
			if len(item.Track.Track.Album.Images) > 0 {
				coverArt = item.Track.Track.Album.Images[0].URL
			}

			tracks = append(tracks, TrackMetadata{
				SpotifyID:   string(item.Track.Track.ID),
				Title:       item.Track.Track.Name,
				Artist:      strings.Join(artistNames, ", "),
				AlbumArtURL: coverArt,
			})
		}
	}

	// TODO: Handle pagination (playlistItems.Next) if a playlist has more than 100 tracks.
	return tracks, nil
}
