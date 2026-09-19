// Resource route: event search for the merge and move dialogs.
import { adminFetch, type EventMatch } from "~/lib/admin.server";
import type { Route } from "./+types/event-search";

export async function loader({ request }: Route.LoaderArgs) {
  const query = new URL(request.url).searchParams.get("q")?.trim() ?? "";
  if (query.length < 2) return { events: [] as EventMatch[] };
  const res = await adminFetch(request, `/api/v1/admin/events?q=${encodeURIComponent(query)}`);
  if (!res.ok) throw new Response("Search failed", { status: 502 });
  return (await res.json()) as { events: EventMatch[] };
}
