import type {
  AuthResponse,
  LoginRequest,
  RegisterRequest,
} from "../types/auth";

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error = await response.json().catch(() => null);

    throw new Error(
      error?.error || error?.message || "Something went wrong"
    );
  }

  return response.json() as Promise<T>;
}

export async function login(
  credentials: LoginRequest
): Promise<AuthResponse> {
  const response = await fetch("/api/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(credentials),
  });

  return handleResponse<AuthResponse>(response);
}

export async function register(
  data: RegisterRequest
): Promise<unknown> {
  const response = await fetch("/api/register", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(data),
  });

  return handleResponse<unknown>(response);
}