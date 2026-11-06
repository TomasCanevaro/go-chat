import { type FormEvent, type KeyboardEvent } from "react";

interface MessageInputProps {
  content: string;
  sending: boolean;
  error: string;
  onContentChange: (content: string) => void;
  onSendMessage: (content: string) => Promise<void>;
  disabled?: boolean;
}

export default function MessageInput({
  content,
  sending,
  error,
  onContentChange,
  onSendMessage,
  disabled = false,
}: MessageInputProps) {
  const isEmpty = content.trim().length === 0;
  const isTooLong = Array.from(content).length > 4000;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (isEmpty || isTooLong || sending || disabled) {
      return;
    }

    await onSendMessage(content);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault();
      event.currentTarget.form?.requestSubmit();
    }
  }

  return (
    <form className="message-input-form" onSubmit={handleSubmit}>
      <div className="message-input-container">
        <textarea
          className="message-input"
          placeholder="Type a message..."
          value={content}
          onChange={(event) => onContentChange(event.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled || sending}
          rows={1}
          aria-label="Message"
        />

        <button
          type="submit"
          className="send-button"
          disabled={disabled || sending || isEmpty || isTooLong}
        >
          {sending ? "Sending..." : "Send"}
        </button>
      </div>

      {isTooLong && (
        <p className="message-input-error" role="alert">
          Messages cannot exceed 4000 characters.
        </p>
      )}

      {error && (
        <p className="message-input-error" role="alert">
          {error}
        </p>
      )}
    </form>
  );
}