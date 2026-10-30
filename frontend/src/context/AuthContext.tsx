import {
  useEffect,
  useState,
  type ReactNode,
} from "react";

import type { User, LoginRequest } from "../types/auth";

import {
  login as loginRequest,
  getCurrentUser,
} from "../services/auth";

import { AuthContext } from "./auth-context";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);

  const [token, setToken] = useState<string | null>(
    () => sessionStorage.getItem("token")
  );

  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function restoreSession() {
      if (!token) {
        setLoading(false);
        return;
      }

      try {
        const currentUser = await getCurrentUser(token);

        if (!cancelled) {
          setUser(currentUser);
        }
      } catch {
        if (!cancelled) {
          sessionStorage.removeItem("token");
          setToken(null);
          setUser(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    restoreSession();

    return () => {
      cancelled = true;
    };
  }, [token]);

  async function login(credentials: LoginRequest) {
    const response = await loginRequest(credentials);

    sessionStorage.setItem("token", response.token);

    setToken(response.token);
    setUser(response.user);
    setLoading(false);
  }

  function logout() {
    sessionStorage.removeItem("token");
    setToken(null);
    setUser(null);
    setLoading(false);
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        loading,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}