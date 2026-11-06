package models

import "time"

type Message struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	SenderID       int64     `json:"sender_id"`
	SenderUsername string    `json:"sender_username"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateMessageRequest struct {
	Content string `json:"content"`
}

// MessagePage contains messages in ascending ID order, ready for display.
type MessagePage struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"has_more"`
}
