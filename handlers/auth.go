// Package handles HTTP handlers for authentication
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"ntp/middleware"
)

// AuthHandler handles authentication requests
type AuthHandler struct{}

// NewAuthHandler creates a new auth handler
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// AuthCheckResponse represents an auth check response
type AuthCheckResponse struct {
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
	Enabled       bool   `json:"enabled"`
}

// Login handles login requests
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	config := middleware.GetAuthConfig()

	// If authentication is disabled, return success
	if !config.Enabled {
		http.Error(w, "Authentication is disabled", http.StatusBadRequest)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate credentials
	if req.Username == config.Username && req.Password == config.Password {
		// Create session
		sessionID, err := middleware.CreateSession(req.Username)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		// Determine if the connection is secure (HTTPS)
		isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionID,
			Path:     "/",
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   isSecure,
		})

		// Return success
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			Success: true,
			Message: "Login successful",
		})
		return
	}

	// Invalid credentials
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(LoginResponse{
		Success: false,
		Message: "Invalid username or password",
	})
}

// Logout handles logout requests
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	sessionCookie, err := r.Cookie("session")
	if err == nil {
		// Delete session
		middleware.DeleteSession(sessionCookie.Value)
	}

	// Determine if the connection is secure (HTTPS)
	isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isSecure,
	})

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Success: true,
		Message: "Logout successful",
	})
}

// CheckAuth checks if the user is authenticated
func (h *AuthHandler) CheckAuth(w http.ResponseWriter, r *http.Request) {
	config := middleware.GetAuthConfig()

	response := AuthCheckResponse{
		Enabled: config.Enabled,
	}

	// If authentication is disabled, return not authenticated
	if !config.Enabled {
		response.Authenticated = false
	} else {
		// Check for session cookie
		sessionCookie, err := r.Cookie("session")
		if err == nil {
			// Verify session is valid (exists and not expired)
			session := middleware.GetSession(sessionCookie.Value)
			if session != nil {
				response.Authenticated = true
				response.Username = session.Username
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
