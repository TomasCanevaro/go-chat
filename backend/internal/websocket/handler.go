package websocket

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
)

func Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:*"},
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	for {
		messageType, data, err := conn.Read(context.Background())
		if err != nil {
			return
		}

		err = conn.Write(
			context.Background(),
			messageType,
			data,
		)
		if err != nil {
			return
		}
	}
}
