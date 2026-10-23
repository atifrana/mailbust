import { defineConfig } from "vitest/config";

// Pure-logic unit tests (extraction, address rules) run in plain Node.
// The full email-pipeline / Durable Object integration tests arrive in M2
// using @cloudflare/vitest-pool-workers.
export default defineConfig({
  test: {
    include: ["test/**/*.test.ts"],
    environment: "node",
  },
});
