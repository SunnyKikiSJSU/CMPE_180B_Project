import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

// Unit tests for pure helpers (FRONTEND_PLAN.md §6.2): no browser, fetch is mocked.
export default defineConfig({
  resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
  test: { environment: "node", include: ["src/**/*.test.ts"] },
});
