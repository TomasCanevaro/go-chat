import { useState } from "react";
import { Link, useNavigate } from "react-router";

import { register } from "../services/auth";

import "../styles/auth.css";

export default function RegisterPage() {
  const navigate = useNavigate();

  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    setError("");
    if (!username.trim()) {
      setError("Username cannot be blank");
      return;
    }
    if (Array.from(username).length > 50) {
      setError("Username cannot exceed 50 characters");
      return;
    }
    if (Array.from(email).length > 255) {
      setError("Email cannot exceed 255 characters");
      return;
    }
    if (Array.from(password).length < 8) {
      setError("Password must contain at least 8 characters");
      return;
    }
    if (new TextEncoder().encode(password).length > 72) {
      setError("Password cannot exceed 72 bytes");
      return;
    }
    setLoading(true);

    try {
      await register({ username, email, password });

      navigate("/login");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Registration failed"
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-card" aria-labelledby="register-title">
        <header className="auth-header">
          <p className="auth-brand">Go Chat</p>
          <h1 id="register-title">Create an account</h1>
          <p className="auth-description">Join Go Chat and start a conversation.</p>
        </header>

        <form className="auth-form" onSubmit={handleSubmit} aria-busy={loading}>
          <div className="auth-field">
            <label htmlFor="username">Username</label>
            <input
              id="username"
              name="username"
              type="text"
              autoComplete="username"
              placeholder="Choose a username"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              disabled={loading}
              aria-describedby={error ? "register-error" : undefined}
              required
            />
          </div>

          <div className="auth-field">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              placeholder="you@example.com"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              disabled={loading}
              aria-describedby={error ? "register-error" : undefined}
              required
            />
          </div>

          <div className="auth-field">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              name="password"
              type="password"
              autoComplete="new-password"
              placeholder="Create a password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              disabled={loading}
              aria-describedby={error ? "password-hint register-error" : "password-hint"}
              minLength={8}
              required
            />
            <p className="auth-hint" id="password-hint">Password must be at least 8 characters long.</p>
          </div>

          {error && (
            <p className="auth-error" id="register-error" role="alert">
              {error}
            </p>
          )}

          <button className="auth-submit" type="submit" disabled={loading}>
            {loading ? "Creating account..." : "Register"}
          </button>
        </form>

        <p className="auth-footer">
          Already have an account? <Link to="/login">Login</Link>
        </p>
      </section>
    </main>
  );
}
