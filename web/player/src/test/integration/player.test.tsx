import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ChapterDTO, Episode, Progress } from "../../api/types";
import { seedEpisode, seedProgress, seedSeries } from "../handlers";
import { makeMediaSessionStub } from "../mediaSession";
import { renderApp } from "../renderApp";

interface PlayerSetup {
  positionSeconds?: number;
  chapters?: ChapterDTO[];
  cover?: boolean;
}

function storedProgress(): Record<string, Progress> {
  const raw = window.localStorage.getItem("podcast.player.progress");
  return raw ? (JSON.parse(raw) as Record<string, Progress>) : {};
}

function progressFor(episode: Episode): Progress | undefined {
  return storedProgress()[episode.id];
}

async function renderPlayer(setup: PlayerSetup = {}) {
  const series = seedSeries({
    slug: "deep-dive",
    title: "Deep Dive",
    cover_url: setup.cover ? "/covers/cover.png" : null,
  });
  const first = seedEpisode(series.slug, {
    id: "episode-1",
    number: 1,
    title: "First episode",
    audio: { url: "/media/first.mp3", duration_seconds: 100 },
    ...(setup.chapters ? { chapters: setup.chapters } : {}),
  });
  const second = seedEpisode(series.slug, {
    id: "episode-2",
    number: 2,
    title: "Second episode",
    audio: { url: "/media/second.mp3", duration_seconds: 100 },
  });
  if (setup.positionSeconds) seedProgress(first.id, { position_seconds: setup.positionSeconds });

  const view = renderApp("/series/deep-dive");
  if (setup.positionSeconds) {
    // Wait until the progress query has landed so the resume position is known.
    await screen.findByRole("button", { name: /^Resume: Episode 1/ });
  }
  await view.user.click(await screen.findByRole("button", { name: "First episode" }));
  await screen.findByText("Now playing");
  // Chapters are fetched from the episode document after playback starts.
  await screen.findByText(/^Chapter \d of 2$/);
  return { ...view, series, first, second };
}

function slider() {
  return screen.getByRole("slider", { name: "Playback position" });
}

describe("player", () => {
  it("starts the episode at its saved position and shows the player", async () => {
    const { engine, first } = await renderPlayer({ positionSeconds: 42 });

    expect(engine.loadCalls).toHaveLength(1);
    expect(engine.loadCalls[0]).toEqual({ source: "/media/first.mp3", startAt: 42 });
    expect(progressFor(first)?.position_seconds).toBe(42);
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
    const { engine, user, first } = await renderPlayer({ positionSeconds: 42 });

    await user.click(screen.getByRole("button", { name: "Back 15 seconds" }));
    expect(engine.seekCalls.at(-1)).toBe(27);
    expect(slider()).toHaveAttribute("aria-valuenow", "27");

    await user.click(screen.getByRole("button", { name: "Forward 30 seconds" }));
    expect(engine.seekCalls.at(-1)).toBe(57);
    expect(slider()).toHaveAttribute("aria-valuenow", "57");

    await waitFor(() => expect(progressFor(first)?.position_seconds).toBe(57));
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
    const second = renderApp("/series/deep-dive");
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
    const { first } = view;
    vi.useFakeTimers({ toFake: ["Date"] });
    const base = Date.now();

    act(() => {
      view.engine.emitTime(5, 100);
    });
    expect(progressFor(first)?.position_seconds ?? 0).toBe(0);

    vi.setSystemTime(new Date(base + 10_000));
    act(() => {
      view.engine.emitTime(15, 100);
    });
    await waitFor(() => expect(progressFor(first)?.position_seconds).toBe(15));

    act(() => {
      view.engine.emitTime(25, 100);
    });
    expect(progressFor(first)?.position_seconds).toBe(15);

    vi.setSystemTime(new Date(base + 20_000));
    act(() => {
      view.engine.emitTime(35, 100);
    });
    await waitFor(() => expect(progressFor(first)?.position_seconds).toBe(35));

    fireEvent.click(screen.getByRole("button", { name: "Pause" }));
    await waitFor(() => expect(progressFor(first)?.position_seconds).toBe(35));

    fireEvent.click(screen.getByRole("button", { name: "Back 15 seconds" }));
    await waitFor(() => expect(progressFor(first)?.position_seconds).toBe(20));
  });

  it("marks the episode played and starts the next one when it ends", async () => {
    const view = await renderPlayer();
    const { first } = view;

    act(() => {
      view.engine.emitEnded();
    });

    await waitFor(() => expect(view.engine.loadCalls).toHaveLength(2));
    expect(view.engine.loadCalls[1]).toEqual({ source: "/media/second.mp3", startAt: 0 });
    await waitFor(() => expect(progressFor(first)?.completed).toBe(true));
    expect(await screen.findByRole("heading", { name: "Second episode" })).toBeInTheDocument();
    expect(screen.getByText("Chapter 1 of 2")).toBeInTheDocument();
  });

  it("restarts a finished episode from the beginning and clears the played markers", async () => {
    const view = await renderPlayer();
    const { first } = view;
    await view.user.click(screen.getByRole("checkbox", { name: "Autoplay next episode" }));
    act(() => {
      view.engine.emitTime(95, 100);
    });
    act(() => {
      view.engine.emitEnded();
    });
    await waitFor(() => expect(progressFor(first)?.completed).toBe(true));
    const playsBefore = view.engine.playCalls;

    await view.user.click(screen.getByRole("button", { name: "Restart from the beginning" }));

    expect(view.engine.seekCalls.at(-1)).toBe(0);
    expect(view.engine.playCalls).toBe(playsBefore + 1);
    expect(progressFor(first)).toBeUndefined();
    expect(slider()).toHaveAttribute("aria-valuenow", "0");
    expect(screen.getByText("Chapter 1 of 2")).toBeInTheDocument();
    const chapterList = screen.getByRole("list", { name: "Chapters" });
    expect(chapterList.querySelector(".chapter-row.is-played")).toBeNull();
  });

  it("does not auto-play when the toggle is off", async () => {
    const view = await renderPlayer();
    const { first } = view;
    await view.user.click(screen.getByRole("checkbox", { name: "Autoplay next episode" }));

    act(() => {
      view.engine.emitEnded();
    });

    await waitFor(() => expect(progressFor(first)?.completed).toBe(true));
    expect(view.engine.loadCalls).toHaveLength(1);
    expect(screen.getByRole("heading", { name: "First episode" })).toBeInTheDocument();
  });

  it("plays the up-next episode from the queue card", async () => {
    const view = await renderPlayer();
    await view.user.click(screen.getByRole("button", { name: "Play next: Second episode" }));

    expect(view.engine.loadCalls.at(-1)?.source).toBe("/media/second.mp3");
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
    expect(stub.metadata?.artwork[0]?.src).toBe(`${window.location.origin}/covers/cover.png`);
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
