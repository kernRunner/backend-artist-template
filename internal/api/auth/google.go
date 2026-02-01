package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"registration-app/config"
	"registration-app/database"
	"registration-app/internal/domain/users"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.GOOGLE_CLIENT_ID,
		ClientSecret: config.GOOGLE_CLIENT_SECRET,
		RedirectURL:  config.GOOGLE_REDIRECT_URL,
		Scopes: []string{
			"openid",
			"email",
			"profile",
		},
		Endpoint: google.Endpoint,
	}
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GET /auth/google
func GoogleStart(c *gin.Context) {
	state, err := randomState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	// store state in an HttpOnly cookie (simple + works well)
	c.SetCookie(
		"oauth_state",
		state,
		300, // 5 minutes
		"/",
		"",    // domain (set in prod)
		false, // secure (true in prod HTTPS)
		true,  // httpOnly
	)

	url := googleOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusFound, url)
}

// GET /auth/google/callback
func GoogleCallback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")
	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code/state"})
		return
	}

	cookieState, err := c.Cookie("oauth_state")
	if err != nil || cookieState != state {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	// exchange code -> tokens
	tok, err := googleOAuthConfig().Exchange(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to exchange code"})
		return
	}

	// Google returns an ID token (JWT) with openid scope
	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing id_token"})
		return
	}

	// ✅ Minimal verification WITHOUT extra libs:
	// Parse (but also validate claims). For full OIDC signature verification, see note below.
	claims, err := verifyGoogleIDToken(c, rawIDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	email := claims.Email
	if email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "google account has no email"})
		return
	}

	// Find or create user
	user, err := findOrCreateGoogleUser(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// issue your normal JWT (same as Login)
	tokenString, err := issueAppJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	// Option A: return JSON token
	// c.JSON(http.StatusOK, gin.H{"token": tokenString})

	// Option B: redirect to frontend with token
	redirect := config.GOOGLE_FRONTEND_REDIRECT
	if redirect == "" {
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
		return
	}
	c.Redirect(http.StatusFound, redirect+"?token="+tokenString)
}

/* ---------------- helpers ---------------- */

type googleIDClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Iss           string `json:"iss"`
	Aud           string `json:"aud"`
	Exp           int64  `json:"exp"`
	Iat           int64  `json:"iat"`
}

// NOTE: This parses and validates claims, but does NOT verify signature.
// Best practice is full OIDC verification (below).
func verifyGoogleIDToken(c *gin.Context, rawIDToken string) (*googleIDClaims, error) {
	ctx := c.Request.Context()

	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, errors.New("failed to init google oidc provider")
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.GOOGLE_CLIENT_ID,
	})

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, errors.New("invalid id_token")
	}

	var claims googleIDClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, errors.New("failed to decode token claims")
	}

	// Optional extra checks
	if claims.Email == "" || claims.Sub == "" {
		return nil, errors.New("token missing required claims")
	}

	return &claims, nil
}

func findOrCreateGoogleUser(gc *googleIDClaims) (users.User, error) {
	var user users.User

	// 1) Try by google_sub
	if gc.Sub != "" {
		if err := database.DB.Where("google_sub = ?", gc.Sub).First(&user).Error; err == nil {
			return user, nil
		}
	}

	// 2) Try by email, then link google_sub if missing
	if err := database.DB.Where("email = ?", gc.Email).First(&user).Error; err == nil {
		// Link sub if not linked yet
		if user.GoogleSub == nil {
			sub := gc.Sub
			user.GoogleSub = &sub
			user.AuthProvider = "google"
			user.IsVerified = true
			if err := database.DB.Save(&user).Error; err != nil {
				return users.User{}, err
			}
		}
		return user, nil
	}

	// 3) Create new user (google)
	now := time.Now()
	trialEnd := now.AddDate(0, 0, 14)
	sub := gc.Sub

	user = users.User{
		Name:         firstNonEmpty(gc.GivenName, gc.Name),
		Lastname:     gc.FamilyName,
		Email:        gc.Email,
		Password:     nil,
		AuthProvider: "google",
		GoogleSub:    &sub,
		Role:         "user",
		IsVerified:   true,
		TrialStartAt: &now,
		TrialEndAt:   &trialEnd,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return users.User{}, err
	}
	return user, nil
}

func issueAppJWT(user users.User) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return t.SignedString([]byte(config.JWT_SECRET))
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}
