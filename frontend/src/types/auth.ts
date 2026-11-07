export interface UserSummary {
  id: number;
  username: string;
}

export interface User extends UserSummary {
  email: string;
  created_at?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}