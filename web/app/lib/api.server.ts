// Server-only client for the Go API. Loaders call these; the browser never
// talks to the API directly.
import type { components } from "./api-types";

export type EventSummary = components["schemas"]["EventSummary"];
export type EventsPage = components["schemas"]["EventsPage"];
export type EventDetail = components["schemas"]["EventDetail"];
export type TimelineStep = components["schemas"]["TimelineStep"];
export type Article = components["schemas"]["Article"];
export type Outlet = components["schemas"]["Outlet"];

const API_URL = process.env.API_URL ?? "http://127.0.0.1:8080";

async function get<T>(path: string, signal?: AbortSignal): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${API_URL}${path}`, { signal, headers: { Accept: "application/json" } });
  } catch (cause) {
    if (signal?.aborted) throw cause;
    console.error(`API unreachable: ${path}`, cause);
    throw new Response("API unreachable", { status: 503 });
  }
  if (res.status === 404) throw new Response("Not found", { status: 404 });
  if (res.status === 400) throw new Response("Bad request", { status: 400 });
  if (!res.ok) {
    console.error(`API error ${res.status}: ${path}`);
    throw new Response("API error", { status: 502 });
  }
  return (await res.json()) as T;
}

export function listEvents(cursor: string | null, signal?: AbortSignal): Promise<EventsPage> {
  const query = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";
  return get<EventsPage>(`/api/v1/events${query}`, signal);
}

export function getEvent(id: string, signal?: AbortSignal): Promise<EventDetail> {
  if (!/^\d+$/.test(id)) throw new Response("Not found", { status: 404 });
  return get<EventDetail>(`/api/v1/events/${id}`, signal);
}

export async function listOutlets(signal?: AbortSignal): Promise<Outlet[]> {
  const body = await get<{ outlets: Outlet[] }>("/api/v1/outlets", signal);
  return body.outlets;
}
