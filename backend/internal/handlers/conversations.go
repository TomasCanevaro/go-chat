package handlers

import (
	"encoding/json"
	"net/http"
	"errors"
	"strconv"

	"go-chat/backend/internal/auth"
	"go-chat/backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"
)

type ConversationHandler struct {
	DB *pgxpool.Pool
}

func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateConversationRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	if req.UserID == userID {
		http.Error(w, "Cannot create a conversation with yourself", http.StatusBadRequest)
		return
	}

	firstID := userID
	secondID := req.UserID

	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	directKey := strconv.FormatInt(firstID, 10) +
		":" +
		strconv.FormatInt(secondID, 10)
	
	var exists bool

	err = h.DB.QueryRow(
		r.Context(),
		`
		SELECT EXISTS(
			SELECT 1
			FROM users
			WHERE id = $1
		)
		`,
		req.UserID,
	).Scan(&exists)

	if err != nil {
		http.Error(w, "Failed to check user", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var existing models.Conversation

	err = h.DB.QueryRow(
		r.Context(),
		`
		SELECT id, type, created_at
		FROM conversations
		WHERE direct_key = $1
		`,
		directKey,
	).Scan(
		&existing.ID,
		&existing.Type,
		&existing.CreatedAt,
	)

	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(existing)
		return
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Failed to check existing conversation", http.StatusInternalServerError)
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	defer tx.Rollback(r.Context())

	var conversation models.Conversation

	err = tx.QueryRow(
		r.Context(),
		`
		INSERT INTO conversations (type, direct_key)
		VALUES ('direct', $1)
		RETURNING id, type, created_at
		`,
		directKey,
	).Scan(
		&conversation.ID,
		&conversation.Type,
		&conversation.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			tx.Rollback(r.Context())

			err = h.DB.QueryRow(
				r.Context(),
				`
				SELECT id, type, created_at
				FROM conversations
				WHERE direct_key = $1
				`,
				directKey,
			).Scan(
				&existing.ID,
				&existing.Type,
				&existing.CreatedAt,
			)

			if err != nil {
				http.Error(w, "Failed to fetch conversation", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(existing)
			return
		}

		http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(
		r.Context(),
		`
		INSERT INTO conversation_members (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		`,
		conversation.ID,
		userID,
		req.UserID,
	)

	if err != nil {
		http.Error(w, "Failed to add conversation members", http.StatusInternalServerError)
		return
	}

	err = tx.Commit(r.Context())
	if err != nil {
		http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conversation)
}