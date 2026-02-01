package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"registration-app/config"
	"registration-app/database"
	"registration-app/internal/domain/site"
	"registration-app/internal/domain/users"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func isPasswordStrong(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, c := range password {
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z':
			hasLetter = true
		case '0' <= c && c <= '9':
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

func isEmailValid(email string) bool {
	pattern := `^[a-zA-Z0-9._%%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func generateVerificationToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func Register(c *gin.Context) {
	var input struct {
		Name     string `json:"name" binding:"required"`
		Lastname string `json:"lastname" binding:"required"`
		Tel      string `json:"tel" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !isPasswordStrong(input.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters long and contain both letters and numbers"})
		return
	}
	if !isEmailValid(input.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	hashed := string(hashedPassword)

	now := time.Now()
	trialDays := 14
	trialEnd := now.AddDate(0, 0, trialDays)

	user := users.User{
		Name:         input.Name,
		Lastname:     input.Lastname,
		Tel:          input.Tel,
		Email:        input.Email,
		Password:     &hashed, // ✅ pointer now
		AuthProvider: "local", // ✅ explicitly mark local signup
		GoogleSub:    nil,     // ✅ (if you added this field)
		Role:         "user",
		IsVerified:   false,

		TrialStartAt: &now,
		TrialEndAt:   &trialEnd,
	}

	// ✅ Create ONCE
	if err := database.DB.Create(&user).Error; err != nil {
		fmt.Println("❌ DB Insert Error:", err)
		c.JSON(http.StatusConflict, gin.H{"error": "Email may already exist", "details": err.Error()})
		return
	}

	// ✅ Ensure slug (UPDATE, not INSERT)
	slug, err := site.EnsureSiteSlug(database.DB, &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate site slug", "details": err.Error()})
		return
	}
	_ = site.BuildPublicURL(slug)

	token := generateVerificationToken()

	fmt.Printf("📬 Verification token for %s: %s\n", user.Email, token)
	link := fmt.Sprintf("http://localhost:8080/verify?token=%s", token)
	fmt.Println("📨 Verification link:", link)

	verif := users.VerificationToken{
		UserID: user.ID,
		Token:  token,
	}

	if err := database.DB.Create(&verif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create verification token"})
		return
	}

	// Send email
	if err := SendVerificationEmail(user.Email, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully. Please check your email to verify your account."})
}

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user users.User
	err := database.DB.Where("email = ?", input.Email).First(&user).Error
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Please verify your email before logging in"})
		return
	}

	if user.Password == nil || *user.Password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "This account uses Google sign-in"})
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(input.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	jwtKey := []byte(config.JWT_SECRET)
	tokenString, err := token.SignedString(jwtKey)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func ResendVerification(c *gin.Context) {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid email"})
		return
	}

	var user users.User
	err := database.DB.Where("email = ?", body.Email).First(&user).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already verified"})
		return
	}

	// Remove old token if exists
	database.DB.Where("user_id = ?", user.ID).Delete(&users.VerificationToken{})

	token := generateVerificationToken()
	newToken := users.VerificationToken{
		UserID: user.ID,
		Token:  token,
	}
	err = database.DB.Create(&newToken).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store verification token"})
		return
	}

	link := fmt.Sprintf("http://localhost:8080/verify?token=%s", token)
	fmt.Println("📨 Verification link:", link)

	c.JSON(http.StatusOK, gin.H{"message": "Verification email resent"})
}

func RequestPasswordReset(c *gin.Context) {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email"})
		return
	}

	var user users.User
	if err := database.DB.Where("email = ?", body.Email).First(&user).Error; err != nil {
		// Don't expose whether the email exists
		c.JSON(http.StatusOK, gin.H{"message": "If your email exists, you'll receive a reset link."})
		return
	}

	// Remove any existing reset tokens for this user
	database.DB.Where("user_id = ? AND type = ?", user.ID, "password_reset").Delete(&users.VerificationToken{})

	// Create secure token
	token := generateVerificationToken()

	reset := users.VerificationToken{
		UserID:    user.ID,
		Token:     token,
		Type:      "password_reset", // ✅ Add the type
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	database.DB.Create(&reset)

	// Simulate email
	resetLink := fmt.Sprintf("http://localhost:5173/reset-password?token=%s", token)
	fmt.Println("📨 Reset link:", resetLink)

	c.JSON(http.StatusOK, gin.H{"message": "If your email exists, you'll receive a reset link."})
}

func ResetPassword(c *gin.Context) {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if !isPasswordStrong(body.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters with letters and numbers"})
		return
	}

	var reset users.VerificationToken
	err := database.DB.Where("token = ? AND type = ?", body.Token, "password_reset").First(&reset).Error
	if err != nil || reset.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Update user password
	hashed, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	database.DB.Model(&users.User{}).Where("id = ?", reset.UserID).Update("password", string(hashed))

	// Remove the used token
	database.DB.Delete(&reset)

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successful"})
}

func ChangePassword(c *gin.Context) {
	userIDD := c.GetUint("user_id")
	fmt.Println("🔐 userID from context:", userIDD)
	// Extract user ID from context
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDAny.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse request body
	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if !isPasswordStrong(body.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be at least 8 characters with letters and numbers"})
		return
	}

	// Fetch user from DB
	var user users.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.Password == nil || *user.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "This account does not have a password. Sign in with Google or set a password first.",
		})
		return
	}

	// Compare current password
	if err := bcrypt.CompareHashAndPassword(
		[]byte(*user.Password),
		[]byte(body.OldPassword),
	); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Old password is incorrect"})
		return
	}

	// Update to new password
	hashedNew, _ := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	database.DB.Model(&user).Update("password", string(hashedNew))

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
