import {
  useEffect,
  useState,
  useRef,
  type ReactNode,
} from "react";

import type { User, LoginRequest } from "../types/auth";

import {
  login as loginRequest,
  getCurrentUser,
} from "../services/auth";

import { AuthContext } from "./auth-context";
import { SESSION_EXPIRED_EVENT } from "../services/session";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);

  const [token, setToken] = useState<string | null>(
    () => sessionStorage.getItem("token")
  );

  const [loading, setLoading] = useState(true);
  const [sessionMessage, setSessionMessage] = useState("");
  const tokenRef = useRef(token);

  useEffect(() => {
    function handleSessionExpired(event: Event) {
      const rejectedToken = (event as CustomEvent<string>).detail;
      if (!rejectedToken || rejectedToken !== tokenRef.current) return;

      tokenRef.current = null;
      sessionStorage.removeItem("token");
      setToken(null);
      setUser(null);
      setLoading(false);
      setSessionMessage("Your session expired. Please log in again.");
    }

    window.addEventListener(SESSION_EXPIRED_EVENT, handleSessionExpired);
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, handleSessionExpired);
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function restoreSession() {
      if (!token) {
        setLoading(false);
        return;
      }

      try {
        const currentUser = await getCurrentUser(token);

        if (!cancelled && tokenRef.current === token) {
          setUser(currentUser);
        }
      } catch {
        if (!cancelled && tokenRef.current === token) {
          tokenRef.current = null;
          sessionStorage.removeItem("token");
          setToken(null);
          setUser(null);
        }
      } finally {
        if (!cancelled && tokenRef.current === token) {
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
    tokenRef.current = response.token;
    setSessionMessage("");

    setToken(response.token);
    setUser(response.user);
    setLoading(false);
  }

  function logout() {
    tokenRef.current = null;
    setSessionMessage("");
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
        sessionMessage,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}