import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  testMatch: "*.spec.ts",
  timeout: 60_000,
  retries: 1,
  use: {
    baseURL: `http://localhost:${process.env.PKB_PORT || "8080"}`,
    trace: "on-first-retry",
  },
  webServer: {
    command: `${process.env.PKB_BINARY || "../../pkb"} serve --addr :${process.env.PKB_PORT || "8080"}`,
    port: Number(process.env.PKB_PORT || "8080"),
    reuseExistingServer: !process.env.CI,
    timeout: 15_000,
  },
});
