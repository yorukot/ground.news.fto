import { redirect } from "react-router";
import { sessionCookie } from "~/lib/admin.server";

export function action() {
  return redirect("/admin/login", { headers: { "Set-Cookie": sessionCookie(null) } });
}
