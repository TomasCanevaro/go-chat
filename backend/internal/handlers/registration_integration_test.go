package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go-chat/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func TestRegistrationAndLoginAtPasswordByteLimit(t *testing.T) {
	_ = godotenv.Load("../../.env")
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}
	config, err := pgxpool.ParseConfig(url)
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
	name := fmt.Sprintf("registration_%d", time.Now().UnixNano())
	defer db.Exec(context.Background(), "DELETE FROM users WHERE username=$1", name)
	password := strings.Repeat("界", 24) // Exactly 72 UTF-8 bytes.
	req := models.RegisterRequest{Username: name, Email: name + "@example.com", Password: password}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	handler := &UserHandler{DB: db, JWTSecret: "registration-test-secret"}
	register := func() *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		handler.Register(response, httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewReader(data)).WithContext(ctx))
		return response
	}
	response := register()
	if response.Code != http.StatusCreated {
		t.Fatalf("Registration: %d %s", response.Code, response.Body.String())
	}
	var hash string
	if err := db.QueryRow(ctx, "SELECT password_hash FROM users WHERE username=$1", name).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash == password || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		t.Fatal("Password was not correctly hashed")
	}
	response = register()
	if response.Code != http.StatusConflict {
		t.Fatalf("Duplicate registration: %d", response.Code)
	}
	login, err := json.Marshal(models.LoginRequest{Email: req.Email, Password: password})
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.Login(response, httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(login)).WithContext(ctx))
	if response.Code != http.StatusOK {
		t.Fatalf("Login: %d %s", response.Code, response.Body.String())
	}
}
