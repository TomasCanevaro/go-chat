import { useState } from "react";
import { Link, useNavigate } from "react-router";

import { useAuth } from "../hooks/useAuth";

import "../styles/auth.css";

export default function LoginPage() {
  const navigate = useNavigate();

  const { login, sessionMessage } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    setError("");
    setLoading(true);

    try {
      await login({ email, password });

      navigate("/");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Login failed"
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-card" aria-labelledby="login-title">
        <header className="auth-header">
          <p className="auth-brand">Go Chat</p>
          <h1 id="login-title">Welcome back</h1>
          <p className="auth-description">Log in to continue your conversations.</p>
        </header>

        {sessionMessage && (
          <p className="auth-error auth-notice" role="status">{sessionMessage}</p>
        )}

        <form className="auth-form" onSubmit={handleSubmit} aria-busy={loading}>
          <div className="auth-field">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              name="email"
              type="email"
              autoComplete="username"
              placeholder="you@example.com"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              disabled={loading}
              aria-describedby={error ? "login-error" : undefined}
              required
            />
          </div>

          <div className="auth-field">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              placeholder="Enter your password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              disabled={loading}
              aria-describedby={error ? "login-error" : undefined}
              required
            />
          </div>

          {error && (
            <p className="auth-error" id="login-error" role="alert">
              {error}
            </p>
          )}

          <button className="auth-submit" type="submit" disabled={loading}>
            {loading ? "Logging in..." : "Login"}
          </button>
        </form>

        <p className="auth-footer">
          Don't have an account? <Link to="/register">Register</Link>
        </p>
      </section>
    </main>
  );
}
