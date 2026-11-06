import { useEffect, useState, useCallback, useRef } from "react";

import { useAuth } from "../hooks/useAuth";
import { useChatSocket } from "../hooks/useChatSocket";
import { createConversation, getConversations } from "../services/conversations";

import ConversationSidebar from "../components/ConversationSidebar";
import MessagePanel from "../components/MessagePanel";

import type { Message } from "../types/message";
import type { Conversation } from "../types/conversation";
import type { User } from "../types/auth";

import "../styles/chat.css";

interface ComposerState {
  content: string;
  clientId: string | null;
  sending: boolean;
  error: string;
}

const EMPTY_COMPOSER: ComposerState = { content: "", clientId: null, sending: false, error: "" };

export default function ChatPage() {
  const { user, token, logout } = useAuth();
  const [incomingMessages, setIncomingMessages] = useState<Message[]>([]);

  const handleNewMessage = useCallback((message: Message) => {
    setIncomingMessages((previous) => {
      if (previous.some((item) => item.id === message.id)) {
        return previous;
      }

      return [...previous, message];
    });
  }, []);

  const [conversationRefresh, setConversationRefresh] = useState(0);
  const refreshConversations = useCallback(() => {
    setConversationRefresh((previous) => previous + 1);
  }, []);

  const { status, sendMessage } = useChatSocket({
    token,
    onNewMessage: handleNewMessage,
    onConversationsChanged: refreshConversations,
  });

  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [selectedConversation, setSelectedConversation] =
    useState<Conversation | null>(null);
  const [showConversationList, setShowConversationList] = useState(true);
  const [composers, setComposers] = useState<Record<number, ComposerState>>({});
  const pendingSends = useRef(new Set<number>());

  function updateComposer(conversationId: number, changes: Partial<ComposerState>) {
    setComposers((previous) => ({
      ...previous,
      [conversationId]: { ...(previous[conversationId] ?? EMPTY_COMPOSER), ...changes },
    }));
  }

  async function handleSendMessage(conversationId: number, content: string) {
    if (pendingSends.current.has(conversationId)) return;

    pendingSends.current.add(conversationId);
    try {
      const clientId = composers[conversationId]?.clientId ?? crypto.randomUUID();
      updateComposer(conversationId, { sending: true, error: "", clientId });
      await sendMessage(conversationId, content.trim(), clientId);
      setComposers((previous) => {
        const current = previous[conversationId] ?? EMPTY_COMPOSER;
        return {
          ...previous,
          [conversationId]: {
            ...current,
            // An acknowledgment belongs only to the draft that was submitted.
            content: current.content === content ? "" : current.content,
            clientId: current.clientId === clientId ? null : current.clientId,
            sending: false,
          },
        };
      });
    } catch (err) {
      updateComposer(conversationId, {
        sending: false,
        error: err instanceof Error ? err.message : "Failed to send message",
      });
    } finally {
      pendingSends.current.delete(conversationId);
    }
  }

  const composer = selectedConversation
    ? composers[selectedConversation.id] ?? EMPTY_COMPOSER
    : EMPTY_COMPOSER;

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!token) return;

    let cancelled = false;

    async function loadConversations() {
      try {
        const data = await getConversations(token!);

        if (!cancelled) {
          // Preserve locally created items if a list request started before creation.
          setConversations((previous) => {
            const fetched = data ?? [];
            const fetchedIds = new Set(fetched.map((conversation) => conversation.id));
            return [
              ...previous.filter((conversation) => !fetchedIds.has(conversation.id)),
              ...fetched,
            ];
          });
          setError("");
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to load conversations"
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadConversations();

    return () => {
      cancelled = true;
    };
  }, [token, conversationRefresh]);

  function handleSelectConversation(conversation: Conversation) {
    setSelectedConversation(conversation);
    setShowConversationList(false);
  }

  async function handleStartConversation(otherUser: User) {
    if (!token) {
      throw new Error("Please log in to start a conversation");
    }

    const result = await createConversation(token, otherUser.id);
    const conversation: Conversation = {
      ...result,
      other_username: otherUser.username,
    };

    setConversations((previous) => {
      if (previous.some((item) => item.id === conversation.id)) {
        return previous;
      }
      return [conversation, ...previous];
    });
    handleSelectConversation(conversation);
  }

  return (
    <div className="chat-app">
      <header className="chat-header">
        <h1>Go Chat</h1>

        <div className="chat-user">
          <span>{user?.username}</span>

          <span className="connection-status">
            Chat connection: {status}
          </span>

          <button onClick={logout} className="logout-button">
            Logout
          </button>
        </div>
      </header>

      <div className={`chat-layout${selectedConversation && !showConversationList ? " chat-layout--conversation-open" : ""}`}>
        <ConversationSidebar
          conversations={conversations}
          selectedConversationId={selectedConversation?.id ?? null}
          onSelectConversation={handleSelectConversation}
          onStartConversation={handleStartConversation}
          loading={loading}
          error={error}
          onRetry={refreshConversations}
        />

        <main className="chat-panel">
          {loading ? (
            <p className="message-status" role="status">Loading conversations...</p>
          ) : selectedConversation ? (
              <MessagePanel
                key={selectedConversation.id}
                conversation={selectedConversation}
                incomingMessages={incomingMessages}
                onSendMessage={handleSendMessage}
                draft={composer.content}
                sending={composer.sending}
                sendError={composer.error}
                onDraftChange={(content) => updateComposer(selectedConversation.id, { content, clientId: null, error: "" })}
                socketConnected={status === "connected"}
                onBack={() => setShowConversationList(true)}
              />
            ) : (
            <div className="chat-placeholder">
              <h2>Welcome to Go Chat</h2>
              <p>Select a conversation to start chatting.</p>
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
