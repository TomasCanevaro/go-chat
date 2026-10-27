package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"fmt"

	"go-chat/backend/internal/auth"
	"go-chat/backend/internal/services"

	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func setupWebSocketTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	_ = godotenv.Load("../../.env")

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required for WebSocket integration tests")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("Invalid test database URL: %v", err)
	}

	if config.ConnConfig.Database != "chatdb_test" {
		t.Fatal("Integration tests must use chatdb_test")
	}

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create test database pool: %v", err)
	}

	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	t.Cleanup(db.Close)

	return db
}

func setupWebSocketTestServer(
	t *testing.T,
	db *pgxpool.Pool,
	secret string,
) *httptest.Server {
	t.Helper()

	hub := NewHub()

	messageService := &services.MessageService{
		DB:       db,
		Notifier: hub,
	}

	handler := &Handler{
		JWTSecret:      secret,
		Hub:            hub,
		MessageService: messageService,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.Handle)

	server := httptest.NewServer(mux)

	t.Cleanup(server.Close)

	return server
}

func connectAuthenticatedClient(
	t *testing.T,
	server *httptest.Server,
	userID int64,
	secret string,
) *websocket.Conn {
	t.Helper()

	token, err := auth.GenerateToken(userID, secret)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}

	t.Cleanup(func() {
		conn.Close(websocket.StatusNormalClosure, "test finished")
	})

	authMessage, err := json.Marshal(map[string]string{
		"type":  "auth",
		"token": token,
	})
	if err != nil {
		t.Fatalf("Failed to marshal authentication message: %v", err)
	}

	if err := conn.Write(ctx, websocket.MessageText, authMessage); err != nil {
		t.Fatalf("Failed to send authentication message: %v", err)
	}

	_, response, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("Failed to read authentication response: %v", err)
	}

	var event struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(response, &event); err != nil {
		t.Fatalf("Failed to decode authentication response: %v", err)
	}

	if event.Type != "auth_success" {
		t.Fatalf("Expected auth_success, got %s", event.Type)
	}

	return conn
}

func TestWebSocketAuthentication(t *testing.T) {
	secret := "websocket-test-secret"

	hub := NewHub()

	handler := &Handler{
		JWTSecret: secret,
		Hub:       hub,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.Handle)

	server := httptest.NewServer(mux)
	defer server.Close()

	conn := connectAuthenticatedClient(t, server, 42, secret)

	if conn == nil {
		t.Fatal("Expected authenticated WebSocket connection")
	}

	if got := hub.ConnectionCount(42); got != 1 {
		t.Errorf("Expected 1 active connection, got %d", got)
	}
}

type websocketTestFixture struct {
	AliceID        int64
	BobID          int64
	ConversationID int64
}

func createWebSocketTestFixture(
	t *testing.T,
	db *pgxpool.Pool,
) websocketTestFixture {
	t.Helper()

	ctx := context.Background()

	uniqueID := time.Now().UnixNano()

	aliceUsername := fmt.Sprintf("ws_alice_%d", uniqueID)
	bobUsername := fmt.Sprintf("ws_bob_%d", uniqueID)

	var aliceID, bobID, conversationID int64

	err := db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, 'test_hash')
		RETURNING id
	`, aliceUsername, aliceUsername+"@example.com").Scan(&aliceID)

	if err != nil {
		t.Fatalf("Failed to create Alice: %v", err)
	}

	err = db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, 'test_hash')
		RETURNING id
	`, bobUsername, bobUsername+"@example.com").Scan(&bobID)

	if err != nil {
		t.Fatalf("Failed to create Bob: %v", err)
	}

	err = db.QueryRow(ctx, `
		INSERT INTO conversations (type)
		VALUES ('direct')
		RETURNING id
	`).Scan(&conversationID)

	if err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO conversation_members (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
	`, conversationID, aliceID, bobID)

	if err != nil {
		t.Fatalf("Failed to create conversation members: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()

		db.Exec(cleanupCtx,
			"DELETE FROM messages WHERE conversation_id = $1",
			conversationID,
		)

		db.Exec(cleanupCtx,
			"DELETE FROM conversation_members WHERE conversation_id = $1",
			conversationID,
		)

		db.Exec(cleanupCtx,
			"DELETE FROM conversations WHERE id = $1",
			conversationID,
		)

		db.Exec(cleanupCtx,
			"DELETE FROM users WHERE id IN ($1, $2)",
			aliceID,
			bobID,
		)
	})

	return websocketTestFixture{
		AliceID:        aliceID,
		BobID:          bobID,
		ConversationID: conversationID,
	}
}

func TestWebSocketMessageDelivery(t *testing.T) {
	db := setupWebSocketTestDB(t)

	fixture := createWebSocketTestFixture(t, db)

	secret := "websocket-test-secret"

	server := setupWebSocketTestServer(t, db, secret)

	alice := connectAuthenticatedClient(t, server, fixture.AliceID, secret)
	bob := connectAuthenticatedClient(t, server, fixture.BobID, secret)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Alice sends a message through WebSocket.
	request := map[string]interface{}{
		"type":            "send_message",
		"conversation_id": fixture.ConversationID,
		"content":         "Hello Bob, this is an automated test!",
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	if err := alice.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Bob should receive the new_message event.
	_, response, err := bob.Read(ctx)
	if err != nil {
		t.Fatalf("Bob did not receive message: %v", err)
	}

	var event struct {
		Type    string `json:"type"`
		Message struct {
			ID             int64  `json:"id"`
			ConversationID int64  `json:"conversation_id"`
			SenderID       int64  `json:"sender_id"`
			Content        string `json:"content"`
		} `json:"message"`
	}

	if err := json.Unmarshal(response, &event); err != nil {
		t.Fatalf("Failed to decode event: %v", err)
	}

	if event.Type != "new_message" {
		t.Fatalf("Expected new_message, got %s", event.Type)
	}

	if event.Message.Content != "Hello Bob, this is an automated test!" {
		t.Errorf("Unexpected content: %s", event.Message.Content)
	}

	if event.Message.SenderID != fixture.AliceID {
		t.Errorf("Expected sender ID %d, got %d",
			fixture.AliceID,
			event.Message.SenderID,
		)
	}

	if event.Message.ConversationID != fixture.ConversationID {
		t.Errorf("Expected conversation ID %d, got %d",
			fixture.ConversationID,
			event.Message.ConversationID,
		)
	}

	// Verify PostgreSQL persistence.
	var savedContent string

	err = db.QueryRow(ctx, `
		SELECT content
		FROM messages
		WHERE id = $1
	`, event.Message.ID).Scan(&savedContent)

	if err != nil {
		t.Fatalf("Failed to retrieve saved message: %v", err)
	}

	if savedContent != event.Message.Content {
		t.Errorf("Persisted content does not match WebSocket event")
	}
}
