package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"go-chat/backend/internal/auth"
	"go-chat/backend/internal/services"

	"github.com/coder/websocket"
)

type Handler struct {
	JWTSecret      string
	Hub            *Hub
	MessageService *services.MessageService
}

type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type SendMessageRequest struct {
	Type           string `json:"type"`
	ConversationID int64  `json:"conversation_id"`
	Content        string `json:"content"`
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
		Send:   make(chan []byte, 16),
	}

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

	writeCtx, cancelWrite := context.WithCancel(r.Context())
	defer cancelWrite()

	go client.WritePump(writeCtx)

	h.Hub.Register(client)
	defer h.Hub.Unregister(client)

	// Keep the connection open until the client disconnects.
	for {
		messageType, data, err := conn.Read(r.Context())
		if err != nil {
			return
		}

		if messageType != websocket.MessageText {
			h.sendError(client, "Only text messages are supported")
			continue
		}

		h.handleMessage(r.Context(), client, data)
	}
}

func (h *Handler) handleMessage(
	ctx context.Context,
	client *Client,
	data []byte,
) {
	var req SendMessageRequest

	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(client, "Invalid JSON")
		return
	}

	switch req.Type {
	case "send_message":
		if req.ConversationID <= 0 {
			h.sendError(client, "Invalid conversation ID")
			return
		}

		_, err := h.MessageService.Create(
			ctx,
			req.ConversationID,
			client.UserID,
			req.Content,
		)

		if err != nil {
			switch {
			case errors.Is(err, services.ErrEmptyMessage):
				h.sendError(client, "Message cannot be empty")

			case errors.Is(err, services.ErrMessageTooLong):
				h.sendError(client, "Message is too long")

			case errors.Is(err, services.ErrConversationNotFound):
				h.sendError(client, "Conversation not found")

			default:
				log.Printf("WebSocket message creation failed: %v", err)
				h.sendError(client, "Failed to send message")
			}
			return
		}

		// MessageService already broadcasts the saved message
		// to every conversation member through the hub.

	default:
		h.sendError(client, "Unknown message type")
	}
}

func (h *Handler) sendError(client *Client, message string) {
	response, err := json.Marshal(Response{
		Type:    "error",
		Message: message,
	})

	if err != nil {
		return
	}

	select {
	case client.Send <- response:
	default:
		log.Printf("WebSocket send queue full for user %d", client.UserID)
	}
}
