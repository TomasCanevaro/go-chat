package websocket

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go-chat/backend/internal/services"
)

func TestHandleMessageErrors(t *testing.T) {
	tests := []struct{ name, request, eventType, clientID, message string }{
		{"invalid conversation", `{"type":"send_message","client_id":"a","conversation_id":0}`, "message_error", "a", "Invalid conversation ID"},
		{"empty content", `{"type":"send_message","client_id":"b","conversation_id":1,"content":"  "}`, "message_error", "b", "Message cannot be empty"},
		{"long content", `{"type":"send_message","client_id":"c","conversation_id":1,"content":"` + strings.Repeat("x", 4001) + `"}`, "message_error", "c", "Message is too long"},
		{"legacy request", `{"type":"send_message","conversation_id":0}`, "error", "", "Invalid conversation ID"},
		{"invalid JSON", `{`, "error", "", "Invalid JSON"},
		{"unknown type", `{"type":"unknown","client_id":"d"}`, "error", "", "Unknown message type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{MessageService: &services.MessageService{}}
			client := newTestClient(1)
			h.handleMessage(context.Background(), client, []byte(tt.request))
			select {
			case data := <-client.Send:
				var got Response
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				if got.Type != tt.eventType || got.ClientID != tt.clientID || got.Message != tt.message || got.MessageID != 0 {
					t.Fatalf("Unexpected response: %+v", got)
				}
			default:
				t.Fatal("Expected error response")
			}
			if len(client.Send) != 0 {
				t.Fatal("Unexpected additional response")
			}
		})
	}
}
