import type { Conversation } from "../types/conversation";

interface ConversationSidebarProps {
  conversations: Conversation[];
  selectedConversationId: number | null;
  onSelectConversation: (conversation: Conversation) => void;
}

export default function ConversationSidebar({
  conversations,
  selectedConversationId,
  onSelectConversation,
}: ConversationSidebarProps) {
  return (
    <aside className="conversation-sidebar">
      <div className="sidebar-header">
        <h2>Conversations</h2>
      </div>

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