package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestInvalidMessageClientID(t *testing.T) {
	service := &MessageService{}
	for _, clientID := range []string{strings.Repeat("x", 129), "   "} {
		if _, err := service.CreateWithClientID(context.Background(), 1, 1, "Hello", clientID); !errors.Is(err, ErrInvalidClientID) {
			t.Fatalf("Expected invalid ID, got %v", err)
		}
	}
}

type deduplicationNotifier struct {
	mu         sync.Mutex
	deliveries int
}

func (n *deduplicationNotifier) SendToUser(_ int64, _ []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.deliveries++
}
func (n *deduplicationNotifier) count() int { n.mu.Lock(); defer n.mu.Unlock(); return n.deliveries }

func TestMessageRetryDeduplication(t *testing.T) {
	db := setupTestDB(t)
	fixture := createTestConversation(t, db)
	other := createTestConversation(t, db)
	ctx := context.Background()
	notifier := &deduplicationNotifier{}
	service := &MessageService{DB: db, Notifier: notifier}
	original, err := service.CreateWithClientID(ctx, fixture.ConversationID, fixture.UserID, "  Hello  ", "retry-1")
	if err != nil {
		t.Fatal(err)
	}
	retry, err := service.CreateWithClientID(ctx, fixture.ConversationID, fixture.UserID, "Hello", "retry-1")
	if err != nil {
		t.Fatal(err)
	}
	if retry.ID != original.ID || !retry.CreatedAt.Equal(original.CreatedAt) || notifier.count() != 1 {
		t.Fatal("Retry must return original message without rebroadcasting")
	}
	if _, err := service.CreateWithClientID(ctx, fixture.ConversationID, fixture.UserID, "Changed", "retry-1"); !errors.Is(err, ErrClientIDConflict) {
		t.Fatalf("Changed content: %v", err)
	}
	// Same key is independent for another sender.
	if _, err := service.CreateWithClientID(ctx, other.ConversationID, other.UserID, "Hello", "retry-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO conversation_members (conversation_id,user_id) VALUES ($1,$2)", other.ConversationID, fixture.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateWithClientID(ctx, other.ConversationID, fixture.UserID, "Hello", "retry-1"); !errors.Is(err, ErrClientIDConflict) {
		t.Fatalf("Changed conversation: %v", err)
	}
	// Retry must not bypass authorization after membership is removed.
	if _, err := db.Exec(ctx, "DELETE FROM conversation_members WHERE conversation_id=$1 AND user_id=$2", fixture.ConversationID, fixture.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateWithClientID(ctx, fixture.ConversationID, fixture.UserID, "Hello", "retry-1"); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("Revoked membership: %v", err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO conversation_members (conversation_id,user_id) VALUES ($1,$2)", fixture.ConversationID, fixture.UserID); err != nil {
		t.Fatal(err)
	}
	// Different IDs allow intentional identical messages; no-ID sends remain compatible.
	next, err := service.CreateWithClientID(ctx, fixture.ConversationID, fixture.UserID, "Hello", "retry-2")
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == original.ID {
		t.Fatal("Different IDs must create distinct messages")
	}
	for i := 0; i < 2; i++ {
		if _, err := service.Create(ctx, fixture.ConversationID, fixture.UserID, "Legacy"); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM messages WHERE conversation_id=$1", fixture.ConversationID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("Expected 4 persisted messages, got %d", count)
	}
}

func TestConcurrentMessageRetries(t *testing.T) {
	db := setupTestDB(t)
	fixture := createTestConversation(t, db)
	notifier := &deduplicationNotifier{}
	service := &MessageService{DB: db, Notifier: notifier}
	type result struct {
		id  int64
		err error
	}
	const workers = 8
	results := make(chan result, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func() {
			<-start
			message, err := service.CreateWithClientID(context.Background(), fixture.ConversationID, fixture.UserID, "Concurrent", "same-id")
			var id int64
			if message != nil {
				id = message.ID
			}
			results <- result{id, err}
		}()
	}
	close(start)
	var expectedID int64
	for i := 0; i < workers; i++ {
		got := <-results
		if got.err != nil {
			t.Errorf("Concurrent send: %v", got.err)
			continue
		}
		if expectedID == 0 {
			expectedID = got.id
		}
		if got.id != expectedID {
			t.Errorf("IDs differ: %d vs %d", got.id, expectedID)
		}
	}
	var count int
	if err := db.QueryRow(context.Background(), "SELECT count(*) FROM messages WHERE conversation_id=$1", fixture.ConversationID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 || notifier.count() != 1 {
		t.Fatalf("Expected one save/broadcast; got %d/%d", count, notifier.count())
	}
}
