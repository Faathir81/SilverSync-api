package handler

import (
	"net/http"
	"silversync-api/internal/config"
	"silversync-api/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

const googleOAuthState = "silversync-google-state"

type GoogleAuthHandler struct {
	driveService service.DriveService
}

func NewGoogleAuthHandler(ds service.DriveService) *GoogleAuthHandler {
	return &GoogleAuthHandler{driveService: ds}
}

// Login redirects the user to Google's OAuth2 consent screen.
func (h *GoogleAuthHandler) Login(c *gin.Context) {
	cfg := h.driveService.GetOAuthConfig()
	url := cfg.AuthCodeURL(googleOAuthState,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)
	config.Logger.Infof("Redirecting user to Google OAuth: %s", url)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// Callback handles the redirect from Google after user grants permission.
func (h *GoogleAuthHandler) Callback(c *gin.Context) {
	if c.Query("state") != googleOAuthState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	code := c.Query("code")
	cfg := h.driveService.GetOAuthConfig()

	token, err := cfg.Exchange(c.Request.Context(), code)
	if err != nil {
		config.Logger.Errorf("Failed to exchange Google OAuth token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	h.driveService.SetToken(token)
	config.Logger.Info("✅ Google Drive OAuth successful!")
	c.JSON(http.StatusOK, gin.H{
		"message": "✅ Google Drive authentication successful! You can now sync music to your Drive.",
	})
}

// AuthStatus returns whether Google Drive is authenticated.
func (h *GoogleAuthHandler) AuthStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"google_drive_authenticated": h.driveService.IsAuthenticated(),
	})
}
