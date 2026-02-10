package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// User represents an authorized user
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// JWT claims
type JWTClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Exp    int64  `json:"exp"`
}

var jwtSecret []byte

func initAuth() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "boom-dev-secret-change-in-production"
		log.Println("WARNING: JWT_SECRET not set, using default (insecure)")
	}
	jwtSecret = []byte(secret)

	seedUsers()
}

func seedUsers() {
	password := os.Getenv("BOOM_ADMIN_PASSWORD")
	if password == "" {
		password = "boom2026"
		log.Println("WARNING: BOOM_ADMIN_PASSWORD not set, using default")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return
	}

	users := []struct {
		email string
		name  string
	}{
		{"justin@nevinstech.com", "Justin"},
		{"burt@nevinstech.com", "Burt"},
		{"justinnevins@protonmail.com", "Justin N"},
	}

	for _, u := range users {
		_, err := db.Exec(
			"INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?) ON CONFLICT(email) DO NOTHING",
			u.email, string(hash), u.name,
		)
		if err != nil {
			log.Printf("Failed to seed user %s: %v", u.email, err)
		}
	}
	log.Println("Users seeded")
}

// generateJWT creates a signed JWT token
func generateJWT(user *User) (string, error) {
	claims := JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Exp:    time.Now().Add(24 * time.Hour).Unix(),
	}

	header := base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	payloadB64 := base64URLEncode(payload)

	signingInput := header + "." + payloadB64
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	signature := base64URLEncode(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// validateJWT parses and validates a JWT token
func validateJWT(tokenStr string) (*JWTClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(signingInput))
	expectedSig := base64URLEncode(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode claims
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func base64URLDecode(s string) ([]byte, error) {
	// Add padding back
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

// authRequired is Fiber middleware that requires a valid JWT
func authRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := validateJWT(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
		}

		c.Locals("userID", claims.UserID)
		c.Locals("userEmail", claims.Email)
		c.Locals("userName", claims.Name)
		return c.Next()
	}
}

// Login handler
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func loginHandler(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Find user by email
	var user User
	err := db.QueryRow(
		"SELECT id, email, password_hash, name, created_at FROM users WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.CreatedAt)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	// Generate token
	token, err := generateJWT(&user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

func meHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"id":    c.Locals("userID"),
		"email": c.Locals("userEmail"),
		"name":  c.Locals("userName"),
	})
}
