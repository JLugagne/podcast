import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { ChapterDTO } from "../../api/types";
import { seedEpisode, seedProgress, seedSeries } from "../handlers";
import { renderApp } from "../renderApp";

const CHAPTERS: ChapterDTO[] = [
  { start_seconds: 0, title: "Intro", toc: true },
  { start_seconds: 30, title: "Main topic", toc: true },
];

function seedSeriesWithEpisodes() {
  const series = seedSeries({
    slug: "deep-dive",
    title: "Deep Dive",
    author: "Ada",
    description: "Long conversations about small things.",
  });
  const first = seedEpisode(series.slug, {
    id: "episode-1",
    number: 1,
    title: "First episode",
    audio: { url: "/media/first.mp3", duration_seconds: 300 },
    chapters: CHAPTERS,
  });
  const second = seedEpisode(series.slug, {
    id: "episode-2",
    number: 2,
    title: "Second episode",
    audio: { url: "/media/second.mp3", duration_seconds: 200 },
    chapters: CHAPTERS,
  });
  const third = seedEpisode(series.slug, {
    id: "episode-3",
    number: 3,
    title: "Third episode",
    audio: { url: "/media/third.mp3", duration_seconds: 120 },
    chapters: CHAPTERS,
  });
  return { series, first, second, third };
}

function episodeRows(title = "Deep Dive") {
  const list = screen.getByRole("list", { name: `Episodes of ${title}` });
  return within(list).getAllByRole("listitem");
}

function storedProgress(): Record<string, { completed: boolean }> {
  const raw = window.localStorage.getItem("podcast.player.progress");
  return raw ? (JSON.parse(raw) as Record<string, { completed: boolean }>) : {};
}

describe("series page", () => {
  it("renders the header and episode states", async () => {
    const { first, second } = seedSeriesWithEpisodes();
    seedProgress(first.id, { position_seconds: 60 });
    seedProgress(second.id, { position_seconds: 200, completed: true });

    renderApp("/series/deep-dive");

    expect(await screen.findByRole("heading", { name: "Deep Dive" })).toBeInTheDocument();
    expect(screen.getByText("Ada")).toBeInTheDocument();
    expect(screen.getByText("Long conversations about small things.")).toBeInTheDocument();
    expect(screen.getByText("3 episodes · 10:20")).toBeInTheDocument();

    await waitFor(() => expect(episodeRows()).toHaveLength(3));
    expect(await screen.findByText("4 min left")).toBeInTheDocument();
    const rows = episodeRows();
    expect(within(rows[0]).getByText("5:00")).toBeInTheDocument();
    expect(within(rows[0]).getByText("2 chapters")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Played")).toBeInTheDocument();
    expect(within(rows[2]).getByText("New")).toBeInTheDocument();
  });

  it("plays the latest episode from the primary button", async () => {
    seedSeriesWithEpisodes();
    const { user, engine } = renderApp("/series/deep-dive");

    const primary = await screen.findByRole("button", { name: /^Play latest/ });
    await user.click(primary);

    expect(engine.loadCalls[0]).toEqual({ source: "/media/third.mp3", startAt: 0 });
    expect(await screen.findByRole("heading", { name: "Third episode" })).toBeInTheDocument();
  });

  it("resumes the most recently played episode", async () => {
    const { first, second } = seedSeriesWithEpisodes();
    seedProgress(first.id, { position_seconds: 45 });
    seedProgress(second.id, { position_seconds: 12 });

    const { user, engine } = renderApp("/series/deep-dive");

    const primary = await screen.findByRole("button", { name: /^Resume: Episode 2/ });
    await user.click(primary);

    expect(engine.loadCalls[0]).toEqual({ source: "/media/second.mp3", startAt: 12 });
  });

  it("toggles an episode in place and opens the player from its title", async () => {
    seedSeriesWithEpisodes();
    const { user, engine } = renderApp("/series/deep-dive");

    const play = await screen.findByRole("button", { name: "Play episode 1: First episode" });
    await user.click(play);

    expect(engine.loadCalls[0].source).toBe("/media/first.mp3");
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
    const { user } = renderApp("/series/deep-dive");
    await waitFor(() => expect(episodeRows()).toHaveLength(3));

    await user.click(screen.getByRole("button", { name: "More actions for episode 1" }));
    await user.click(
      within(screen.getByRole("menu")).getByRole("menuitem", { name: "Mark as played" }),
    );

    await waitFor(() => expect(storedProgress()[first.id]?.completed).toBe(true));
    await waitFor(() => expect(within(episodeRows()[0]).getByText("Played")).toBeInTheDocument());

    await user.click(screen.getByRole("button", { name: "More actions for episode 1" }));
    await user.click(
      within(screen.getByRole("menu")).getByRole("menuitem", { name: "Mark as unplayed" }),
    );

    await waitFor(() => expect(storedProgress()[first.id]).toBeUndefined());
    await waitFor(() => expect(within(episodeRows()[0]).getByText("New")).toBeInTheDocument());
  });

  it("shows episode artwork with the series cover as fallback", async () => {
    const series = seedSeries({
      slug: "art-show",
      title: "Art Show",
      cover_url: "https://covers.test/series.jpg",
    });
    seedEpisode(series.slug, {
      id: "with-art",
      number: 1,
      title: "With art",
      image_url: "https://covers.test/episode.jpg",
    });
    seedEpisode(series.slug, { id: "no-art", number: 2, title: "No art" });

    renderApp("/series/art-show");
    await waitFor(() => expect(episodeRows("Art Show")).toHaveLength(2));

    const images = Array.from(document.querySelectorAll(".episode-row-cover img"));
    expect(images.map((image) => image.getAttribute("src"))).toEqual([
      "https://covers.test/episode.jpg",
      "https://covers.test/series.jpg",
    ]);
  });

  it("switches episodes between list and grid and remembers the choice", async () => {
    seedSeriesWithEpisodes();
    const { user } = renderApp("/series/deep-dive");
    await waitFor(() => expect(episodeRows()).toHaveLength(3));

    expect(screen.getByRole("list", { name: "Episodes of Deep Dive" })).not.toHaveClass("is-grid");
    expect(screen.getByRole("button", { name: "Grid" })).toHaveAttribute("aria-pressed", "false");

    await user.click(screen.getByRole("button", { name: "Grid" }));

    expect(screen.getByRole("list", { name: "Episodes of Deep Dive" })).toHaveClass("is-grid");
    expect(screen.getByRole("button", { name: "Grid" })).toHaveAttribute("aria-pressed", "true");
    expect(episodeRows()).toHaveLength(3);
    expect(window.localStorage.getItem("podcast.player.episode-view")).toBe("grid");

    await user.click(screen.getByRole("button", { name: "List" }));

    expect(screen.getByRole("list", { name: "Episodes of Deep Dive" })).not.toHaveClass("is-grid");
    expect(window.localStorage.getItem("podcast.player.episode-view")).toBe("list");
  });

  it("opens a series in the remembered view", async () => {
    seedSeriesWithEpisodes();
    window.localStorage.setItem("podcast.player.episode-view", "grid");

    renderApp("/series/deep-dive");

    expect(await screen.findByRole("list", { name: "Episodes of Deep Dive" })).toHaveClass(
      "is-grid",
    );
  });

  it("shows a not found state for an unknown series", async () => {
    seedSeriesWithEpisodes();
    renderApp("/series/missing");

    expect(await screen.findByText("Series not found")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to the library" })).toHaveAttribute("href", "/");
  });
});
