package websocket

import (
	"sync"
	"testing"
)

func newTestClient(userID int64) *Client {
	return &Client{
		UserID: userID,
		Send:   make(chan []byte, 16),
	}
}

func TestHubRegisterAndUnregister(t *testing.T) {
	hub := NewHub()

	client := newTestClient(1)

	hub.Register(client)

	if got := hub.ConnectionCount(1); got != 1 {
		t.Errorf("Expected 1 connection, got %d", got)
	}

	hub.Unregister(client)

	if got := hub.ConnectionCount(1); got != 0 {
		t.Errorf("Expected 0 connections, got %d", got)
	}
}

func TestHubSendToUser(t *testing.T) {
	hub := NewHub()

	alice := newTestClient(1)
	bob := newTestClient(2)

	hub.Register(alice)
	hub.Register(bob)

	message := []byte(`{"type":"new_message","content":"Hello Bob!"}`)

	hub.SendToUser(2, message)

	select {
	case received := <-bob.Send:
		if string(received) != string(message) {
			t.Errorf(
				"Expected %s, got %s",
				message,
				received,
			)
		}
	default:
		t.Fatal("Bob did not receive the message")
	}

	select {
	case <-alice.Send:
		t.Fatal("Alice received a message intended for Bob")
	default:
		// Correct: Alice received nothing.
	}
}

func TestHubMultipleConnections(t *testing.T) {
	hub := NewHub()

	bobTab1 := newTestClient(2)
	bobTab2 := newTestClient(2)

	hub.Register(bobTab1)
	hub.Register(bobTab2)

	if got := hub.ConnectionCount(2); got != 2 {
		t.Fatalf("Expected 2 connections, got %d", got)
	}

	message := []byte(`{"type":"new_message","content":"Hello Bob!"}`)

	hub.SendToUser(2, message)

	for i, client := range []*Client{bobTab1, bobTab2} {
		select {
		case received := <-client.Send:
			if string(received) != string(message) {
				t.Errorf(
					"Tab %d received incorrect message: %s",
					i+1,
					received,
				)
			}
		default:
			t.Errorf("Tab %d did not receive the message", i+1)
		}
	}

	hub.Unregister(bobTab1)

	if got := hub.ConnectionCount(2); got != 1 {
		t.Errorf("Expected 1 remaining connection, got %d", got)
	}
}

func TestHubConcurrentConnections(t *testing.T) {
	hub := NewHub()

	const clientCount = 100

	var wg sync.WaitGroup

	for i := 0; i < clientCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			client := newTestClient(1)

			hub.Register(client)
			hub.SendToUser(1, []byte("test"))
			hub.Unregister(client)
		}()
	}

	wg.Wait()

	if got := hub.ConnectionCount(1); got != 0 {
		t.Errorf("Expected 0 connections, got %d", got)
	}
}
