import { index, type RouteConfig, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  route("events/:id", "routes/event.tsx"),
  route("outlets", "routes/outlets.tsx"),
  route("about", "routes/about.tsx"),
  route("preferences", "routes/preferences.ts"),
  route("admin/login", "routes/admin/login.tsx"),
  route("admin/logout", "routes/admin/logout.ts"),
  route("admin/event-search", "routes/admin/event-search.ts"),
  route("admin", "routes/admin/links.tsx"),
  route("admin/events/:id", "routes/admin/event.tsx"),
] satisfies RouteConfig;
