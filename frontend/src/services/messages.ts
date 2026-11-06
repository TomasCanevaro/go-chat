import { apiRequest } from "./api";
import type { Message, MessagePage } from "../types/message";

export async function getMessages(
  token: string,
  conversationId: number,
  beforeId?: number,
  signal?: AbortSignal
): Promise<MessagePage> {
  const query = new URLSearchParams({ limit: "30" });
  if (beforeId !== undefined) query.set("before_id", String(beforeId));

  return apiRequest<MessagePage>(
    `/api/conversations/${conversationId}/messages?${query}`,
    token,
    { signal }
  );
}

export async function sendMessage(
  token: string,
  conversationId: number,
  content: string
): Promise<Message> {
  return apiRequest<Message>(
    `/api/conversations/${conversationId}/messages`,
    token,
    {
      method: "POST",
      body: JSON.stringify({ content }),
    }
  );
}