import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { downloadEpisode, downloadedObjectURL, isDownloaded, removeDownload } from "./downloads";

/** Minimal in-memory Cache Storage. */
function installCaches(): void {
  const stores = new Map<string, Map<string, Response>>();
  vi.stubGlobal("caches", {
    open: async (name: string) => {
      const store = stores.get(name) ?? new Map<string, Response>();
      stores.set(name, store);
      return {
        put: async (key: string, response: Response) => void store.set(key, response),
        match: async (key: string) => store.get(key)?.clone(),
        delete: async (key: string) => store.delete(key),
      };
    },
  });
}

describe("offline downloads", () => {
  beforeEach(() => {
    installCaches();
    URL.createObjectURL = vi.fn(() => "blob:fake");
  });
  afterEach(() => vi.unstubAllGlobals());

  it("stores the audio, exposes it as an object URL, and removes it", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("audio-bytes", { headers: { "content-type": "audio/mpeg" } })),
    );
    const progress: number[] = [];

    await downloadEpisode("show/ep-1", "https://cdn.example/1.mp3", (f) => progress.push(f));

    expect(isDownloaded("show/ep-1")).toBe(true);
    expect(progress.at(-1)).toBe(1);
    expect(await downloadedObjectURL("show/ep-1")).toBe("blob:fake");

    await removeDownload("show/ep-1");
    expect(isDownloaded("show/ep-1")).toBe(false);
    expect(await downloadedObjectURL("show/ep-1")).toBeNull();
  });

  it("explains CORS failures and keeps nothing", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => Promise.reject(new TypeError("Failed to fetch"))),
    );

    await expect(downloadEpisode("show/ep-2", "https://cdn.example/2.mp3")).rejects.toThrow(/CORS/);
    expect(isDownloaded("show/ep-2")).toBe(false);
  });
});
