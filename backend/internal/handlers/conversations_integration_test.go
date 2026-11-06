package handlers

import (
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
	"go-chat/backend/internal/auth"
	"go-chat/backend/internal/models"
)

type conversationNotification struct {
	userID int64
	data   []byte
}
type conversationTestNotifier struct {
	notifications []conversationNotification
	onSend        func([]byte)
}

func (n *conversationTestNotifier) SendToUser(userID int64, data []byte) {
	if n.onSend != nil {
		n.onSend(data)
	}
	n.notifications = append(n.notifications, conversationNotification{userID, data})
}

func TestConversationCreationNotifications(t *testing.T) {
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
	ids := make([]int64, 2)
	defer func() {
		cleanup := context.Background()
		db.Exec(cleanup, "DELETE FROM conversation_members WHERE conversation_id IN (SELECT id FROM conversations WHERE direct_key = $1)", makeDirectKey(ids[0], ids[1]))
		db.Exec(cleanup, "DELETE FROM conversations WHERE direct_key = $1", makeDirectKey(ids[0], ids[1]))
		db.Exec(cleanup, "DELETE FROM users WHERE id = ANY($1)", ids)
	}()
	for i := range ids {
		name := fmt.Sprintf("conversation_event_%d_%d", time.Now().UnixNano(), i)
		if err := db.QueryRow(ctx, "INSERT INTO users (username, email, password_hash) VALUES ($1, $2, 'test') RETURNING id", name, name+"@example.com").Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	notifier := &conversationTestNotifier{}
	notifier.onSend = func(data []byte) {
		var event struct {
			Type           string `json:"type"`
			ConversationID int64  `json:"conversation_id"`
		}
		if err := json.Unmarshal(data, &event); err != nil {
			t.Fatal(err)
		}
		if event.Type != "conversation_created" {
			t.Fatalf("Unexpected event: %+v", event)
		}
		// A separate database query must see both committed memberships when notified.
		var members int
		if err := db.QueryRow(ctx, "SELECT count(*) FROM conversation_members WHERE conversation_id = $1", event.ConversationID).Scan(&members); err != nil {
			t.Fatal(err)
		}
		if members != 2 {
			t.Fatalf("Notification before committed membership: %d", members)
		}
	}
	handler := &ConversationHandler{DB: db, Notifier: notifier}
	secret := "conversation-event-test-secret"
	token, err := auth.GenerateToken(ids[0], secret)
	if err != nil {
		t.Fatal(err)
	}
	create := func(otherID int64) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(fmt.Sprintf(`{"user_id":%d}`, otherID))).WithContext(ctx)
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		auth.Middleware(secret, handler.Handle)(response, req)
		return response
	}
	for _, rejected := range []int64{0, ids[0], -1} {
		response := create(rejected)
		if response.Code < 400 {
			t.Fatalf("Expected rejection, got %d", response.Code)
		}
		if len(notifier.notifications) != 0 {
			t.Fatal("Rejected request emitted notification")
		}
	}
	response := create(ids[1])
	if response.Code != http.StatusCreated {
		t.Fatalf("Create: %d %s", response.Code, response.Body.String())
	}
	var created models.Conversation
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if len(notifier.notifications) != 2 {
		t.Fatalf("Expected two recipients, got %d", len(notifier.notifications))
	}
	for i, notification := range notifier.notifications {
		if notification.userID != ids[i] {
			t.Fatalf("Wrong recipient: %d", notification.userID)
		}
		var event struct {
			ConversationID int64 `json:"conversation_id"`
		}
		if err := json.Unmarshal(notification.data, &event); err != nil {
			t.Fatal(err)
		}
		if event.ConversationID != created.ID {
			t.Fatal("Event ID differs from response")
		}
	}
	response = create(ids[1])
	if response.Code != http.StatusOK {
		t.Fatalf("Reuse: %d", response.Code)
	}
	var reused models.Conversation
	if err := json.Unmarshal(response.Body.Bytes(), &reused); err != nil {
		t.Fatal(err)
	}
	if reused.ID != created.ID || len(notifier.notifications) != 2 {
		t.Fatal("Reuse must not emit another creation event")
	}
}
