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
  onConversationsChanged: () => void;
}

interface PendingMessage {
  resolve: () => void;
  reject: (error: Error) => void;
  timer: ReturnType<typeof setTimeout>;
}

const ACK_TIMEOUT = 10000;
const MAX_RECONNECT_DELAY = 10000;
const INITIAL_RECONNECT_DELAY = 1000;

export function useChatSocket({
  token,
  onNewMessage,
  onConversationsChanged,
}: ChatSocketOptions) {
  const [status, setStatus] = useState<SocketStatus>(
    token ? "connecting" : "disconnected"
  );

  const socketRef = useRef<WebSocket | null>(null);
  const authenticatedRef = useRef(false);

  const pendingMessagesRef = useRef(new Map<string, PendingMessage>());

  useEffect(() => {
    if (!token) return;

    const pendingMessages = pendingMessagesRef.current;

    function rejectPendingMessages() {
      for (const pending of pendingMessages.values()) {
        clearTimeout(pending.timer);
        pending.reject(new Error(
          "Connection closed before confirmation. Your message may have been saved; check the conversation before retrying."
        ));
      }
      pendingMessages.clear();
    }

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
              onConversationsChanged();
              break;

            case "conversation_created":
              if (Number.isSafeInteger(data.conversation_id) && data.conversation_id > 0) {
                onConversationsChanged();
              }
              break;

            case "new_message":
              if (data.message) {
                onNewMessage(data.message as Message);
              }
              break;

            case "message_ack":
            case "message_error": {
              if (typeof data.client_id !== "string") break;
              if (data.type === "message_ack" &&
                  (!Number.isSafeInteger(data.message_id) || data.message_id <= 0)) break;

              const pending = pendingMessages.get(data.client_id);
              // Ignore responses for sends that already timed out or settled.
              if (!pending) break;

              clearTimeout(pending.timer);
              pendingMessages.delete(data.client_id);
              if (data.type === "message_ack") {
                pending.resolve();
              } else {
                pending.reject(new Error(
                  typeof data.message === "string" ? data.message : "Failed to send message"
                ));
              }
              break;
            }

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

        rejectPendingMessages();
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
      rejectPendingMessages();

      socketRef.current?.close();
      socketRef.current = null;
    };
  }, [token, onNewMessage, onConversationsChanged]);

  const sendMessage = useCallback(
    async (conversationId: number, content: string): Promise<void> => {
      const socket = socketRef.current;

      if (
        !socket ||
        socket.readyState !== WebSocket.OPEN ||
        !authenticatedRef.current
      ) {
        throw new Error("Chat connection is unavailable");
      }

      const clientId = crypto.randomUUID();
      const pendingMessages = pendingMessagesRef.current;

      return new Promise<void>((resolve, reject) => {
        const timer = setTimeout(() => {
          pendingMessages.delete(clientId);
          reject(new Error(
            "No confirmation received. Your message may have been saved; check the conversation before retrying."
          ));
        }, ACK_TIMEOUT);

        // Register before sending so a response can always find its request.
        pendingMessages.set(clientId, { resolve, reject, timer });

        try {
          socket.send(JSON.stringify({
            type: "send_message",
            client_id: clientId,
            conversation_id: conversationId,
            content,
          }));
        } catch (error) {
          clearTimeout(timer);
          pendingMessages.delete(clientId);
          reject(error instanceof Error ? error : new Error("Failed to send message"));
        }
      });
    },
    []
  );

  return { status, sendMessage };
}