package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go-chat/backend/internal/auth"
	"go-chat/backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	if req.UserID == userID {
		http.Error(
			w,
			"Cannot create a conversation with yourself",
			http.StatusBadRequest,
		)
		return
	}

	exists, err := h.userExists(r.Context(), req.UserID)
	if err != nil {
		http.Error(w, "Failed to check user", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	directKey := makeDirectKey(userID, req.UserID)

	conversation, created, err := h.getOrCreateDirectConversation(
		r.Context(),
		userID,
		req.UserID,
		directKey,
	)

	if err != nil {
		http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if created {
		w.WriteHeader(http.StatusCreated)
	}

	json.NewEncoder(w).Encode(conversation)
}

func (h *ConversationHandler) userExists(
	ctx context.Context,
	userID int64,
) (bool, error) {
	var exists bool

	err := h.DB.QueryRow(
		ctx,
		`
		SELECT EXISTS(
			SELECT 1
			FROM users
			WHERE id = $1
		)
		`,
		userID,
	).Scan(&exists)

	return exists, err
}

func makeDirectKey(userID1, userID2 int64) string {
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	return strconv.FormatInt(userID1, 10) +
		":" +
		strconv.FormatInt(userID2, 10)
}

func (h *ConversationHandler) findDirectConversation(
	ctx context.Context,
	directKey string,
) (*models.Conversation, error) {
	var conversation models.Conversation

	err := h.DB.QueryRow(
		ctx,
		`
		SELECT id, type, created_at
		FROM conversations
		WHERE direct_key = $1
		`,
		directKey,
	).Scan(
		&conversation.ID,
		&conversation.Type,
		&conversation.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (h *ConversationHandler) createDirectConversation(
	ctx context.Context,
	userID int64,
	otherUserID int64,
	directKey string,
) (*models.Conversation, error) {
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	var conversation models.Conversation

	err = tx.QueryRow(
		ctx,
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
		return nil, err
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO conversation_members (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		`,
		conversation.ID,
		userID,
		otherUserID,
	)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (h *ConversationHandler) getOrCreateDirectConversation(
	ctx context.Context,
	userID int64,
	otherUserID int64,
	directKey string,
) (*models.Conversation, bool, error) {
	conversation, err := h.findDirectConversation(ctx, directKey)

	if err == nil {
		return conversation, false, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	conversation, err = h.createDirectConversation(
		ctx,
		userID,
		otherUserID,
		directKey,
	)

	if err == nil {
		return conversation, true, nil
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		conversation, err = h.findDirectConversation(ctx, directKey)

		if err != nil {
			return nil, false, err
		}

		return conversation, false, nil
	}

	return nil, false, err
}
