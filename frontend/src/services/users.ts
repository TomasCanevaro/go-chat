import { apiRequest } from "./api";
import type { UserSummary } from "../types/auth";

export async function getUsers(token: string): Promise<UserSummary[]> {
  return apiRequest<UserSummary[]>("/api/users", token);
}
