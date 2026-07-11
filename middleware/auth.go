// Package middleware provides HTTP middleware functions
package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultSessionTTL = 365 * 24 * time.Hour

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Username      string
	Password      string
	SessionSecret string
	SessionTTL    time.Duration
	Enabled       bool
}

var (
	authConfig     *AuthConfig
	authConfigOnce sync.Once
	sessionStore   = make(map[string]*sessionInfo)
	sessionMutex   sync.RWMutex
)

type sessionInfo struct {
	Username  string
	ExpiresAt time.Time
}

// GetAuthConfig returns the authentication configuration (lazy loaded)
func GetAuthConfig() *AuthConfig {
	authConfigOnce.Do(func() {
		username := os.Getenv("AUTH_USERNAME")
		password := os.Getenv("AUTH_PASSWORD")
		secret := os.Getenv("SESSION_SECRET")
		sessionTTL := parseSessionTTL(os.Getenv("SESSION_TTL"))

		// If both username and password are not set, disable authentication
		enabled := username != "" && password != ""

		// Generate a random session secret if not provided
		if secret == "" && enabled {
			secret = generateRandomSecret()
		}

		authConfig = &AuthConfig{
			Username:      username,
			Password:      password,
			SessionSecret: secret,
			SessionTTL:    sessionTTL,
			Enabled:       enabled,
		}
	})
	return authConfig
}

// parseSessionTTL parses the session validity period from environment variables.
func parseSessionTTL(raw string) time.Duration {
	if raw == "" {
		return defaultSessionTTL
	}

	if strings.HasSuffix(raw, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
		if err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	ttl, err := time.ParseDuration(raw)
	if err == nil && ttl > 0 {
		return ttl
	}

	log.Printf("无效的 SESSION_TTL=%q，使用默认值 %s", raw, defaultSessionTTL)
	return defaultSessionTTL
}

// generateRandomSecret generates a random 32-byte secret
func generateRandomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based seed if crypto/rand fails
		return base64.StdEncoding.EncodeToString([]byte(time.Now().String()))
	}
	return base64.StdEncoding.EncodeToString(b)
}

// AuthMiddleware checks if the user is authenticated
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := GetAuthConfig()

		// If authentication is disabled, pass through
		if !config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Check for session cookie
		sessionCookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Validate session
		sessionMutex.RLock()
		session, exists := sessionStore[sessionCookie.Value]
		sessionMutex.RUnlock()

		if !exists || time.Now().After(session.ExpiresAt) {
			// Session expired or invalid
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Session is valid, continue
		next.ServeHTTP(w, r)
	})
}

// APIAuthMiddleware checks if the user is authenticated for API requests
func APIAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := GetAuthConfig()

		// If authentication is disabled, pass through
		if !config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Check for session cookie
		sessionCookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate session
		sessionMutex.RLock()
		session, exists := sessionStore[sessionCookie.Value]
		sessionMutex.RUnlock()

		if !exists || time.Now().After(session.ExpiresAt) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Session is valid, continue
		next.ServeHTTP(w, r)
	})
}

// CreateSession creates a new session for the given username
func CreateSession(username string) (string, error) {
	// Generate session ID using random data
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	sessionID := base64.StdEncoding.EncodeToString(randomBytes)

	config := GetAuthConfig()

	// Store session with configurable expiration
	sessionMutex.Lock()
	sessionStore[sessionID] = &sessionInfo{
		Username:  username,
		ExpiresAt: time.Now().Add(config.SessionTTL),
	}
	sessionMutex.Unlock()

	// Clean up expired sessions
	cleanupExpiredSessions()

	return sessionID, nil
}

// DeleteSession removes a session
func DeleteSession(sessionID string) {
	sessionMutex.Lock()
	delete(sessionStore, sessionID)
	sessionMutex.Unlock()
}

// GetSession retrieves a session by ID
func GetSession(sessionID string) *sessionInfo {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()

	session := sessionStore[sessionID]
	if session == nil || time.Now().After(session.ExpiresAt) {
		return nil
	}
	return session
}

// cleanupExpiredSessions removes expired sessions from the store
func cleanupExpiredSessions() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	now := time.Now()
	for id, session := range sessionStore {
		if now.After(session.ExpiresAt) {
			delete(sessionStore, id)
		}
	}
}
