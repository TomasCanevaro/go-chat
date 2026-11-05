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

const MAX_RECONNECT_DELAY = 10000;
const INITIAL_RECONNECT_DELAY = 1000;

export function useChatSocket({
  token,
  onNewMessage,
}: ChatSocketOptions) {
  const [status, setStatus] = useState<SocketStatus>(
    token ? "connecting" : "disconnected"
  );

  const socketRef = useRef<WebSocket | null>(null);
  const authenticatedRef = useRef(false);

  useEffect(() => {
    if (!token) return;

    let active = true;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let reconnectAttempts = 0;

    const protocol =
      window.location.protocol === "https:" ? "wss:" : "ws:";

    const url = `${protocol}//${window.location.host}/ws`;

    function connect() {
      if (!active) return;

      const socket = new WebSocket(url);

      socketRef.current = socket;
      authenticatedRef.current = false;

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
              authenticatedRef.current = true;
              reconnectAttempts = 0;
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
        if (active) {
          setStatus("error");
        }
      };

      socket.onclose = () => {
        if (!active) return;

        socketRef.current = null;
        authenticatedRef.current = false;

        setStatus("disconnected");

        const delay = Math.min(
          INITIAL_RECONNECT_DELAY * 2 ** reconnectAttempts,
          MAX_RECONNECT_DELAY
        );

        reconnectAttempts += 1;

        reconnectTimer = setTimeout(() => {
          if (active) {
            setStatus("connecting");
            connect();
          }
        }, delay);
      };
    }

    connect();

    return () => {
      active = false;

      if (reconnectTimer !== null) {
        clearTimeout(reconnectTimer);
      }

      authenticatedRef.current = false;

      socketRef.current?.close();
      socketRef.current = null;
    };
  }, [token, onNewMessage]);

  const sendMessage = useCallback(
    (conversationId: number, content: string) => {
      const socket = socketRef.current;

      if (
        !socket ||
        socket.readyState !== WebSocket.OPEN ||
        !authenticatedRef.current
      ) {
        throw new Error("Chat connection is unavailable");
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