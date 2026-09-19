// Server-only helpers for the admin pages. The admin token lives in an
// HTTP-only cookie on this server and is sent to the Go API as a bearer
// token; it never reaches browser JavaScript.
import { redirect } from "react-router";

const API_URL = process.env.API_URL ?? "http://127.0.0.1:8080";
const COOKIE = "admin_token";

export type LinkDecision = {
  articleId: number;
  headline: string;
  outletName: string;
  publishedAt: string;
  confidence: number;
  eventId: number;
  eventTitle: string;
};

export type EventMatch = { id: number; title: string; updatedAt: string; articleCount: number };

function readToken(request: Request): string | null {
  const match = request.headers.get("Cookie")?.match(/(?:^|;\s*)admin_token=([^;]+)/);
  return match?.[1] ? decodeURIComponent(match[1]) : null;
}

export function sessionCookie(token: string | null): string {
  const secure = process.env.NODE_ENV === "production" ? "; Secure" : "";
  const base = `${COOKIE}=${token ? encodeURIComponent(token) : ""}; Path=/; HttpOnly; SameSite=Strict${secure}`;
  return token ? `${base}; Max-Age=${60 * 60 * 12}` : `${base}; Max-Age=0`;
}

/** True when the API accepts the token. False also covers "admin is disabled". */
export async function tokenIsValid(token: string): Promise<boolean> {
  const res = await fetch(`${API_URL}/api/v1/admin/session`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  return res.status === 204;
}

/** Calls an admin endpoint, sending the reader to the login page if the session is gone. */
export async function adminFetch(request: Request, path: string, init?: RequestInit) {
  const token = readToken(request);
  if (!token) throw redirect("/admin/login");
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      ...init?.headers,
      Authorization: `Bearer ${token}`,
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
    },
  });
  if (res.status === 401 || (res.status === 404 && path === "/api/v1/admin/session")) {
    throw redirect("/admin/login", { headers: { "Set-Cookie": sessionCookie(null) } });
  }
  return res;
}

/** The API's error message for a failed action, or a generic one. */
export async function errorMessage(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as { error?: { message?: string } };
    return body.error?.message ?? `Request failed (${res.status})`;
  } catch {
    return `Request failed (${res.status})`;
  }
}
