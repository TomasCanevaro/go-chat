package websocket

import (
	"log"
	"sync"

	"github.com/coder/websocket"
)

type Client struct {
	UserID int64
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[int64]map[*Client]struct{}),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]struct{})
	}

	h.clients[client.UserID][client] = struct{}{}

	log.Printf(
		"User %d connected (%d active connections)",
		client.UserID,
		len(h.clients[client.UserID]),
	)
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	connections, exists := h.clients[client.UserID]
	if !exists {
		return
	}

	delete(connections, client)

	if len(connections) == 0 {
		delete(h.clients, client.UserID)
	}

	log.Printf(
		"User %d disconnected (%d active connections)",
		client.UserID,
		len(connections),
	)
}

func (h *Hub) ConnectionCount(userID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients[userID])
}

func (h *Hub) SendToUser(userID int64, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[userID] {
		select {
		case client.Send <- message:
		default:
			// The client isn't consuming messages fast enough.
			// We'll handle slow-client disconnection later.
		}
	}
}
