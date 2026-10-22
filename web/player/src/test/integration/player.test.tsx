import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AudioFileDTO, ChapterDTO } from "../../api/types";
import { seedEpisode, seedProgress, seedSeries, store } from "../handlers";
import { makeMediaSessionStub } from "../mediaSession";
import { renderApp } from "../renderApp";

function audioFile(id: string, duration: number): AudioFileDTO {
  return {
    url: `/api/v1/episodes/${id}/audio`,
    content_type: "audio/mpeg",
    size_bytes: 2048,
    duration_seconds: duration,
    original_name: "episode.mp3",
  };
}

interface PlayerSetup {
  positionSeconds?: number;
  chapters?: ChapterDTO[];
  cover?: boolean;
}

async function renderPlayer(setup: PlayerSetup = {}) {
  const series = seedSeries({
    id: "series-1",
    title: "Deep Dive",
    slug: "deep-dive",
    cover_url: setup.cover ? "/api/v1/series/series-1/cover" : null,
  });
  const first = seedEpisode(series.id, {
    id: "episode-1",
    number: 1,
    title: "First episode",
    audio: audioFile("episode-1", 100),
    ...(setup.chapters ? { chapters: setup.chapters } : {}),
  });
  const second = seedEpisode(series.id, {
    id: "episode-2",
    number: 2,
    title: "Second episode",
    audio: audioFile("episode-2", 100),
  });
  if (setup.positionSeconds) seedProgress(first.id, { position_seconds: setup.positionSeconds });

  const view = renderApp("/series/series-1");
  if (setup.positionSeconds) {
    // Wait until the progress query has landed so the resume position is known.
    await screen.findByRole("button", { name: /^Resume: Episode 1/ });
  }
  await view.user.click(await screen.findByRole("button", { name: "First episode" }));
  await screen.findByText("Now playing");
  return { ...view, series, first, second };
}

function slider() {
  return screen.getByRole("slider", { name: "Playback position" });
}

function progressPuts() {
  return store.requests.filter(
    (request) => request.method === "PUT" && request.path.startsWith("/api/v1/listening/progress/"),
  );
}

