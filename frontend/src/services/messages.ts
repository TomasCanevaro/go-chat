import { apiRequest } from "./api";
import type { Message } from "../types/message";

export async function getMessages(
  token: string,
  conversationId: number
): Promise<Message[]> {
  return apiRequest<Message[]>(
    `/api/conversations/${conversationId}/messages`,
    token
  );
}