import "@testing-library/jest-dom/vitest";
import { cleanup, configure } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, vi } from "vitest";
import { setUploadTransport } from "../api/upload";
import { setAudioDurationReader } from "../lib/media";
import { resetStore } from "./handlers";
import { server } from "./server";

configure({ asyncUtilTimeout: 3000 });

beforeAll(() => {
  server.listen({ onUnhandledFrame: "error" });
  // jsdom does not implement media playback; seeking tests only need the setter.
  Object.defineProperty(HTMLMediaElement.prototype, "play", {
    configurable: true,
    writable: true,
    value: vi.fn().mockResolvedValue(undefined),
  });
});

afterEach(() => {
  server.resetHandlers();
  resetStore();
  setUploadTransport(null);
  setAudioDurationReader(null);
  cleanup();
});

afterAll(() => {
  server.close();
});
