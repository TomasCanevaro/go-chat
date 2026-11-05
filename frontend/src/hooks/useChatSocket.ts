import { useCallback, useEffect, useRef, useState } from "react";
import type { Message } from "../types/message";

export type SocketStatus =
  | "disconnected"
  | "connecting"
  | "connected"
  | "error";

interface ChatSocketOptions {
  token: string | null;
  onNewMessage: (message: Message) => void;
}

export function useChatSocket({
  token,
  onNewMessage,
}: ChatSocketOptions) {
  const [status, setStatus] = useState<SocketStatus>(
    token ? "connecting" : "disconnected"
  );

  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!token) return;

    const protocol =
      window.location.protocol === "https:" ? "wss:" : "ws:";

    const url = `${protocol}//${window.location.host}/ws`;

    const socket = new WebSocket(url);
    socketRef.current = socket;

    let active = true;

    socket.onopen = () => {
      if (!active) return;

      socket.send(
        JSON.stringify({
          type: "auth",
          token,
        })
      );
    };

    socket.onmessage = (event: MessageEvent<string>) => {
      if (!active) return;

      try {
        const data = JSON.parse(event.data);

        switch (data.type) {
          case "auth_success":
            setStatus("connected");
            break;

          case "new_message":
            if (data.message) {
              onNewMessage(data.message as Message);
            }
            break;

          case "error":
            console.error("WebSocket error:", data.message);
            setStatus("error");
            break;
        }
      } catch {
        console.error("Invalid WebSocket message");
      }
    };

    socket.onerror = () => {
      if (active) setStatus("error");
    };

    socket.onclose = () => {
      if (active) setStatus("disconnected");
    };

    return () => {
      active = false;
      socketRef.current = null;
      socket.close();
    };
  }, [token, onNewMessage]);

  const sendMessage = useCallback(
    (conversationId: number, content: string) => {
      const socket = socketRef.current;

      if (!socket || socket.readyState !== WebSocket.OPEN) {
        throw new Error("WebSocket is not connected");
      }

      socket.send(
        JSON.stringify({
          type: "send_message",
          conversation_id: conversationId,
          content,
        })
      );
    },
    []
  );

  return { status, sendMessage };
}