package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go-chat/backend/internal/auth"

	"github.com/coder/websocket"
)

type Handler struct {
	JWTSecret string
	Hub       *Hub
}

type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type Response struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:*"},
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	// The client has 10 seconds to authenticate.
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_, data, err := conn.Read(ctx)
	if err != nil {
		conn.Close(websocket.StatusPolicyViolation, "Authentication required")
		return
	}

	var authMsg AuthMessage

	if err := json.Unmarshal(data, &authMsg); err != nil {
		conn.Close(websocket.StatusPolicyViolation, "Invalid authentication message")
		return
	}

	if authMsg.Type != "auth" || authMsg.Token == "" {
		conn.Close(websocket.StatusPolicyViolation, "Authentication required")
		return
	}

	userID, err := auth.ValidateToken(authMsg.Token, h.JWTSecret)
	if err != nil {
		conn.Close(websocket.StatusPolicyViolation, "Invalid token")
		return
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
	}

	h.Hub.Register(client)
	defer h.Hub.Unregister(client)

	// Authentication succeeded.
	response := Response{
		Type:    "auth_success",
		Message: "Authenticated successfully",
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		return
	}

	if err := conn.Write(ctx, websocket.MessageText, responseData); err != nil {
		return
	}

	// Keep the connection open until the client disconnects.
	for {
		_, _, err := conn.Read(r.Context())
		if err != nil {
			return
		}
	}
}
