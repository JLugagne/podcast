import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AudioFileDTO } from "../../api/types";
import { seedEpisode, seedProgress, seedSeries, store } from "../handlers";
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

function seedSeriesWithEpisodes() {
  const series = seedSeries({
    id: "series-1",
    title: "Deep Dive",
    author: "Ada",
    slug: "deep-dive",
    description: "Long conversations about small things.",
  });
  const first = seedEpisode(series.id, {
    id: "episode-1",
    number: 1,
    title: "First episode",
    audio: audioFile("episode-1", 300),
  });
  const second = seedEpisode(series.id, {
    id: "episode-2",
    number: 2,
    title: "Second episode",
    audio: audioFile("episode-2", 200),
  });
  const third = seedEpisode(series.id, {
    id: "episode-3",
    number: 3,
    title: "Third episode",
    audio: audioFile("episode-3", 120),
  });
  return { series, first, second, third };
}

function episodeRows() {
  const list = screen.getByRole("list", { name: "Episodes of Deep Dive" });
  return within(list).getAllByRole("listitem");
}

describe("series page", () => {
  it("renders the header, feed URL and episode states", async () => {
    const { first, second } = seedSeriesWithEpisodes();
    seedProgress(first.id, { position_seconds: 60 });
    seedProgress(second.id, { position_seconds: 200, completed: true });

    renderApp("/series/series-1");

    expect(await screen.findByRole("heading", { name: "Deep Dive" })).toBeInTheDocument();
    expect(screen.getByText("Ada")).toBeInTheDocument();
    expect(screen.getByText("Long conversations about small things.")).toBeInTheDocument();
    expect(screen.getByText("3 episodes · 10:20")).toBeInTheDocument();
    expect(screen.getByText(`${window.location.origin}/feeds/deep-dive.xml`)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Copy RSS feed/ })).toBeInTheDocument();

    await waitFor(() => expect(episodeRows()).toHaveLength(3));
    expect(await screen.findByText("4 min left")).toBeInTheDocument();
    const rows = episodeRows();
    expect(within(rows[0]).getByText("5:00")).toBeInTheDocument();
    expect(within(rows[0]).getByText("2 chapters")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Played")).toBeInTheDocument();
    expect(within(rows[2]).getByText("New")).toBeInTheDocument();
  });

  it("copies the absolute RSS feed URL to the clipboard", async () => {
    seedSeriesWithEpisodes();

    const { user } = renderApp("/series/series-1");
    await screen.findByRole("heading", { name: "Deep Dive" });
    // user-event installs its own clipboard stub on setup, so replace it afterwards.
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    await user.click(screen.getByRole("button", { name: /Copy RSS feed/ }));

    expect(writeText).toHaveBeenCalledWith(`${window.location.origin}/feeds/deep-dive.xml`);
    expect(await screen.findByText("Copied")).toBeInTheDocument();
  });

  it("plays the latest episode from the primary button", async () => {
    seedSeriesWithEpisodes();
    const { user, engine } = renderApp("/series/series-1");

    const primary = await screen.findByRole("button", { name: /^Play latest/ });
    await user.click(primary);

    expect(engine.loadCalls[0]).toEqual({
      source: "/api/v1/episodes/episode-3/audio",
      startAt: 0,
    });
    expect(await screen.findByRole("heading", { name: "Third episode" })).toBeInTheDocument();
  });

  it("resumes the most recently played episode", async () => {
    const { first, second } = seedSeriesWithEpisodes();
    seedProgress(first.id, { position_seconds: 45 });
    seedProgress(second.id, { position_seconds: 12 });

    const { user, engine } = renderApp("/series/series-1");

    const primary = await screen.findByRole("button", { name: /^Resume: Episode 2/ });
    await user.click(primary);

    expect(engine.loadCalls[0]).toEqual({
      source: "/api/v1/episodes/episode-2/audio",
      startAt: 12,
    });
  });

  it("toggles an episode in place and opens the player from its title", async () => {
    seedSeriesWithEpisodes();
    const { user, engine } = renderApp("/series/series-1");

    const play = await screen.findByRole("button", { name: "Play episode 1: First episode" });
    await user.click(play);

    expect(engine.loadCalls[0].source).toBe("/api/v1/episodes/episode-1/audio");
    expect(engine.playCalls).toBe(1);
    expect(
      screen.getByRole("button", { name: "Pause episode 1: First episode" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Deep Dive" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Pause episode 1: First episode" }));
    expect(engine.pauseCalls).toBe(1);
    expect(
      screen.getByRole("button", { name: "Play episode 1: First episode" }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "First episode" }));
    expect(await screen.findByRole("heading", { name: "First episode" })).toBeInTheDocument();
  });

  it("marks an episode as played and unplayed from the row menu", async () => {
    const { first } = seedSeriesWithEpisodes();
    const { user } = renderApp("/series/series-1");
    await waitFor(() => expect(episodeRows()).toHaveLength(3));

    await user.click(screen.getByRole("button", { name: "More actions for episode 1" }));
    await user.click(
      within(screen.getByRole("menu")).getByRole("menuitem", { name: "Mark as played" }),
    );

    await waitFor(() =>
      expect(
        store.requests.some(
          (request) =>
            request.method === "POST" &&
            request.path === "/api/v1/listening/progress/episode-1/played",
        ),
      ).toBe(true),
    );
    await waitFor(() => expect(within(episodeRows()[0]).getByText("Played")).toBeInTheDocument());

    seedProgress(first.id, { position_seconds: 300, completed: true });
    await user.click(screen.getByRole("button", { name: "More actions for episode 1" }));
    await user.click(
      within(screen.getByRole("menu")).getByRole("menuitem", { name: "Mark as unplayed" }),
    );

    await waitFor(() =>
      expect(
        store.requests.some(
          (request) =>
            request.method === "DELETE" && request.path === "/api/v1/listening/progress/episode-1",
        ),
      ).toBe(true),
    );
  });

  it("shows a not found state for an unknown series", async () => {
    renderApp("/series/missing");

    expect(await screen.findByText("Series not found")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to the library" })).toHaveAttribute("href", "/");
  });
});
