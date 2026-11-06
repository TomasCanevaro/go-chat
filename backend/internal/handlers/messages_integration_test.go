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
	"go-chat/backend/internal/models"
)

func TestMessageHistoryPagination(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var userID, conversationID, emptyID int64
	defer func() {
		db.Exec(context.Background(), "DELETE FROM conversations WHERE id = ANY($1)", []int64{conversationID, emptyID})
		db.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	}()
	name := fmt.Sprintf("history_%d", time.Now().UnixNano())
	if err := db.QueryRow(ctx, "INSERT INTO users (username,email,password_hash) VALUES ($1,$2,'test') RETURNING id", name, name+"@example.com").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []*int64{&conversationID, &emptyID} {
		if err := db.QueryRow(ctx, "INSERT INTO conversations (type) VALUES ('direct') RETURNING id").Scan(id); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, "INSERT INTO conversation_members (conversation_id,user_id) VALUES ($1,$2)", *id, userID); err != nil {
			t.Fatal(err)
		}
	}
	ids := []int64{}
	insert := func(conversation int64) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(ctx, "INSERT INTO messages (conversation_id,sender_id,content,created_at) VALUES ($1,$2,'page test','2026-01-01T00:00:00Z') RETURNING id", conversation, userID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	for i := 0; i < 35; i++ {
		ids = append(ids, insert(conversationID))
	}
	// A newer message in another conversation must not leak into this page.
	unrelatedID := insert(emptyID)
	secret := "pagination-test-secret"
	token, err := auth.GenerateToken(userID, secret)
	if err != nil {
		t.Fatal(err)
	}
	handler := &MessageHandler{DB: db}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/conversations/{id}/messages", auth.Middleware(secret, handler.List))
	request := func(conversation int64, query, bearer string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/conversations/%d/messages%s", conversation, query), nil).WithContext(ctx)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	page := func(conversation int64, query string) models.MessagePage {
		t.Helper()
		response := request(conversation, query, token)
		if response.Code != 200 {
			t.Fatalf("Page status %d: %s", response.Code, response.Body.String())
		}
		var result models.MessagePage
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	assertIDs := func(result models.MessagePage, expected []int64, more bool) {
		t.Helper()
		if len(result.Messages) != len(expected) || result.HasMore != more {
			t.Fatalf("Unexpected page: %d messages, has_more=%v; expected %d/%v", len(result.Messages), result.HasMore, len(expected), more)
		}
		for i, message := range result.Messages {
			if message.ID != expected[i] || message.ConversationID != conversationID || message.SenderUsername != name {
				t.Fatalf("Unexpected message: %+v", message)
			}
		}
	}
	assertIDs(page(conversationID, "?limit=30"), ids[5:], true)
	// A cursor without an explicit limit uses the default page size of 30.
	assertIDs(page(conversationID, fmt.Sprintf("?before_id=%d", unrelatedID)), ids[5:], true)
	assertIDs(page(conversationID, "?limit=100"), ids, false)
	assertIDs(page(conversationID, "?limit=1"), ids[34:], true)
	first := page(conversationID, "?limit=2")
	assertIDs(first, ids[33:], true)
	insert(conversationID)
	// A concurrent new insert must not shift older pages or duplicate rows.
	assertIDs(page(conversationID, fmt.Sprintf("?limit=2&before_id=%d", first.Messages[0].ID)), ids[31:33], true)
	assertIDs(page(conversationID, fmt.Sprintf("?limit=2&before_id=%d", ids[2])), ids[:2], false)
	assertIDs(page(conversationID, fmt.Sprintf("?limit=2&before_id=%d", ids[0])), []int64{}, false)
	empty := page(emptyID, "?limit=30&before_id=1")
	if empty.Messages == nil || len(empty.Messages) != 0 || empty.HasMore {
		t.Fatalf("Expected empty JSON array: %+v", empty)
	}
	legacy := request(conversationID, "", token)
	var messages []models.Message
	if legacy.Code != 200 {
		t.Fatal(legacy.Code)
	}
	if err := json.Unmarshal(legacy.Body.Bytes(), &messages); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 36 {
		t.Fatalf("Legacy request lost messages: %d", len(messages))
	}
	for _, query := range []string{"?limit=0", "?limit=101", "?limit=-1", "?limit=abc", "?limit=", "?before_id=0", "?before_id=-1", "?before_id=abc", "?before_id=", "?before_id=9999999999999999999999999"} {
		if got := request(conversationID, query, token).Code; got != 400 {
			t.Errorf("%s: got %d", query, got)
		}
	}
	if got := request(conversationID, "?limit=30", "").Code; got != 401 {
		t.Fatalf("Unauthenticated: %d", got)
	}
	outsider, err := auth.GenerateToken(userID+1000000, secret)
	if err != nil {
		t.Fatal(err)
	}
	if got := request(conversationID, "?limit=30", outsider).Code; got != 404 {
		t.Fatalf("Nonmember: %d", got)
	}
	if got := request(9223372036854775807, "?limit=30", token).Code; got != 404 {
		t.Fatalf("Missing conversation: %d", got)
	}
}
