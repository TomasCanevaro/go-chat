package models

import "time"

type Conversation struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateConversationRequest struct {
	UserID int64 `json:"user_id"`
}

type ConversationListItem struct {
	ID            int64     `json:"id"`
	Type          string    `json:"type"`
	OtherUserID   int64     `json:"other_user_id"`
	OtherUsername string    `json:"other_username"`
	CreatedAt     time.Time `json:"created_at"`
}
