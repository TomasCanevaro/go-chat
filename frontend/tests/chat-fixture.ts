import { expect, type Page, type WebSocketRoute } from "@playwright/test";
import type { Message } from "../src/types/message.ts";

interface SendRequest {
  type: "send_message";
  client_id: string;
  conversation_id: number;
  content: string;
}

export function history(count: number): Message[] {
  return Array.from({ length: count }, (_, index) => ({
    id: index + 1,
    conversation_id: 1,
    sender_id: index % 2 ? 1 : 2,
    sender_username: index % 2 ? "Alice" : "Bob",
    content: `Message ${index + 1}`,
    created_at: "2026-01-01T12:00:00Z",
  }));
}

// These routes never forward requests to the Go server or a database.
export async function mockChat(page: Page, initialMessages: Message[] = []) {
  const state = {
    messages: [...initialMessages],
    sends: [] as SendRequest[],
    dropNextAck: false,
    rejectNextSend: false,
    rejectAuthentication: false,
    usersStatus: 200,
    failOlderPage: false,
    connections: 0,
    requestedBeforeIds: [] as number[],
  };
  let socket: WebSocketRoute | undefined;
  const savedRequests = new Map<string, Message>();
  const pageErrors: string[] = [];
  page.on("pageerror", (error) => pageErrors.push(error.message));

  await page.addInitScript(() => sessionStorage.setItem("token", "test-token"));
  await page.route("**/api/**", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === "/api/me") {
      return route.fulfill({ json: { id: 1, username: "Alice", email: "alice@example.com" } });
    }
    if (url.pathname === "/api/conversations") {
      return route.fulfill({ json: [
        { id: 1, type: "direct", other_username: "Bob" },
        { id: 2, type: "direct", other_username: "Charlie" },
      ] });
    }
    if (url.pathname === "/api/users") {
      return route.fulfill({ status: state.usersStatus, json: state.usersStatus === 200
        ? [{ id: 2, username: "Bob" }]
        : { message: "Test directory failure" } });
    }
    if (/^\/api\/conversations\/\d+\/messages$/.test(url.pathname)) {
      const before = url.searchParams.get("before_id");
      if (before) {
        state.requestedBeforeIds.push(Number(before));
        if (state.failOlderPage) {
          state.failOlderPage = false;
          return route.fulfill({ status: 500, contentType: "text/plain", body: "Test history failure" });
        }
      }
      expect(url.searchParams.get("limit")).toBe("30");
      const conversationId = Number(url.pathname.split("/")[3]);
      const matching = state.messages
        .filter((message) => message.conversation_id === conversationId &&
          (!before || message.id < Number(before)))
        .sort((a, b) => a.id - b.id);
      return route.fulfill({ json: { messages: matching.slice(-30), has_more: matching.length > 30 } });
    }
    throw new Error(`Unexpected mock API request: ${route.request().method()} ${url.pathname}`);
  });

  await page.routeWebSocket("**/ws", (connection) => {
    socket = connection;
    state.connections++;
    connection.onMessage((raw) => {
      const request = JSON.parse(String(raw));
      if (request.type === "auth") {
        if (state.rejectAuthentication) {
          connection.close({ code: 1008, reason: "Invalid token" });
        } else {
          connection.send(JSON.stringify({ type: "auth_success" }));
        }
        return;
      }
      state.sends.push(request as SendRequest);
      if (state.rejectNextSend) {
        state.rejectNextSend = false;
        connection.send(JSON.stringify({ type: "message_error", client_id: request.client_id, message: "Test rejection" }));
        return;
      }
      let message = savedRequests.get(request.client_id);
      if (!message) {
        message = {
          id: Math.max(0, ...state.messages.map((item) => item.id)) + 1,
          conversation_id: request.conversation_id,
          sender_id: 1,
          sender_username: "Alice",
          content: request.content,
          created_at: "2026-01-01T12:00:00Z",
        };
        savedRequests.set(request.client_id, message);
        state.messages.push(message);
        connection.send(JSON.stringify({ type: "new_message", message }));
      }
      if (state.dropNextAck) {
        state.dropNextAck = false;
        return;
      }
      connection.send(JSON.stringify({ type: "message_ack", client_id: request.client_id, message_id: message.id }));
    });
  });

  const input = page.getByRole("textbox", { name: "Message", exact: true });
  return {
    state,
    pageErrors,
    input,
    async open(name = "Bob") {
      await page.locator(".conversation-sidebar .conversation-item").filter({ hasText: name }).click();
      await expect(input).toBeEnabled();
    },
    disconnect() {
      if (!socket) throw new Error("No mock WebSocket connected");
      socket.close({ code: 1011, reason: "Test connection loss" });
    },
  };
}
