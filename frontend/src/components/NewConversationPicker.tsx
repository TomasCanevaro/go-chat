import { useEffect, useState } from "react";

import { useAuth } from "../hooks/useAuth";
import { getUsers } from "../services/users";
import type { UserSummary } from "../types/auth";

interface NewConversationPickerProps {
  onSelectUser: (user: UserSummary) => Promise<void>;
  onCancel: () => void;
}

export default function NewConversationPicker({
  onSelectUser,
  onCancel,
}: NewConversationPickerProps) {
  const { token, user } = useAuth();
  const [users, setUsers] = useState<UserSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [selectionError, setSelectionError] = useState("");
  const [selectingUserId, setSelectingUserId] = useState<number | null>(null);
  const [retryAttempt, setRetryAttempt] = useState(0);

  useEffect(() => {
    if (!token) return;

    let cancelled = false;

    async function loadUsers() {
      try {
        const data = await getUsers(token!);
        if (!cancelled) {
          setUsers(data ?? []);
          setLoadError("");
        }
      } catch (err) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : "Failed to load users");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    loadUsers();

    return () => {
      cancelled = true;
    };
  }, [token, retryAttempt]);

  async function handleSelectUser(selectedUser: UserSummary) {
    if (selectingUserId !== null) return;

    setSelectingUserId(selectedUser.id);
    setSelectionError("");

    try {
      await onSelectUser(selectedUser);
    } catch (err) {
      setSelectionError(
        err instanceof Error ? err.message : "Failed to open conversation"
      );
    } finally {
      setSelectingUserId(null);
    }
  }

  const otherUsers = users.filter((candidate) => candidate.id !== user?.id);
  const selecting = selectingUserId !== null;

  return (
    <section className="new-conversation-picker" aria-label="New conversation">
      <h3>New conversation</h3>
      <button
        type="button"
        className="logout-button"
        onClick={onCancel}
        disabled={selecting}
      >
        Cancel
      </button>

      {loading ? (
        <p role="status">Loading users...</p>
      ) : loadError ? (
        <div className="chat-feedback" role="alert">
          <p>Could not load users: {loadError}</p>
          <button
            type="button"
            className="logout-button"
            onClick={() => {
              setLoading(true);
              setLoadError("");
              setRetryAttempt((previous) => previous + 1);
            }}
          >
            Retry
          </button>
        </div>
      ) : otherUsers.length === 0 ? (
        <p>No other users available yet.</p>
      ) : (
        <ul className="conversation-list">
          {otherUsers.map((candidate) => (
            <li key={candidate.id}>
              <button
                type="button"
                className="conversation-item"
                disabled={selecting}
                onClick={() => handleSelectUser(candidate)}
              >
                {selectingUserId === candidate.id
                  ? `Opening ${candidate.username}...`
                  : candidate.username}
              </button>
            </li>
          ))}
        </ul>
      )}

      {selectionError && <p className="chat-feedback" role="alert">{selectionError}</p>}
    </section>
  );
}
