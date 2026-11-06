export async function handleResponse<T>(response: Response): Promise<T> {
  if (response.ok) {
    return response.json() as Promise<T>;
  }

  const fallback = `Request failed (${response.status})`;
  const contentType = response.headers.get("Content-Type")?.toLowerCase() ?? "";
  const body = await response.text().catch(() => "");
  let message = fallback;

  if (contentType.includes("application/json") || contentType.includes("+json")) {
    try {
      const error: unknown = JSON.parse(body);
      if (error !== null && typeof error === "object") {
        for (const key of ["error", "message"] as const) {
          if (key in error) {
            const value = (error as Record<string, unknown>)[key];
            if (typeof value === "string" && value.trim()) {
              message = value.trim();
              break;
            }
          }
        }
      }
    } catch {
      // Malformed JSON falls back to the HTTP status message.
    }
  } else if (contentType.includes("text/plain") && body.trim()) {
    message = body.trim();
  }

  throw new Error(message);
}
