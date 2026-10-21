package websocket

import (
	"context"

	"github.com/coder/websocket"
)

func (c *Client) WritePump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case message := <-c.Send:
			if err := c.Conn.Write(ctx, websocket.MessageText, message); err != nil {
				return
			}
		}
	}
}
