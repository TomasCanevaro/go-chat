package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"go-chat/backend/internal/auth"
	"go-chat/backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageHandler struct {
	DB *pgxpool.Pool
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversationID, err := strconv.ParseInt(
		r.PathValue("id"), 10, 64,
	)
	if err != nil || conversationID <= 0 {
		http.Error(w, "Invalid conversation ID", http.StatusBadRequest)
		return
	}

	var isMember bool

	err = h.DB.QueryRow(
		r.Context(),
		`
		SELECT EXISTS (
			SELECT 1
			FROM conversation_members
			WHERE conversation_id = $1
			  AND user_id = $2
		)
		`,
		conversationID,
		userID,
	).Scan(&isMember)

	if err != nil {
		http.Error(w, "Failed to check membership", http.StatusInternalServerError)
		return
	}

	if !isMember {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	rows, err := h.DB.Query(
		r.Context(),
		`
		SELECT
			m.id,
			m.conversation_id,
			m.sender_id,
			u.username,
			m.content,
			m.created_at
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.conversation_id = $1
		ORDER BY m.created_at ASC, m.id ASC
		`,
		conversationID,
	)

	if err != nil {
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	messages := []models.Message{}

	for rows.Next() {
		var message models.Message

		err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.SenderUsername,
			&message.Content,
			&message.CreatedAt,
		)
		if err != nil {
			http.Error(w, "Failed to read message", http.StatusInternalServerError)
			return
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to read messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (h *MessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conversationID, err := strconv.ParseInt(
		r.PathValue("id"), 10, 64,
	)
	if err != nil || conversationID <= 0 {
		http.Error(w, "Invalid conversation ID", http.StatusBadRequest)
		return
	}

	var req models.CreateMessageRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Content = strings.TrimSpace(req.Content)

	if req.Content == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	if len(req.Content) > 4000 {
		http.Error(w, "Message is too long", http.StatusBadRequest)
		return
	}

	var message models.Message

	err = h.DB.QueryRow(
		r.Context(),
		`
		INSERT INTO messages (conversation_id, sender_id, content)
		SELECT $1, $2, $3
		WHERE EXISTS (
			SELECT 1
			FROM conversation_members
			WHERE conversation_id = $1
			  AND user_id = $2
		)
		RETURNING id, conversation_id, sender_id, content, created_at
		`,
		conversationID,
		userID,
		req.Content,
	).Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Content,
		&message.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Failed to create message", http.StatusInternalServerError)
		return
	}

	err = h.DB.QueryRow(
		r.Context(),
		`SELECT username FROM users WHERE id = $1`,
		userID,
	).Scan(&message.SenderUsername)

	if err != nil {
		http.Error(w, "Failed to fetch sender", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}
