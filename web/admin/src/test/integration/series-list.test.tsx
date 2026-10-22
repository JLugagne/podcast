import { screen, waitFor } from "@testing-library/react";
import { HttpResponse, http } from "msw/http";
import { describe, expect, it } from "vitest";
import { seedEpisode, seedSeries } from "../handlers";
import { renderApp } from "../renderApp";
import { server } from "../server";

describe("series list", () => {
  it("shows an empty state with a link to create a series", async () => {
    renderApp("/");

    expect(await screen.findByText("No series yet")).toBeInTheDocument();
    const links = screen.getAllByRole("link", { name: "New series" });
    expect(links.length).toBeGreaterThan(0);
    expect(links[0]).toHaveAttribute("href", "/series/new");
  });

  it("renders series cards with counts and links", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive", author: "Ada", slug: "deep-dive" });
    seedSeries({ id: "series-2", title: "Beta Show", author: "Bob", slug: "beta-show" });
    seedEpisode("series-1");
    seedEpisode("series-1", {
      status: "published",
      published_at: "2026-01-02T00:00:00.000Z",
      audio: {
        url: "/api/v1/episodes/episode-2/audio",
        content_type: "audio/mpeg",
        size_bytes: 1024,
        duration_seconds: 95,
        original_name: "one.mp3",
      },
    });

    renderApp("/");

    expect(await screen.findByText("Deep Dive")).toBeInTheDocument();
    expect(screen.getByText("by Ada")).toBeInTheDocument();
    expect(screen.getByText("2 episodes")).toBeInTheDocument();
    expect(screen.getByText("1 published")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Deep Dive/ })).toHaveAttribute(
      "href",
      "/series/series-1",
    );
  });

  it("filters series as the user types and clears the search", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive", author: "Ada", slug: "deep-dive" });
    seedSeries({ id: "series-2", title: "Beta Show", author: "Bob", slug: "beta-show" });

    const { user } = renderApp("/");
    await screen.findByText("Deep Dive");

    await user.type(screen.getByLabelText("Search series"), "beta");
    expect(await screen.findByText("Beta Show")).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByText("Deep Dive")).not.toBeInTheDocument());

    await user.clear(screen.getByLabelText("Search series"));
    await user.type(screen.getByLabelText("Search series"), "zzz");
    expect(await screen.findByText(/No series match/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(await screen.findByText("Deep Dive")).toBeInTheDocument();
  });

  it("shows the API error and recovers on retry", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive", author: "Ada", slug: "deep-dive" });
    server.use(
      http.get("/api/v1/series", () =>
        HttpResponse.json({ code: "INTERNAL", message: "The database is down." }, { status: 500 }),
      ),
    );

    const { user } = renderApp("/");
    expect(await screen.findByText("Could not load series")).toBeInTheDocument();
    expect(screen.getByText("The database is down.")).toBeInTheDocument();

    server.resetHandlers();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByText("Deep Dive")).toBeInTheDocument();
  });
});
