import { useEffect, useRef, type FormEvent, type KeyboardEvent } from "react";

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
  const inputRef = useRef<HTMLTextAreaElement | null>(null);
  const restoreFocus = useRef(false);

  useEffect(() => {
    if (sending || disabled || !restoreFocus.current) return;
    restoreFocus.current = false;

    const input = inputRef.current;
    const focused = document.activeElement;
    // Restore focus after clicking Send, but leave deliberate navigation alone.
    if (input && input.getClientRects().length > 0 &&
        (focused === document.body || focused === input ||
         focused === input.form?.querySelector('button[type="submit"]'))) {
      input.focus({ preventScroll: true });
    }
  }, [sending, disabled]);

  const isEmpty = content.trim().length === 0;
  const isTooLong = Array.from(content).length > 4000;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (isEmpty || isTooLong || sending || disabled) {
      return;
    }

    restoreFocus.current = true;
    await onSendMessage(content);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault();
      event.currentTarget.form?.requestSubmit();
    }
  }

  return (
    <form className="message-input-form" onSubmit={handleSubmit} aria-busy={sending}>
      <div className="message-input-container">
        <textarea
          ref={inputRef}
          className="message-input"
          placeholder="Type a message..."
          value={content}
          onChange={(event) => onContentChange(event.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          readOnly={sending}
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
