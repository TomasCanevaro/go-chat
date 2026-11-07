package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go-chat/backend/internal/auth"
)

func TestUserDirectoryResponse(t *testing.T) {
	_ = godotenv.Load("../../.env")
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal("Invalid test database configuration")
	}
	if config.ConnConfig.Database != "chatdb_test" {
		t.Fatal("Tests must use chatdb_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	name := fmt.Sprintf("directory_%d", time.Now().UnixNano())
	var userID int64
	if err := db.QueryRow(ctx, "INSERT INTO users (username,email,password_hash) VALUES ($1,$2,'private_hash') RETURNING id", name, name+"@example.com").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(context.Background(), "DELETE FROM users WHERE id=$1", userID)
	const secret = "directory-test-secret"
	token, err := auth.GenerateToken(userID, secret)
	if err != nil {
		t.Fatal(err)
	}
	handler := &UserHandler{DB: db}
	request := httptest.NewRequest(http.MethodGet, "/api/users", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	auth.Middleware(secret, handler.GetUsers)(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatal("Expected JSON")
	}
	var users []map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &users); err != nil {
		t.Fatal(err)
	}
	found := false
	var previousID int64
	for _, user := range users {
		if len(user) != 2 || user["id"] == nil || user["username"] == nil {
			t.Fatalf("Unexpected directory fields: %v", user)
		}
		var id int64
		var username string
		if err := json.Unmarshal(user["id"], &id); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(user["username"], &username); err != nil {
			t.Fatal(err)
		}
		if id <= previousID {
			t.Fatal("Directory must be ordered by ID")
		}
		previousID = id
		if id == userID {
			found = true
			if username != name {
				t.Fatal("Wrong username")
			}
		}
	}
	if !found {
		t.Fatal("Fixture user missing from authenticated directory")
	}
	// /api/me keeps its separate, full user response.
	response = httptest.NewRecorder()
	auth.Middleware(secret, handler.Me)(response, request)
	var profile map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || profile["email"] == nil || profile["created_at"] == nil {
		t.Fatal("Current user profile must retain its fields")
	}
}
