import { useAuth } from "../hooks/useAuth";

export default function ChatPage() {
  const { user, logout } = useAuth();

  return (
    <main>
      <h1>Go Chat</h1>

      <p>
        Welcome, <strong>{user?.username}</strong>!
      </p>

      <p>Your conversations will appear here.</p>

      <button onClick={logout}>
        Logout
      </button>
    </main>
  );
}