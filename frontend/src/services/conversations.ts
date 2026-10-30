import { apiRequest } from "./api";
import type { Conversation } from "../types/conversation";

export async function getConversations(
  token: string
): Promise<Conversation[]> {
  return apiRequest<Conversation[]>(
    "/api/conversations",
    token
  );
}