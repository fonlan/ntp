package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
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
	translator := middleware.TranslatorFromContext(r.Context())
	config := middleware.GetAuthConfig()

	// If authentication is disabled, return success
	if !config.Enabled {
		respondJSON(w, http.StatusBadRequest, LoginResponse{
			Success: false,
			Message: translator.T("auth.disabled"),
		})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, LoginResponse{
			Success: false,
			Message: translator.T("common.invalidRequest"),
		})
		return
	}

	// Validate credentials
	if req.Username == config.Username && req.Password == config.Password {
		// Create session
		sessionID, err := middleware.CreateSession(req.Username)
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, LoginResponse{
				Success: false,
				Message: translator.T("common.internalError"),
			})
			return
		}

		// Determine if the connection is secure (HTTPS)
		isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionID,
			Path:     "/",
			Expires:  time.Now().Add(config.SessionTTL),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   isSecure,
		})

		respondJSON(w, http.StatusOK, LoginResponse{
			Success: true,
			Message: translator.T("auth.loginSuccess"),
		})
		return
	}

	// Invalid credentials
	respondJSON(w, http.StatusUnauthorized, LoginResponse{
		Success: false,
		Message: translator.T("auth.invalidCredentials"),
	})
}

// Logout handles logout requests
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	translator := middleware.TranslatorFromContext(r.Context())

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
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecure,
	})

	// Return success
	respondJSON(w, http.StatusOK, LoginResponse{
		Success: true,
		Message: translator.T("auth.logoutSuccess"),
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
				response.Username = strings.TrimSpace(session.Username)
			}
		}
	}

	respondJSON(w, http.StatusOK, response)
}
