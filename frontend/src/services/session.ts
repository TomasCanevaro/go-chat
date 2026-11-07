export const SESSION_EXPIRED_EVENT = "go-chat:session-expired";

// Include the rejected token so a delayed response cannot end a newer session.
export function notifySessionExpired(token: string) {
  window.dispatchEvent(new CustomEvent<string>(SESSION_EXPIRED_EVENT, { detail: token }));
}
