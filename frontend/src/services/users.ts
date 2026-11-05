import { apiRequest } from "./api";
import type { User } from "../types/auth";

export async function getUsers(token: string): Promise<User[]> {
  return apiRequest<User[]>("/api/users", token);
}
