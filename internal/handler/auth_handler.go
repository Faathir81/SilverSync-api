package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"silversync-api/internal/config"
	"silversync-api/internal/service"
)

// AuthHandler handles Spotify OAuth2 authentication.
type AuthHandler struct {
	SpotifyService *service.SpotifyService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(spotifyService *service.SpotifyService) *AuthHandler {
	return &AuthHandler{SpotifyService: spotifyService}
}

const oauthStateString = "silversync-secure-state"

// Login redirects the user to Spotify's authorization page.
// GET /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	url := h.SpotifyService.Authenticator.AuthURL(oauthStateString)
	config.Logger.Infof("Redirecting user to Spotify OAuth: %s", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// Callback handles the redirect from Spotify after user grants access.
// GET /auth/callback
func (h *AuthHandler) Callback(c *gin.Context) {
	// Validate the state to prevent CSRF
	state := c.Query("state")
	if state != oauthStateString {
		config.Logger.Warn("OAuth callback received invalid state")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	// Exchange the authorization code for a token
	token, err := h.SpotifyService.Authenticator.Token(c.Request.Context(), oauthStateString, c.Request)
	if err != nil {
		config.Logger.Errorf("Failed to exchange OAuth token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get token: " + err.Error()})
		return
	}

	// Save token and initialize client
	h.SpotifyService.SetToken(token)

	config.Logger.Info("✅ Spotify OAuth successful! SilverSync is now authenticated.")
	c.JSON(http.StatusOK, gin.H{
		"message":      "✅ Spotify authentication successful! SilverSync is now ready to sync.",
		"token_type":   token.TokenType,
		"expiry":       token.Expiry,
		"has_refresh":  token.RefreshToken != "",
	})
}

// Status checks if the Spotify service is authenticated.
// GET /auth/status
func (h *AuthHandler) AuthStatus(c *gin.Context) {
	if h.SpotifyService.IsAuthenticated() {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": true,
			"message":       "Spotify client is active and ready.",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"message":       "Not authenticated. Please visit GET /auth/login to connect your Spotify account.",
			"login_url":     "/auth/login",
		})
	}
}
