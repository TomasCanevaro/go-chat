import { useEffect, useState } from "react";

export type SocketStatus =
  | "disconnected"
  | "connecting"
  | "connected"
  | "error";

export function useChatSocket(token: string | null) {
  const [status, setStatus] = useState<SocketStatus>("disconnected");

  useEffect(() => {
    if (!token) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${protocol}//${window.location.host}/ws`;

    const socket = new WebSocket(url);

    let active = true;

    socket.onopen = () => {
      if (!active) return;

      socket.send(
        JSON.stringify({
          type: "auth",
          token,
        })
      );
    };

    socket.onmessage = (event: MessageEvent<string>) => {
      if (!active) return;

      try {
        const data: unknown = JSON.parse(event.data);

        if (typeof data !== "object" || data === null || !("type" in data)) {
          return;
        }

        if (data.type === "auth_success") {
          setStatus("connected");
        } else if (data.type === "error") {
          console.error("WebSocket error:", data);
          setStatus("error");
        }
      } catch {
        console.error("Invalid WebSocket message");
      }
    };

    socket.onerror = () => {
      if (active) setStatus("error");
    };

    socket.onclose = () => {
      if (active) setStatus("disconnected");
    };

    return () => {
      active = false;
      socket.close();
    };
  }, [token]);

  return { status };
}