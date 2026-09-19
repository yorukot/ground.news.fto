import { defineConfig } from "vitest/config";

// Separate from vite.config.ts: the React Router plugin is for the app build
// and isn't needed (or supported) when testing components in isolation.
export default defineConfig({
  resolve: { tsconfigPaths: true },
  test: {
    environment: "jsdom",
    include: ["app/**/*.test.{ts,tsx}"],
    setupFiles: ["./vitest.setup.ts"],
    css: { modules: { classNameStrategy: "non-scoped" } },
  },
});
