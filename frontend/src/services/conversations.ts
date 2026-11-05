import { apiRequest } from "./api";
import type {
  Conversation,
  CreateConversationResponse,
} from "../types/conversation";

export async function getConversations(
  token: string
): Promise<Conversation[]> {
  return apiRequest<Conversation[]>(
    "/api/conversations",
    token
  );
}

export async function createConversation(
  token: string,
  userId: number
): Promise<CreateConversationResponse> {
  return apiRequest<CreateConversationResponse>(
    "/api/conversations",
    token,
    {
      method: "POST",
      body: JSON.stringify({ user_id: userId }),
    }
  );
}
