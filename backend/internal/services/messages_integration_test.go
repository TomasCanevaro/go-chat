package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Load environment variables from the backend directory.
	_ = godotenv.Load("../../.env")

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	// Prevent accidentally running integration tests against chatdb.
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

func TestCreateMessageIntegration(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	// Create a temporary user.
	var userID int64

	err := db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ('integration_alice', 'integration_alice@example.com', 'test_hash')
		RETURNING id
	`).Scan(&userID)

	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create a temporary conversation.
	var conversationID int64

	err = db.QueryRow(ctx, `
		INSERT INTO conversations (type)
		VALUES ('direct')
		RETURNING id
	`).Scan(&conversationID)

	if err != nil {
		t.Fatalf("Failed to create test conversation: %v", err)
	}

	// Remove test data when finished.
	t.Cleanup(func() {
		cleanupCtx := context.Background()

		_, _ = db.Exec(cleanupCtx,
			"DELETE FROM messages WHERE conversation_id = $1",
			conversationID,
		)

		_, _ = db.Exec(cleanupCtx,
			"DELETE FROM conversation_members WHERE conversation_id = $1",
			conversationID,
		)

		_, _ = db.Exec(cleanupCtx,
			"DELETE FROM conversations WHERE id = $1",
			conversationID,
		)

		_, _ = db.Exec(cleanupCtx,
			"DELETE FROM users WHERE id = $1",
			userID,
		)
	})

	// Add Alice to the conversation.
	_, err = db.Exec(ctx, `
		INSERT INTO conversation_members (conversation_id, user_id)
		VALUES ($1, $2)
	`, conversationID, userID)

	if err != nil {
		t.Fatalf("Failed to add test conversation member: %v", err)
	}

	service := &MessageService{
		DB: db,
	}

	message, err := service.Create(
		ctx,
		conversationID,
		userID,
		"Hello from our integration test!",
	)

	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	if message.Content != "Hello from our integration test!" {
		t.Errorf("Unexpected message content: %s", message.Content)
	}

	if message.SenderID != userID {
		t.Errorf("Expected sender ID %d, got %d", userID, message.SenderID)
	}

	if message.ConversationID != conversationID {
		t.Errorf(
			"Expected conversation ID %d, got %d",
			conversationID,
			message.ConversationID,
		)
	}

	// Verify the message was actually persisted.
	var savedContent string

	err = db.QueryRow(ctx, `
		SELECT content
		FROM messages
		WHERE id = $1
	`, message.ID).Scan(&savedContent)

	if err != nil {
		t.Fatalf("Failed to retrieve saved message: %v", err)
	}

	if savedContent != message.Content {
		t.Errorf(
			"Expected saved content %q, got %q",
			message.Content,
			savedContent,
		)
	}
}

type testConversation struct {
	UserID         int64
	ConversationID int64
}

func createTestConversation(t *testing.T, db *pgxpool.Pool) testConversation {
	t.Helper()

	ctx := context.Background()

	// Unique username and email for this test.
	uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())

	username := "testuser_" + uniqueID
	email := username + "@example.com"

	var userID int64

	err := db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, 'test_hash')
		RETURNING id
	`, username, email).Scan(&userID)

	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Register cleanup immediately so failures don't leave the user behind.
	t.Cleanup(func() {
		if _, err := db.Exec(context.Background(),
			"DELETE FROM users WHERE id = $1", userID,
		); err != nil {
			t.Errorf("Failed to clean up test user: %v", err)
		}
	})

	var conversationID int64

	err = db.QueryRow(ctx, `
		INSERT INTO conversations (type)
		VALUES ('direct')
		RETURNING id
	`).Scan(&conversationID)

	if err != nil {
		t.Fatalf("Failed to create test conversation: %v", err)
	}

	// This cleanup runs before the user cleanup (LIFO order).
	t.Cleanup(func() {
		cleanupCtx := context.Background()

		queries := []string{
			"DELETE FROM messages WHERE conversation_id = $1",
			"DELETE FROM conversation_members WHERE conversation_id = $1",
			"DELETE FROM conversations WHERE id = $1",
		}

		for _, query := range queries {
			if _, err := db.Exec(cleanupCtx, query, conversationID); err != nil {
				t.Errorf("Test cleanup failed: %v", err)
			}
		}
	})

	_, err = db.Exec(ctx, `
		INSERT INTO conversation_members (conversation_id, user_id)
		VALUES ($1, $2)
	`, conversationID, userID)

	if err != nil {
		t.Fatalf("Failed to add test conversation member: %v", err)
	}

	return testConversation{
		UserID:         userID,
		ConversationID: conversationID,
	}
}

func TestCreateMessageUnauthorized(t *testing.T) {
	db := setupTestDB(t)

	fixture := createTestConversation(t, db)

	ctx := context.Background()

	// Create another user who is NOT a conversation member.
	var outsiderID int64

	err := db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, 'test_hash')
		RETURNING id
	`,
		fmt.Sprintf("outsider_%d", time.Now().UnixNano()),
		fmt.Sprintf("outsider_%d@example.com", time.Now().UnixNano()),
	).Scan(&outsiderID)

	if err != nil {
		t.Fatalf("Failed to create outsider user: %v", err)
	}

	t.Cleanup(func() {
		if _, err := db.Exec(context.Background(),
			"DELETE FROM users WHERE id = $1", outsiderID,
		); err != nil {
			t.Errorf("Failed to clean up outsider: %v", err)
		}
	})

	service := &MessageService{
		DB: db,
	}

	message, err := service.Create(
		ctx,
		fixture.ConversationID,
		outsiderID,
		"This message should be rejected",
	)

	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf(
			"Expected ErrConversationNotFound, got %v",
			err,
		)
	}

	if message != nil {
		t.Errorf("Expected nil message, got %+v", message)
	}

	// Confirm nothing was inserted into PostgreSQL.
	var count int

	err = db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM messages
		WHERE conversation_id = $1
	`, fixture.ConversationID).Scan(&count)

	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 messages, found %d", count)
	}
}

func TestCreateMessageNonexistentConversation(t *testing.T) {
	db := setupTestDB(t)

	fixture := createTestConversation(t, db)

	service := &MessageService{
		DB: db,
	}

	message, err := service.Create(
		context.Background(),
		-1,
		fixture.UserID,
		"This conversation doesn't exist",
	)

	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf(
			"Expected ErrConversationNotFound, got %v",
			err,
		)
	}

	if message != nil {
		t.Errorf("Expected nil message, got %+v", message)
	}
}
