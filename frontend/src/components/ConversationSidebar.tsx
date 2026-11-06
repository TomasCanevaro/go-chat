import { useState } from "react";

import NewConversationPicker from "./NewConversationPicker";
import type { Conversation } from "../types/conversation";
import type { User } from "../types/auth";

interface ConversationSidebarProps {
  conversations: Conversation[];
  selectedConversationId: number | null;
  onSelectConversation: (conversation: Conversation) => void;
  onStartConversation: (user: User) => Promise<void>;
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

  async function handleSelectUser(user: User) {
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
        <div className="sidebar-header" role="alert">
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

      {conversations.length === 0 ? (
        <p className="empty-conversations">
          You don't have any conversations yet.
        </p>
      ) : (
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
      )}
    </aside>
  );
}
