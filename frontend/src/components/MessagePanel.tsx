import { useEffect, useState } from "react";

import { useAuth } from "../hooks/useAuth";
import { getMessages } from "../services/messages";

import MessageBubble from "./MessageBubble";

import type { Conversation } from "../types/conversation";
import type { Message } from "../types/message";

interface MessagePanelProps {
  conversation: Conversation;
}

export default function MessagePanel({
  conversation,
}: MessagePanelProps) {
  const { user, token } = useAuth();

  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!token) return;

    let cancelled = false;

    async function loadMessages() {
      setLoading(true);
      setError("");
      setMessages([]);

      try {
        const data = await getMessages(token!, conversation.id);

        if (!cancelled) {
          setMessages(data ?? []);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to load messages"
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
  }, [conversation.id, token]);

  return (
    <section className="message-panel">
      <header className="message-panel-header">
        <h2>{conversation.other_username}</h2>
      </header>

      <div className="message-list">
        {loading ? (
          <p className="message-status">Loading messages...</p>
        ) : error ? (
          <p className="message-status" role="alert">
            {error}
          </p>
        ) : messages.length === 0 ? (
          <p className="message-status">
            No messages yet. Start the conversation!
          </p>
        ) : (
          messages.map((message) => (
            <MessageBubble
              key={message.id}
              message={message}
              isOwnMessage={message.sender_id === user?.id}
            />
          ))
        )}
      </div>
    </section>
  );
}