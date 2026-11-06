export interface Message {
  id: number;
  conversation_id: number;
  sender_id: number;
  sender_username: string;
  content: string;
  created_at: string;
}

export interface MessagePage {
  messages: Message[];
  has_more: boolean;
}
