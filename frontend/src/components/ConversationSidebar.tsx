import { useState } from "react";

import NewConversationPicker from "./NewConversationPicker";
import type { Conversation } from "../types/conversation";
import type { UserSummary } from "../types/auth";

interface ConversationSidebarProps {
  conversations: Conversation[];
  selectedConversationId: number | null;
  onSelectConversation: (conversation: Conversation) => void;
  onStartConversation: (user: UserSummary) => Promise<void>;
  loading: boolean;
  error: string;
  onRetry: () => void;
}

export default function ConversationSidebar({
  conversations,
  selectedConversationId,
  onSelectConversation,
  onStartConversation,
  loading,
  error,
  onRetry,
}: ConversationSidebarProps) {
  const [showPicker, setShowPicker] = useState(false);

  async function handleSelectUser(user: UserSummary) {
    await onStartConversation(user);
    setShowPicker(false);
  }

  return (
    <aside className="conversation-sidebar">
      <div className="sidebar-header">
        <h2>Conversations</h2>
        {!showPicker && (
          <button
            type="button"
            className="logout-button new-conversation-button"
            disabled={loading}
            onClick={() => setShowPicker(true)}
          >
            New conversation
          </button>
        )}
      </div>

      {error && (
        <div className="chat-feedback sidebar-feedback" role="alert">
          <p>Could not refresh conversations: {error}</p>
          <button type="button" className="logout-button" onClick={onRetry}>
            Retry
          </button>
        </div>
      )}

      {showPicker && (
        <NewConversationPicker
          onSelectUser={handleSelectUser}
          onCancel={() => setShowPicker(false)}
        />
      )}

      {loading ? (
        <p className="empty-conversations" role="status">Loading conversations...</p>
      ) : conversations.length === 0 && !error ? (
        <p className="empty-conversations">
          You don't have any conversations yet.
        </p>
      ) : conversations.length > 0 ? (
        <ul className="conversation-list">
          {conversations.map((conversation) => (
            <li key={conversation.id}>
              <button
                className={`conversation-item ${
                  selectedConversationId === conversation.id
                    ? "selected"
                    : ""
                }`}
                onClick={() => onSelectConversation(conversation)}
              >
                <span className="conversation-avatar">
                  {conversation.other_username.charAt(0).toUpperCase()}
                </span>

                <span className="conversation-name">
                  {conversation.other_username}
                </span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </aside>
  );
}
