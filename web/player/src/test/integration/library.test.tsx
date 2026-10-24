import { screen, waitFor } from "@testing-library/react";
import { HttpResponse, http } from "msw/http";
import { describe, expect, it } from "vitest";
import { seedEpisode, seedProgress, seedSeries } from "../handlers";
import { renderApp } from "../renderApp";
import { server } from "../server";

describe("library", () => {
  it("shows an empty state when there is no content", async () => {
    renderApp("/");

    expect(await screen.findByText("Nothing to listen to yet")).toBeInTheDocument();
    expect(
      screen.getByText(/Add a series and an episode JSON to the content directory/),
    ).toBeInTheDocument();
  });

  it("lists series with their episode and unplayed counts", async () => {
    const alpha = seedSeries({ slug: "alpha-show", title: "Alpha Show", author: "Ada" });
    seedSeries({ slug: "empty-show", title: "Empty Show", author: "Bob" });
    const first = seedEpisode(alpha.slug, { id: "episode-1", title: "First" });
    seedEpisode(alpha.slug, { id: "episode-2", title: "Second" });
    seedProgress(first.id, { position_seconds: 100, completed: true });

    renderApp("/");

    expect(await screen.findByRole("link", { name: /Alpha Show/ })).toHaveAttribute(
      "href",
      "/series/alpha-show",
    );
    expect(screen.getByText("Ada")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText(/2 episodes · 1 unplayed/)).toBeInTheDocument());
    expect(screen.queryByText("Empty Show")).not.toBeInTheDocument();
  });

  it("shows continue listening items with the current chapter and time left", async () => {
    const series = seedSeries({ slug: "deep-dive", title: "Deep Dive" });
    const episode = seedEpisode(series.slug, {
      id: "episode-1",
      number: 2,
      title: "The long one",
      audio: { url: "/media/long.mp3", duration_seconds: 600 },
    });
    seedProgress(episode.id, { position_seconds: 60 });

    renderApp("/");

    expect(await screen.findByText("Continue listening")).toBeInTheDocument();
    expect(await screen.findByText("Chapter 2: Main topic")).toBeInTheDocument();
    expect(screen.getByText("Deep Dive · Episode 2")).toBeInTheDocument();
    expect(screen.getByText("9 min left")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Resume Deep Dive episode 2: The long one" }),
    ).toBeInTheDocument();
  });

  it("resumes a continue listening item at its saved position", async () => {
    const series = seedSeries({ slug: "deep-dive", title: "Deep Dive" });
    const episode = seedEpisode(series.slug, {
      id: "episode-1",
      title: "The long one",
      audio: { url: "/media/long.mp3", duration_seconds: 600 },
    });
    seedProgress(episode.id, { position_seconds: 42 });

    const { user, engine } = renderApp("/");
    const card = await screen.findByRole("button", {
      name: "Resume Deep Dive episode 1: The long one",
    });
    await waitFor(() => expect(card).toBeEnabled());

    await user.click(card);

    expect(engine.loadCalls).toHaveLength(1);
    expect(engine.loadCalls[0]).toEqual({ source: "/media/long.mp3", startAt: 42 });
    expect(await screen.findByRole("heading", { name: "The long one" })).toBeInTheDocument();
  });

  it("shows the load error and recovers on retry", async () => {
    const series = seedSeries({ slug: "alpha-show", title: "Alpha Show" });
    seedEpisode(series.slug, { id: "episode-1", title: "First" });
    server.use(
      http.get("*/data/index.json", () => HttpResponse.json({ message: "boom" }, { status: 500 })),
    );

    const { user } = renderApp("/");
    expect(await screen.findByText("Could not load your library")).toBeInTheDocument();
    expect(screen.getByText("The site returned an unexpected response (500).")).toBeInTheDocument();

    server.resetHandlers();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByText("Alpha Show")).toBeInTheDocument();
  });
});
