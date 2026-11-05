import { useEffect, useRef, useState } from "react";

import { useAuth } from "../hooks/useAuth";
import { getMessages } from "../services/messages";

import MessageBubble from "./MessageBubble";
import MessageInput from "./MessageInput";

import type { Conversation } from "../types/conversation";
import type { Message } from "../types/message";

interface MessagePanelProps {
  conversation: Conversation;
  incomingMessages: Message[];
  onSendMessage: (conversationId: number, content: string) => Promise<void>;
  socketConnected: boolean;
}

export default function MessagePanel({
  conversation,
  incomingMessages,
  onSendMessage,
  socketConnected,
}: MessagePanelProps) {
  const { user, token } = useAuth();

  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [retryAttempt, setRetryAttempt] = useState(0);

  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  const visibleMessages = [
    ...messages,
    ...incomingMessages.filter(
      (incoming) =>
        incoming.conversation_id === conversation.id &&
        !messages.some((message) => message.id === incoming.id)
    ),
  ].sort((a, b) => a.id - b.id);

  useEffect(() => {
    if (!token || !socketConnected) return;

    let cancelled = false;

    async function loadMessages() {
      try {
        const data = await getMessages(token!, conversation.id);

        if (!cancelled) {
          // Keep messages already loaded while merging the refreshed history.
          setMessages((previous) => {
            const byId = new Map(previous.map((message) => [message.id, message]));
            for (const message of data ?? []) {
              byId.set(message.id, message);
            }
            return [...byId.values()];
          });
          setError("");
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to synchronize messages"
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadMessages();

    return () => {
      cancelled = true;
    };
  }, [conversation.id, token, socketConnected, retryAttempt]);

  useEffect(() => {
    if (!loading) {
      messagesEndRef.current?.scrollIntoView({
        behavior: "smooth",
      });
    }
  }, [visibleMessages.length, loading]);

  async function handleSendMessage(content: string) {
    if (!socketConnected) {
      throw new Error("Chat connection is unavailable");
    }

    await onSendMessage(conversation.id, content);
  }

  return (
    <section className="message-panel">
      <header className="message-panel-header">
        <h2>{conversation.other_username}</h2>
      </header>

      <div className="message-list">
        {error && (
          <div role="alert">
            <p>Could not synchronize messages: {error}</p>
            <button
              type="button"
              className="logout-button"
              disabled={!socketConnected}
              onClick={() => setRetryAttempt((previous) => previous + 1)}
            >
              Retry
            </button>
          </div>
        )}

        {loading && visibleMessages.length === 0 ? (
          <p className="message-status">
            {socketConnected ? "Loading messages..." : "Waiting for chat connection..."}
          </p>
        ) : visibleMessages.length === 0 ? (
          !error && (
            <p className="message-status">
              No messages yet. Start the conversation!
            </p>
          )
        ) : (
          visibleMessages.map((message) => (
            <MessageBubble
              key={message.id}
              message={message}
              isOwnMessage={message.sender_id === user?.id}
            />
          ))
        )}

        <div ref={messagesEndRef} />
      </div>

      <MessageInput
        onSendMessage={handleSendMessage}
        disabled={loading || !socketConnected}
      />
    </section>
  );
}