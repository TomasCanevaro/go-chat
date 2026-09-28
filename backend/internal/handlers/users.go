package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"go-chat/backend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserHandler struct {
	DB *pgxpool.Pool
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(
		context.Background(),
		`
		SELECT id, username, email, created_at
		FROM users
		ORDER BY id
		`,
	)

	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []models.User{}

	for rows.Next() {
		var user models.User

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.CreatedAt,
		)

		if err != nil {
			http.Error(w, "Failed to read user", http.StatusInternalServerError)
			return
		}

		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(users)
}