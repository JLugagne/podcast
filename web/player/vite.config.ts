import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  // Project sites are served under "/<repo>/"; set BASE_PATH for the build.
  base: process.env.BASE_PATH ?? "/",
  plugins: [react()],
  server: {
    port: 5174,
  },
  preview: {
    port: 4173,
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.test.{ts,tsx}", "scripts/**/*.test.mjs"],
    css: false,
    testTimeout: 15000,
    hookTimeout: 15000,
  },
});
