import { useEffect, useState } from "react";

import { useAuth } from "../hooks/useAuth";
import { getConversations } from "../services/conversations";

import ConversationSidebar from "../components/ConversationSidebar";

import type { Conversation } from "../types/conversation";

import "../styles/chat.css";

export default function ChatPage() {
  const { user, token, logout } = useAuth();

  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [selectedConversation, setSelectedConversation] =
    useState<Conversation | null>(null);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!token) return;

    let cancelled = false;

    async function loadConversations() {
      try {
        const data = await getConversations(token!);

        if (!cancelled) {
          setConversations(data ?? []);
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
  }, [token]);

  return (
    <div className="chat-app">
      <header className="chat-header">
        <h1>Go Chat</h1>

        <div className="chat-user">
          <span>{user?.username}</span>

          <button onClick={logout} className="logout-button">
            Logout
          </button>
        </div>
      </header>

      <div className="chat-layout">
        <ConversationSidebar
          conversations={conversations}
          selectedConversationId={selectedConversation?.id ?? null}
          onSelectConversation={setSelectedConversation}
        />

        <main className="chat-panel">
          {loading ? (
            <p>Loading conversations...</p>
          ) : error ? (
            <p role="alert">{error}</p>
          ) : selectedConversation ? (
            <div>
              <h2>{selectedConversation.other_username}</h2>
              <p>Messages will appear here.</p>
            </div>
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