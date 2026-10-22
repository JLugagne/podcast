import { screen, waitFor } from "@testing-library/react";
import { HttpResponse, http } from "msw/http";
import { describe, expect, it } from "vitest";
import { seedEpisode, seedProgress, seedSeries } from "../handlers";
import { renderApp } from "../renderApp";
import { server } from "../server";

describe("library", () => {
  it("shows an empty state explaining publishing when nothing is published", async () => {
    seedSeries({ id: "series-1", title: "Draft Show" });

    renderApp("/");

    expect(await screen.findByText("Nothing published yet")).toBeInTheDocument();
    expect(
      screen.getByText(/Episodes appear here once they are published in the admin/),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open the admin" })).toHaveAttribute("href", "/admin/");
  });

  it("lists series with their published and unplayed counts", async () => {
    const alpha = seedSeries({ id: "series-1", title: "Alpha Show", author: "Ada" });
    seedSeries({ id: "series-2", title: "Draft Show", author: "Bob" });
    const first = seedEpisode(alpha.id, { id: "episode-1", title: "First" });
    seedEpisode(alpha.id, { id: "episode-2", title: "Second" });
    seedProgress(first.id, { position_seconds: 100, completed: true });

    renderApp("/");

    expect(await screen.findByRole("link", { name: /Alpha Show/ })).toHaveAttribute(
      "href",
      "/series/series-1",
    );
    expect(screen.getByText("Ada")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText(/2 episodes · 1 unplayed/)).toBeInTheDocument());
    expect(screen.queryByText("Draft Show")).not.toBeInTheDocument();
  });

  it("shows continue listening items with the current chapter and time left", async () => {
    const series = seedSeries({ id: "series-1", title: "Deep Dive", slug: "deep-dive" });
    const episode = seedEpisode(series.id, {
      id: "episode-1",
      number: 2,
      title: "The long one",
      audio: {
        url: "/api/v1/episodes/episode-1/audio",
        content_type: "audio/mpeg",
        size_bytes: 2048,
        duration_seconds: 600,
        original_name: "episode.mp3",
      },
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
    const series = seedSeries({ id: "series-1", title: "Deep Dive" });
    const episode = seedEpisode(series.id, { id: "episode-1", title: "The long one" });
    seedProgress(episode.id, { position_seconds: 42 });

    const { user, engine } = renderApp("/");
    const card = await screen.findByRole("button", {
      name: "Resume Deep Dive episode 1: The long one",
    });
    await waitFor(() => expect(card).toBeEnabled());

    await user.click(card);

    expect(engine.loadCalls).toHaveLength(1);
    expect(engine.loadCalls[0]).toEqual({
      source: "/api/v1/episodes/episode-1/audio",
      startAt: 42,
    });
    expect(await screen.findByRole("heading", { name: "The long one" })).toBeInTheDocument();
  });

  it("shows the API error and recovers on retry", async () => {
    const series = seedSeries({ id: "series-1", title: "Alpha Show" });
    seedEpisode(series.id, { id: "episode-1", title: "First" });
    server.use(
      http.get("/api/v1/series", () =>
        HttpResponse.json({ code: "INTERNAL", message: "The database is down." }, { status: 500 }),
      ),
    );

    const { user } = renderApp("/");
    expect(await screen.findByText("Could not load your library")).toBeInTheDocument();
    expect(screen.getByText("The database is down.")).toBeInTheDocument();

    server.resetHandlers();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByText("Alpha Show")).toBeInTheDocument();
  });
});
