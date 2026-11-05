export interface Conversation {
  id: number;
  type: "direct" | "group";
  created_at: string;
  other_username: string;
}

export type CreateConversationResponse = Pick<
  Conversation,
  "id" | "type" | "created_at"
>;
