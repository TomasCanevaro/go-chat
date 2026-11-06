import type { Message } from "../types/message";

interface MessageBubbleProps {
  message: Message;
  isOwnMessage: boolean;
}

export default function MessageBubble({
  message,
  isOwnMessage,
}: MessageBubbleProps) {
  const time = new Date(message.created_at).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });

  return (
    <div
      data-message-id={message.id}
      className={`message-row ${isOwnMessage ? "own" : "other"}`}
    >
      <div className="message-bubble">
        <p className="message-content">{message.content}</p>
        <span className="message-time">{time}</span>
      </div>
    </div>
  );
}