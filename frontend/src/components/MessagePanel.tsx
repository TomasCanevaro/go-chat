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
  onSendMessage: (conversationId: number, content: string) => void;
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
    if (!token) return;

    let cancelled = false;

    async function loadMessages() {
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

    onSendMessage(conversation.id, content);
  }

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
        ) : visibleMessages.length === 0 ? (
          <p className="message-status">
            No messages yet. Start the conversation!
          </p>
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
        disabled={loading || Boolean(error) || !socketConnected}
      />
    </section>
  );
}