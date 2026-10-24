package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"go-chat/backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEmptyMessage         = errors.New("message cannot be empty")
	ErrMessageTooLong       = errors.New("message is too long")
	ErrConversationNotFound = errors.New("conversation not found")
)

type MessageNotifier interface {
	SendToUser(userID int64, message []byte)
}

type MessageService struct {
	DB       *pgxpool.Pool
	Notifier MessageNotifier
}

func (s *MessageService) Create(
	ctx context.Context,
	conversationID int64,
	senderID int64,
	content string,
) (*models.Message, error) {
	content, err := ValidateMessageContent(content)
	if err != nil {
		return nil, err
	}

	var message models.Message

	err = s.DB.QueryRow(
		ctx,
		`
		WITH inserted AS (
			INSERT INTO messages (conversation_id, sender_id, content)
			SELECT $1, $2, $3
			WHERE EXISTS (
				SELECT 1
				FROM conversation_members
				WHERE conversation_id = $1
				  AND user_id = $2
			)
			RETURNING id, conversation_id, sender_id, content, created_at
		)
		SELECT
			inserted.id,
			inserted.conversation_id,
			inserted.sender_id,
			u.username,
			inserted.content,
			inserted.created_at
		FROM inserted
		JOIN users u ON u.id = inserted.sender_id
		`,
		conversationID,
		senderID,
		content,
	).Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.SenderUsername,
		&message.Content,
		&message.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	if s.Notifier != nil {
		if err := s.notifyConversationMembers(ctx, message); err != nil {
			// Message is already persisted. Delivery is best-effort.
			// The caller still receives the successfully saved message.
			// We'll add structured logging later.
			fmt.Printf("Failed to notify conversation members: %v\n", err)
		}
	}

	return &message, nil
}

func ValidateMessageContent(content string) (string, error) {
	content = strings.TrimSpace(content)

	if content == "" {
		return "", ErrEmptyMessage
	}

	if utf8.RuneCountInString(content) > 4000 {
		return "", ErrMessageTooLong
	}

	return content, nil
}

func (s *MessageService) notifyConversationMembers(
	ctx context.Context,
	message models.Message,
) error {
	rows, err := s.DB.Query(
		ctx,
		`
		SELECT user_id
		FROM conversation_members
		WHERE conversation_id = $1
		`,
		message.ConversationID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var userIDs []int64

	for rows.Next() {
		var userID int64

		if err := rows.Scan(&userID); err != nil {
			return err
		}

		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	event, err := json.Marshal(struct {
		Type    string         `json:"type"`
		Message models.Message `json:"message"`
	}{
		Type:    "new_message",
		Message: message,
	})
	if err != nil {
		return err
	}

	for _, userID := range userIDs {
		s.Notifier.SendToUser(userID, event)
	}

	return nil
}
