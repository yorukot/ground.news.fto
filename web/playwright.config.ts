import { defineConfig, devices } from "@playwright/test";

// End-to-end tests run against a running site (web + API + a seeded database):
//   make dev-db migrate seed serve   and   pnpm build && pnpm start
// E2E_BASE_URL overrides where the site is; E2E_ADMIN_TOKEN enables the admin tests.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  reporter: "list",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://127.0.0.1:3000",
    locale: "en-GB",
    trace: "retain-on-failure",
  },
  projects: [
    { name: "desktop", use: { ...devices["Desktop Chrome"] } },
    { name: "phone", use: { ...devices["Pixel 7"] }, testIgnore: /admin/ },
  ],
});
