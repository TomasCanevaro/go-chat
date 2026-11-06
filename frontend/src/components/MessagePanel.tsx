import { useEffect, useLayoutEffect, useRef, useState } from "react";

import { useAuth } from "../hooks/useAuth";
import { getMessages } from "../services/messages";

import MessageBubble from "./MessageBubble";
import MessageInput from "./MessageInput";

import type { Conversation } from "../types/conversation";
import type { Message } from "../types/message";

function mergeMessages(previous: Message[], incoming: Message[]): Message[] {
  const byId = new Map(previous.map((message) => [message.id, message]));
  for (const message of incoming) byId.set(message.id, message);
  return [...byId.values()].sort((a, b) => a.id - b.id);
}

interface MessagePanelProps {
  conversation: Conversation;
  incomingMessages: Message[];
  onSendMessage: (conversationId: number, content: string) => Promise<void>;
  draft: string;
  sending: boolean;
  sendError: string;
  onDraftChange: (content: string) => void;
  socketConnected: boolean;
  onBack: () => void;
}

export default function MessagePanel({
  conversation,
  incomingMessages,
  onSendMessage,
  draft,
  sending,
  sendError,
  onDraftChange,
  socketConnected,
  onBack,
}: MessagePanelProps) {
  const { user, token } = useAuth();

  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [retryAttempt, setRetryAttempt] = useState(0);

  const [hasOlder, setHasOlder] = useState(false);
  const [loadingOlder, setLoadingOlder] = useState(false);
  const [olderError, setOlderError] = useState("");
  const listRef = useRef<HTMLDivElement | null>(null);
  const lastSyncedId = useRef<number | null>(null);
  const oldestLoadedId = useRef<number | null>(null);
  const olderRequest = useRef<AbortController | null>(null);
  const scrollAnchor = useRef<{ id: string; top: number } | null>(null);
  const followLatest = useRef(true);
  const renderedNewestId = useRef<number | null>(null);

  const visibleMessages = [
    ...messages,
    ...incomingMessages.filter(
      (incoming) =>
        incoming.conversation_id === conversation.id &&
        (messages.length === 0 || incoming.id >= messages[0].id) &&
        !messages.some((message) => message.id === incoming.id)
    ),
  ].sort((a, b) => a.id - b.id);

  const newestVisibleId = visibleMessages.at(-1)?.id ?? 0;

  useEffect(() => {
    if (!token || !socketConnected) return;

    const controller = new AbortController();

    async function loadMessages() {
      try {
        const target = lastSyncedId.current;
        let page = await getMessages(token!, conversation.id, undefined, controller.signal);
        const newestId = page.messages.at(-1)?.id ?? target ?? 0;
        let recovered = page.messages;

        // Reconnect may span several pages. Stop only after reaching the last
        // successful sync, rather than assuming the newest page fills the gap.
        while (target !== null && page.has_more && page.messages.length > 0 &&
               page.messages[0].id > target) {
          page = await getMessages(token!, conversation.id, page.messages[0].id, controller.signal);
          recovered = [...page.messages, ...recovered];
        }

        if (controller.signal.aborted) return;
        if (oldestLoadedId.current === null) {
          oldestLoadedId.current = recovered[0]?.id ?? null;
          setHasOlder(page.has_more);
        }
        lastSyncedId.current = newestId;
        setMessages((previous) => mergeMessages(previous, recovered));
        setError("");
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(err instanceof Error ? err.message : "Failed to synchronize messages");
        }
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }

    loadMessages();

    return () => {
      controller.abort();
      olderRequest.current?.abort();
    };
  }, [conversation.id, token, socketConnected, retryAttempt]);

  async function loadOlderMessages() {
    if (!token || !socketConnected || !hasOlder || olderRequest.current ||
        oldestLoadedId.current === null) return;

    const controller = new AbortController();
    olderRequest.current = controller;
    setLoadingOlder(true);
    setOlderError("");
    try {
      const page = await getMessages(token, conversation.id, oldestLoadedId.current, controller.signal);
      if (controller.signal.aborted) return;

      const list = listRef.current;
      if (list) {
        const top = list.getBoundingClientRect().top;
        const row = Array.from(list.querySelectorAll<HTMLElement>("[data-message-id]"))
          .find((element) => element.getBoundingClientRect().bottom > top);
        if (row) scrollAnchor.current = {
          id: row.dataset.messageId!,
          top: row.getBoundingClientRect().top,
        };
      }
      oldestLoadedId.current = page.messages[0]?.id ?? oldestLoadedId.current;
      setHasOlder(page.has_more);
      setMessages((previous) => mergeMessages(previous, page.messages));
    } catch (err) {
      if (!controller.signal.aborted) {
        setOlderError(err instanceof Error ? err.message : "Failed to load older messages");
      }
    } finally {
      olderRequest.current = null;
      setLoadingOlder(false);
    }
  }

  useLayoutEffect(() => {
    const list = listRef.current;
    if (!list || loading) return;

    const anchor = scrollAnchor.current;
    if (anchor) {
      const row = list.querySelector<HTMLElement>(`[data-message-id="${anchor.id}"]`);
      if (row) list.scrollTop += row.getBoundingClientRect().top - anchor.top;
      scrollAnchor.current = null;
    } else if (renderedNewestId.current === null ||
               (newestVisibleId !== renderedNewestId.current && followLatest.current)) {
      list.scrollTop = list.scrollHeight;
    }
    renderedNewestId.current = newestVisibleId;
  }, [messages, loading, newestVisibleId]);

  return (
    <section className="message-panel">
      <header className="message-panel-header">
        <button
          type="button"
          className="logout-button conversation-back-button"
          onClick={onBack}
        >
          Back to conversations
        </button>
        <h2>{conversation.other_username}</h2>
      </header>

      <div
        className="message-list"
        ref={listRef}
        onScroll={() => {
          const list = listRef.current;
          if (list) followLatest.current = list.scrollHeight - list.scrollTop - list.clientHeight < 80;
        }}
      >
        {hasOlder && (
          <div className="message-history-controls">
            <button
              type="button"
              className="logout-button"
              disabled={loadingOlder || !socketConnected}
              onClick={loadOlderMessages}
            >
              {loadingOlder ? "Loading older messages..." : "Load older messages"}
            </button>
          </div>
        )}
        {olderError && (
          <div className="chat-feedback" role="alert">
            <p>Could not load older messages: {olderError}</p>
            <button
              type="button"
              className="logout-button"
              disabled={loadingOlder || !socketConnected}
              onClick={loadOlderMessages}
            >
              Retry older messages
            </button>
          </div>
        )}
        {error && (
          <div className="chat-feedback" role="alert">
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

      </div>

      <MessageInput
        content={draft}
        sending={sending}
        error={sendError}
        onContentChange={onDraftChange}
        onSendMessage={(content) => onSendMessage(conversation.id, content)}
        disabled={loading || !socketConnected}
      />
    </section>
  );
}
