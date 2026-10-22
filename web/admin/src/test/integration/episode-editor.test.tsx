import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import type { Episode } from "../../api/types";
import { setUploadTransport, type UploadRequest, type UploadResponse } from "../../api/upload";
import { setAudioDurationReader } from "../../lib/media";
import { seedEpisode, seedSeries, store } from "../handlers";
import { renderApp } from "../renderApp";

beforeEach(() => {
  setAudioDurationReader(async () => 123.4);
});

const VALID_CHAPTERS = JSON.stringify(
  {
    version: "1.2.0",
    chapters: [
      { startTime: 0, title: "Intro" },
      { startTime: 60, title: "Main topic" },
    ],
  },
  null,
  2,
);

const AUDIO = {
  url: "/api/v1/episodes/episode-1/audio",
  content_type: "audio/mpeg",
  size_bytes: 4096,
  duration_seconds: 120,
  original_name: "pilot.mp3",
};

function seedEditor(overrides: Partial<Episode> = {}): Episode {
  seedSeries({ id: "series-1", title: "Deep Dive" });
  return seedEpisode("series-1", {
    id: "episode-1",
    number: 1,
    title: "Pilot",
    ...overrides,
  });
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

async function openEditor(user: ReturnType<typeof renderApp>["user"]) {
  await screen.findByRole("heading", { name: "Pilot" });
  return user;
}

describe("episode editor - metadata", () => {
  it("saves metadata with the loaded version", async () => {
    seedEditor({ version: 2 });
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    const number = await screen.findByLabelText(/^Number/);
    await user.clear(number);
    await user.type(number, "5");

    const title = await screen.findByLabelText(/^Title/);
    await user.clear(title);
    await user.type(title, "Renamed");

    const description = screen.getByLabelText("Description");
    await user.clear(description);
    await user.type(description, "Updated description");

    await user.click(screen.getByRole("button", { name: "Save details" }));

    expect(await screen.findByText("Episode details saved.")).toBeInTheDocument();
    const request = store.requests.find(
      (entry) => entry.method === "PUT" && entry.path === "/api/v1/episodes/episode-1",
    );
    expect(request?.body).toMatchObject({
      number: 5,
      title: "Renamed",
      description: "Updated description",
      version: 2,
    });
    expect(store.episodes[0]).toMatchObject({ number: 5, title: "Renamed" });
  });

  it("shows a server validation error", async () => {
    seedEditor({ version: 2 });
    store.failNextEpisodeUpdate = {
      status: 422,
      code: "INVALID_EPISODE",
      message: "The episode title is invalid.",
    };

    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    await user.click(screen.getByRole("button", { name: "Save details" }));

    expect(await screen.findByText("The episode title is invalid.")).toBeInTheDocument();
    expect(store.episodes[0].title).toBe("Pilot");
  });
});

describe("episode editor - audio", () => {
  it("detects the duration, reports progress and uploads the audio", async () => {
    const episode = seedEditor();
    const pending = deferred<UploadResponse>();
    let captured: UploadRequest | undefined;
    setUploadTransport((request) => {
      captured = request;
      request.onProgress?.({ percent: 42, loaded: 42, total: 100 });
      return pending.promise;
    });

    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    await user.upload(
      screen.getByLabelText(/upload audio file/i),
      new File(["audio-bytes"], "pilot.mp3", { type: "audio/mpeg" }),
    );

    const percent = await screen.findByText("42%");
    const liveRegion = percent.closest('[aria-live="polite"]');
    expect(liveRegion).not.toBeNull();
    expect(
      within(liveRegion as HTMLElement).getByRole("progressbar", {
        name: "Audio upload progress",
      }),
    ).toHaveAttribute("aria-valuenow", "42");
    expect(captured?.method).toBe("PUT");
    expect(captured?.url).toBe("/api/v1/episodes/episode-1/audio");
    expect(captured?.field).toBe("file");
    expect(captured?.file.name).toBe("pilot.mp3");
    expect(captured?.fields).toEqual({ duration_seconds: "123.4" });

    const updated: Episode = { ...episode, version: 2, audio: AUDIO };
    Object.assign(store.episodes[0], updated);

    await act(async () => {
      pending.resolve({ status: 200, body: JSON.stringify(updated) });
    });

    expect(await screen.findByText("pilot.mp3")).toBeInTheDocument();
    expect(screen.queryByText("42%")).not.toBeInTheDocument();
    const audio = document.querySelector("audio");
    expect(audio?.getAttribute("src")).toContain("/api/v1/episodes/episode-1/audio");
  });

  it("appends duration_seconds before the file part in the multipart body", async () => {
    seedEditor();
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    await user.upload(
      screen.getByLabelText(/upload audio file/i),
      new File(["audio-bytes"], "pilot.mp3", { type: "audio/mpeg" }),
    );

    expect(await screen.findByText("pilot.mp3")).toBeInTheDocument();
    const request = store.requests.find(
      (entry) => entry.method === "PUT" && entry.path === "/api/v1/episodes/episode-1/audio",
    );
    expect(request?.contentType).toContain("multipart/form-data");
    expect(request?.body).toMatchObject({
      order: ["duration_seconds", "file"],
      fields: { duration_seconds: "123.4" },
      fileName: "pilot.mp3",
    });
  });

  it("accepts an audio file dropped on the drop zone", async () => {
    const episode = seedEditor();
    const updated: Episode = {
      ...episode,
      version: 2,
      audio: { ...AUDIO, original_name: "dropped.mp3" },
    };
    Object.assign(store.episodes[0], updated);
    setUploadTransport(async () => ({ status: 200, body: JSON.stringify(updated) }));

    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    const dropZone = screen.getByLabelText(/audio file/i).closest("fieldset") as HTMLElement;
    fireEvent.drop(dropZone, {
      dataTransfer: { files: [new File(["audio"], "dropped.mp3", { type: "audio/mpeg" })] },
    });

    expect(await screen.findByText("dropped.mp3")).toBeInTheDocument();
  });

  it("cancels an in-flight upload", async () => {
    seedEditor();
    let captured: UploadRequest | undefined;
    setUploadTransport((request) => {
      captured = request;
      request.onProgress?.({ percent: 10, loaded: 10, total: 100 });
      return new Promise<UploadResponse>((_resolve, reject) => {
        request.signal?.addEventListener(
          "abort",
          () => reject(new DOMException("Upload cancelled", "AbortError")),
          { once: true },
        );
      });
    });

    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    await user.upload(
      screen.getByLabelText(/upload audio file/i),
      new File(["audio-bytes"], "pilot.mp3", { type: "audio/mpeg" }),
    );

    expect(await screen.findByText("10%")).toBeInTheDocument();
    expect(captured?.signal?.aborted).toBe(false);

    await user.click(screen.getByRole("button", { name: "Cancel upload" }));

    expect(await screen.findByText("Upload cancelled.")).toBeInTheDocument();
    expect(captured?.signal?.aborted).toBe(true);
    expect(screen.getByLabelText(/upload audio file/i)).not.toBeDisabled();
  });
});

describe("episode editor - chapters", () => {
  it("loads a chapters document from a .json file", async () => {
    seedEditor();
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    await user.upload(
      screen.getByLabelText(/choose a chapters .json file/i),
      new File([VALID_CHAPTERS], "chapters.json", { type: "application/json" }),
    );

    expect(await screen.findByText("The chapters document is valid")).toBeInTheDocument();
    expect(screen.getByText("Main topic")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save chapters" })).toBeEnabled();
  });

  it("validates pasted JSON client-side and blocks saving", async () => {
    seedEditor();
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    const textarea = screen.getByLabelText("Chapters JSON");
    await user.click(textarea);
    await user.paste(
      JSON.stringify({
        version: "1.2.0",
        chapters: [{ startTime: 10, title: "Intro" }],
      }),
    );

    expect(await screen.findByText("Fix the chapters document")).toBeInTheDocument();
    expect(screen.getByText(/earliest starts at 10s/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save chapters" })).toBeDisabled();
  });

  it("saves the raw Podcasting 2.0 document", async () => {
    seedEditor();
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    const textarea = screen.getByLabelText("Chapters JSON");
    await user.click(textarea);
    await user.paste(VALID_CHAPTERS);
    await screen.findByText("The chapters document is valid");

    await user.click(screen.getByRole("button", { name: "Save chapters" }));

    expect(await screen.findByText("Chapters saved.")).toBeInTheDocument();
    const request = store.requests.find(
      (entry) => entry.method === "PUT" && entry.path === "/api/v1/episodes/episode-1/chapters",
    );
    expect(request?.contentType).toContain("application/json");
    expect(request?.body).toBe(VALID_CHAPTERS);
    expect(store.episodes[0].chapters).toHaveLength(2);
  });

  it("shows a server validation error", async () => {
    seedEditor();
    store.failNextChapters = {
      status: 422,
      code: "INVALID_CHAPTERS",
      message: "Chapter start times overlap.",
    };
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    const textarea = screen.getByLabelText("Chapters JSON");
    await user.click(textarea);
    await user.paste(VALID_CHAPTERS);
    await screen.findByText("The chapters document is valid");
    await user.click(screen.getByRole("button", { name: "Save chapters" }));

    expect(await screen.findByText("Chapter start times overlap.")).toBeInTheDocument();
  });

  it("seeks the audio player when a chapter is selected", async () => {
    seedEditor({
      audio: AUDIO,
      chapters: [
        { start_seconds: 0, title: "Intro", toc: true },
        { start_seconds: 95.5, title: "Main topic", toc: true },
      ],
    });
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);
    await screen.findByText("Main topic");

    const audio = document.querySelector("audio") as HTMLAudioElement;
    expect(audio).not.toBeNull();

    await user.click(screen.getByRole("button", { name: "Play chapter 2 at 0:01:35" }));
    expect(audio.currentTime).toBe(95.5);

    audio.currentTime = 0;
    await user.click(screen.getByText("Main topic"));
    expect(audio.currentTime).toBe(95.5);

    await user.click(screen.getByRole("button", { name: "Play chapter 1 at 0:00:00" }));
    expect(audio.currentTime).toBe(0);
  });

  it("links to the chapters download endpoint", async () => {
    seedEditor();
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    const link = screen.getByRole("link", { name: "Download chapters JSON" });
    expect(link).toHaveAttribute("href", "/api/v1/episodes/episode-1/chapters");
    expect(link).toHaveAttribute("download", "episode-1-chapters.json");
  });
});

describe("episode editor - publishing", () => {
  it("blocks publishing until audio and chapters exist", async () => {
    seedEditor();
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    expect(screen.getByRole("button", { name: "Publish episode" })).toBeDisabled();
    expect(screen.getByText("Audio missing")).toBeInTheDocument();
    expect(screen.getByText("0 chapters (at least one required)")).toBeInTheDocument();
    expect(
      screen.getByText(/needs audio and at least one chapter before it can be published/i),
    ).toBeInTheDocument();
  });

  it("publishes and unpublishes an episode", async () => {
    seedEditor({
      audio: AUDIO,
      chapters: [{ start_seconds: 0, title: "Intro", toc: true }],
    });
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    await user.click(screen.getByRole("button", { name: "Publish episode" }));
    expect(await screen.findByRole("button", { name: "Unpublish episode" })).toBeInTheDocument();
    expect(
      store.requests.some(
        (entry) => entry.method === "POST" && entry.path === "/api/v1/episodes/episode-1/publish",
      ),
    ).toBe(true);
    expect(store.episodes[0].status).toBe("published");

    await user.click(screen.getByRole("button", { name: "Unpublish episode" }));
    expect(await screen.findByRole("button", { name: "Publish episode" })).toBeInTheDocument();
    expect(store.episodes[0].status).toBe("draft");
  });
});

describe("episode editor - deletion", () => {
  it("keeps the episode when deletion is cancelled", async () => {
    seedEditor();
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    await user.click(screen.getByRole("button", { name: "Delete episode" }));
    const dialog = screen.getByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    await waitFor(() => expect(dialog.hasAttribute("open")).toBe(false));
    expect(store.episodes).toHaveLength(1);
    expect(screen.getByRole("heading", { name: "Pilot" })).toBeInTheDocument();
  });

  it("deletes the episode after confirmation", async () => {
    seedEditor();
    const { user } = renderApp("/episodes/episode-1");
    await openEditor(user);

    await user.click(screen.getByRole("button", { name: "Delete episode" }));
    const dialog = screen.getByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "Delete episode" }));

    await waitFor(() => expect(store.episodes).toHaveLength(0));
    expect(await screen.findByText("No episodes yet")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Deep Dive" })).toBeInTheDocument();
  });
});