describe("player", () => {
  it("starts the episode at its saved position and shows the player", async () => {
    const { engine } = await renderPlayer({ positionSeconds: 42 });

    expect(engine.loadCalls).toHaveLength(1);
    expect(engine.loadCalls[0]).toEqual({
      source: "/api/v1/episodes/episode-1/audio",
      startAt: 42,
    });
    expect(screen.getByRole("heading", { name: "First episode" })).toBeInTheDocument();
    expect(screen.getByText("Chapter 2 of 2")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Main topic" })).toBeInTheDocument();
    expect(slider()).toHaveAttribute("aria-valuetext", "0:42 of 1:40, chapter 2: Main topic");
    expect(screen.getByText("0:42")).toBeInTheDocument();
    expect(screen.getByText("-0:58")).toBeInTheDocument();
  });

  it("pauses and resumes playback", async () => {
    const { engine, user } = await renderPlayer();

    await user.click(screen.getByRole("button", { name: "Pause" }));
    expect(engine.pauseCalls).toBe(1);
    expect(screen.getByRole("button", { name: "Play" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Play" }));
    expect(engine.playCalls).toBe(2);
    expect(screen.getByRole("button", { name: "Pause" })).toBeInTheDocument();
  });

  it("seeks backwards 15 seconds and forwards 30 seconds", async () => {
    const { engine, user } = await renderPlayer({ positionSeconds: 42 });
    store.requests = [];

    await user.click(screen.getByRole("button", { name: "Back 15 seconds" }));
    expect(engine.seekCalls.at(-1)).toBe(27);
    expect(slider()).toHaveAttribute("aria-valuenow", "27");

    await user.click(screen.getByRole("button", { name: "Forward 30 seconds" }));
    expect(engine.seekCalls.at(-1)).toBe(57);
    expect(slider()).toHaveAttribute("aria-valuenow", "57");

    await waitFor(() => expect(progressPuts().length).toBeGreaterThanOrEqual(1));
    expect(progressPuts().at(-1)?.body).toEqual({ position_seconds: 57 });
  });

  it("restarts the current chapter, then steps to the previous one", async () => {
    const { engine, user } = await renderPlayer({ positionSeconds: 42 });

    await user.click(screen.getByRole("button", { name: "Previous chapter" }));
    expect(engine.seekCalls.at(-1)).toBe(30);

    await user.click(screen.getByRole("button", { name: "Previous chapter" }));
    expect(engine.seekCalls.at(-1)).toBe(0);

    await user.click(screen.getByRole("button", { name: "Next chapter" }));
    expect(engine.seekCalls.at(-1)).toBe(30);

    await user.click(screen.getByRole("button", { name: "Next chapter" }));
    expect(engine.seekCalls.at(-1)).toBe(30);
  });

  it("seeks when the segmented bar is clicked and with the arrow keys", async () => {
    const { engine } = await renderPlayer();
    const bar = slider();
    vi.spyOn(bar, "getBoundingClientRect").mockReturnValue({
      left: 0,
      width: 1000,
      top: 0,
      height: 10,
      right: 1000,
      bottom: 10,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    });

    fireEvent.click(bar, { clientX: 500 });
    expect(engine.seekCalls.at(-1)).toBe(50);
    expect(slider()).toHaveAttribute("aria-valuenow", "50");

    fireEvent.keyDown(slider(), { key: "ArrowRight" });
    expect(engine.seekCalls.at(-1)).toBe(55);
    fireEvent.keyDown(slider(), { key: "ArrowLeft" });
    expect(engine.seekCalls.at(-1)).toBe(50);
    fireEvent.keyDown(slider(), { key: "Home" });
    expect(engine.seekCalls.at(-1)).toBe(0);
  });

  it("jumps to a chapter from the chapter list", async () => {
    const { engine, user } = await renderPlayer();
    await user.click(screen.getByRole("button", { name: "Chapter 2: Main topic" }));

    expect(engine.seekCalls.at(-1)).toBe(30);
    expect(screen.getByRole("button", { name: "Chapter 2: Main topic" })).toHaveAttribute(
      "aria-current",
      "true",
    );
    expect(screen.getByText("Chapter 2 of 2")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Now playing chapter 2: Main topic");
  });

  it("changes the playback speed and remembers it", async () => {
    const view = await renderPlayer();
    await view.user.click(screen.getByRole("button", { name: "Playback speed: 1×" }));
    await view.user.click(screen.getByRole("menuitemradio", { name: "1.5×" }));

    expect(view.engine.playbackRate).toBe(1.5);
    expect(window.localStorage.getItem("podcast.player.rate")).toBe("1.5");
    expect(screen.getByRole("button", { name: "Playback speed: 1.5×" })).toBeInTheDocument();

    view.unmount();
    const second = renderApp("/series/series-1");
    await second.user.click(await screen.findByRole("button", { name: "First episode" }));
    expect(await screen.findByRole("button", { name: "Playback speed: 1.5×" })).toBeInTheDocument();
  });

  it("pauses playback when a minute sleep timer expires", async () => {
    const view = await renderPlayer();
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval", "Date"] });

    fireEvent.click(screen.getByRole("button", { name: "Sleep timer off" }));
    fireEvent.click(screen.getByRole("menuitemradio", { name: "15 minutes" }));
    expect(screen.getByRole("button", { name: "Sleep timer 15:00" })).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(15 * 60 * 1000);
    });

    expect(view.engine.pauseCalls).toBeGreaterThanOrEqual(1);
    expect(screen.getByRole("button", { name: "Sleep timer off" })).toBeInTheDocument();
  });

  it("pauses at the end of the chapter for a chapter sleep timer", async () => {
    const view = await renderPlayer();
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval", "Date"] });

    fireEvent.click(screen.getByRole("button", { name: "Sleep timer off" }));
    fireEvent.click(screen.getByRole("menuitemradio", { name: "End of chapter" }));

    act(() => {
      view.engine.emitTime(30, 100);
    });

    expect(view.engine.pauseCalls).toBe(1);
    expect(screen.getByRole("button", { name: "Sleep timer off" })).toBeInTheDocument();
  });

  it("throttles progress saves while playing and saves on pause and seek", async () => {
    const view = await renderPlayer();
    store.requests = [];
    vi.useFakeTimers({ toFake: ["Date"] });
    const base = Date.now();

    act(() => {
      view.engine.emitTime(5, 100);
    });
    expect(progressPuts()).toHaveLength(0);

    vi.setSystemTime(new Date(base + 10_000));
    act(() => {
      view.engine.emitTime(15, 100);
    });
    await waitFor(() => expect(progressPuts()).toHaveLength(1));
    expect(progressPuts()[0].body).toEqual({ position_seconds: 15 });

    act(() => {
      view.engine.emitTime(25, 100);
    });
    expect(progressPuts()).toHaveLength(1);

    vi.setSystemTime(new Date(base + 20_000));
    act(() => {
      view.engine.emitTime(35, 100);
    });
    await waitFor(() => expect(progressPuts()).toHaveLength(2));
    expect(progressPuts()[1].body).toEqual({ position_seconds: 35 });

    fireEvent.click(screen.getByRole("button", { name: "Pause" }));
    await waitFor(() => expect(progressPuts().at(-1)?.body).toEqual({ position_seconds: 35 }));

    fireEvent.click(screen.getByRole("button", { name: "Back 15 seconds" }));
    await waitFor(() => expect(progressPuts().at(-1)?.body).toEqual({ position_seconds: 20 }));
  });

  it("marks the episode played and starts the next one when it ends", async () => {
    const view = await renderPlayer();

    act(() => {
      view.engine.emitEnded();
    });

    await waitFor(() => expect(view.engine.loadCalls).toHaveLength(2));
    expect(view.engine.loadCalls[1]).toEqual({
      source: "/api/v1/episodes/episode-2/audio",
      startAt: 0,
    });
    expect(store.requests.some((request) => request.method === "POST")).toBe(true);
    expect(await screen.findByRole("heading", { name: "Second episode" })).toBeInTheDocument();
    expect(screen.getByText("Chapter 1 of 2")).toBeInTheDocument();
  });

  it("does not auto-play when the toggle is off", async () => {
    const view = await renderPlayer();
    await view.user.click(screen.getByRole("checkbox", { name: "Autoplay next episode" }));

    act(() => {
      view.engine.emitEnded();
    });

    await waitFor(() =>
      expect(store.requests.some((request) => request.method === "POST")).toBe(true),
    );
    expect(view.engine.loadCalls).toHaveLength(1);
    expect(screen.getByRole("heading", { name: "First episode" })).toBeInTheDocument();
  });

  it("plays the up-next episode from the queue card", async () => {
    const view = await renderPlayer();
    await view.user.click(screen.getByRole("button", { name: "Play next: Second episode" }));

    expect(view.engine.loadCalls.at(-1)?.source).toBe("/api/v1/episodes/episode-2/audio");
    expect(await screen.findByRole("heading", { name: "Second episode" })).toBeInTheDocument();
  });

  it("handles keyboard shortcuts and ignores them while typing", async () => {
    const view = await renderPlayer({ positionSeconds: 42 });

    fireEvent.keyDown(document.body, { key: " " });
    expect(view.engine.pauseCalls).toBe(1);
    fireEvent.keyDown(document.body, { key: " " });
    expect(view.engine.playCalls).toBe(2);

    fireEvent.keyDown(document.body, { key: "ArrowLeft" });
    expect(view.engine.seekCalls.at(-1)).toBe(27);
    fireEvent.keyDown(document.body, { key: "ArrowRight" });
    expect(view.engine.seekCalls.at(-1)).toBe(57);
    fireEvent.keyDown(document.body, { key: "ArrowLeft", shiftKey: true });
    expect(view.engine.seekCalls.at(-1)).toBe(30);
    fireEvent.keyDown(document.body, { key: "ArrowRight", shiftKey: true });
    expect(view.engine.seekCalls.at(-1)).toBe(30);

    const pauseCalls = view.engine.pauseCalls;
    fireEvent.keyDown(screen.getByRole("checkbox", { name: "Autoplay next episode" }), {
      key: " ",
    });
    expect(view.engine.pauseCalls).toBe(pauseCalls);
  });

  it("wires the Media Session metadata and action handlers", async () => {
    const stub = makeMediaSessionStub();
    const view = await renderPlayer({ cover: true });

    expect(stub.metadata?.title).toBe("First episode");
    expect(stub.metadata?.artist).toBe("Deep Dive");
    expect(stub.metadata?.album).toBe("Intro");
    expect(stub.metadata?.artwork[0]?.src).toBe(
      `${window.location.origin}/api/v1/series/series-1/cover`,
    );
    expect(stub.setPositionState).toHaveBeenCalled();

    act(() => stub.handlers.pause?.({ action: "pause" }));
    expect(view.engine.pauseCalls).toBe(1);

    act(() => stub.handlers.play?.({ action: "play" }));
    expect(view.engine.playCalls).toBe(2);

    act(() => stub.handlers.nexttrack?.({ action: "nexttrack" }));
    expect(view.engine.seekCalls.at(-1)).toBe(30);

    act(() => stub.handlers.previoustrack?.({ action: "previoustrack" }));
    expect(view.engine.seekCalls.at(-1)).toBe(0);

    act(() => stub.handlers.seekto?.({ action: "seekto", seekTime: 12 }));
    expect(view.engine.seekCalls.at(-1)).toBe(12);
  });

  it("updates the document title while playing", async () => {
    const view = await renderPlayer();
    expect(document.title).toBe("▶ First episode — Deep Dive");

    await view.user.click(screen.getByRole("button", { name: "Pause" }));
    expect(document.title).toBe("First episode — Deep Dive");
  });

  it("shows the chapter image and chapter link for the active chapter", async () => {
    const chapters: ChapterDTO[] = [
      {
        start_seconds: 0,
        title: "Intro",
        toc: true,
        image_url: "/images/intro.png",
        url: "https://example.com/intro",
      },
      { start_seconds: 30, title: "Main topic", toc: true },
    ];
    const { user } = await renderPlayer({ chapters });

    expect(screen.getByRole("link", { name: /Open chapter link/ })).toHaveAttribute(
      "href",
      "https://example.com/intro",
    );
    const image = document.querySelector(".player-artwork img");
    expect(image).toHaveAttribute("src", "/images/intro.png");

    await user.click(screen.getByRole("button", { name: "Chapter 2: Main topic" }));
    expect(screen.queryByRole("link", { name: /Open chapter link/ })).not.toBeInTheDocument();
    expect(document.querySelector(".player-artwork img")).toBeNull();
  });
});
