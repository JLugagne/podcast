import "@testing-library/jest-dom/vitest";
import { cleanup, configure } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, vi } from "vitest";
import { resetStore } from "./handlers";
import { server } from "./server";

configure({ asyncUtilTimeout: 3000 });

beforeAll(() => {
  server.listen({ onUnhandledFrame: "error" });
});

afterEach(async () => {
  cleanup();
  vi.useRealTimers();
  // Let in-flight progress saves finish before the next test resets the store.
  await new Promise((resolve) => setTimeout(resolve, 0));
  server.resetHandlers();
  resetStore();
  window.localStorage.clear();
  Reflect.deleteProperty(navigator, "mediaSession");
  Reflect.deleteProperty(navigator, "clipboard");
  Reflect.deleteProperty(globalThis, "MediaMetadata");
  vi.restoreAllMocks();
});

afterAll(() => {
  server.close();
});
