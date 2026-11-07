package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go-chat/backend/internal/auth"
)

func TestUserDirectoryAuthentication(t *testing.T) {
	const secret = "directory-test-secret"
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1, "exp": time.Now().Add(-time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	handler := &UserHandler{} // Rejected requests must not reach the database.
	protected := auth.Middleware(secret, handler.GetUsers)
	for _, header := range []string{"", "Bearer invalid", "Basic credentials", "Bearer " + expired} {
		t.Run(header, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			if header != "" {
				request.Header.Set("Authorization", header)
			}
			response := httptest.NewRecorder()
			protected(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("Expected 401, got %d", response.Code)
			}
		})
	}
	token, err := auth.GenerateToken(1, secret)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	protected(response, request)
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET" {
		t.Fatalf("Unexpected method response: %d", response.Code)
	}
	response = httptest.NewRecorder()
	handler.GetUsers(response, httptest.NewRequest(http.MethodGet, "/api/users", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatal("Unwrapped handler must require user context")
	}
}
