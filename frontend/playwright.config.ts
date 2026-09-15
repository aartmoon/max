import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./tests",
  use: {
    baseURL: process.env.BASE_URL || "http://localhost:3000",
    viewport: { width: 390, height: 844 },
  },
  workers: 1,
  reporter: "list",
});
