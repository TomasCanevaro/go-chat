package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-chat/backend/internal/models"
)

func TestInvalidRegistrationRejectedBeforeDatabase(t *testing.T) {
	valid := models.RegisterRequest{Username: "alice", Email: "alice@example.com", Password: "password123"}
	tests := []struct{ name, field, value, message string }{
		{"blank username", "username", "   ", "Username, email and password are required"},
		{"missing username", "username", "", "Username, email and password are required"},
		{"long username", "username", strings.Repeat("a", 51), "Username cannot exceed 50 characters"},
		{"long unicode username", "username", strings.Repeat("界", 51), "Username cannot exceed 50 characters"},
		{"missing email", "email", "", "Username, email and password are required"},
		{"blank email", "email", "  ", "Username, email and password are required"},
		{"malformed email", "email", "not-an-email", "Enter a valid email address"},
		{"display name", "email", "Alice <alice@example.com>", "Enter a valid email address"},
		{"email list", "email", "a@example.com,b@example.com", "Enter a valid email address"},
		{"surrounding whitespace", "email", " alice@example.com ", "Enter a valid email address"},
		{"long email", "email", strings.Repeat("a", 244) + "@example.com", "Email cannot exceed 255 characters"},
		{"missing password", "password", "", "Username, email and password are required"},
		{"short password", "password", "1234567", "Password must contain at least 8 characters"},
		{"short emoji password", "password", strings.Repeat("😀", 4), "Password must contain at least 8 characters"},
		{"long password", "password", strings.Repeat("a", 73), "Password cannot exceed 72 bytes"},
		{"long unicode password", "password", strings.Repeat("界", 25), "Password cannot exceed 72 bytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			switch tt.field {
			case "username":
				req.Username = tt.value
			case "email":
				req.Email = tt.value
			case "password":
				req.Password = tt.value
			}
			data, err := json.Marshal(req)
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			// A nil DB proves invalid requests never reach persistence.
			(&UserHandler{}).Register(response, httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(data)))
			if response.Code != 400 || strings.TrimSpace(response.Body.String()) != tt.message {
				t.Fatalf("Unexpected response %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRegistrationValidationBoundaries(t *testing.T) {
	tests := []models.RegisterRequest{
		{Username: strings.Repeat("a", 50), Email: "alice@example.com", Password: "12345678"},
		{Username: strings.Repeat("界", 50), Email: "alice@example.com", Password: strings.Repeat("a", 72)},
		{Username: "alice", Email: "alice@example.com", Password: strings.Repeat("界", 24)},
		{Username: "alice", Email: "alice@example.com", Password: strings.Repeat("😀", 8)},
		{Username: "alice", Email: strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 62), Password: "12345678"},
	}
	for _, req := range tests {
		if err := validateRegistration(req); err != nil {
			t.Errorf("Valid boundary rejected: %v", err)
		}
	}
}
