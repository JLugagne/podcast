import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { seedEpisode, seedSeries } from "../handlers";
import { renderApp } from "../renderApp";

describe("series detail", () => {
  it("renders the episode table with status, duration, chapters and audio", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive", author: "Ada" });
    seedEpisode("series-1", {
      id: "episode-1",
      number: 1,
      title: "First",
      audio: {
        url: "/api/v1/episodes/episode-1/audio",
        content_type: "audio/mpeg",
        size_bytes: 2048,
        duration_seconds: 95,
        original_name: "first.mp3",
      },
      chapters: [
        { start_seconds: 0, title: "Intro", toc: true },
        { start_seconds: 10, title: "Body", toc: true },
      ],
    });
    seedEpisode("series-1", {
      id: "episode-2",
      number: 2,
      title: "Second",
      status: "published",
      published_at: "2026-01-02T00:00:00.000Z",
    });

    renderApp("/series/series-1");

    expect(await screen.findByRole("heading", { name: "Deep Dive" })).toBeInTheDocument();

    const table = screen.getByRole("table", { name: "Episodes of Deep Dive" });
    const rows = within(table).getAllByRole("row");
    expect(rows).toHaveLength(3);

    const firstRow = rows[1];
    expect(within(firstRow).getByRole("link", { name: "First" })).toHaveAttribute(
      "href",
      "/episodes/episode-1",
    );
    expect(within(firstRow).getByText("1:35")).toBeInTheDocument();
    expect(within(firstRow).getByText("2")).toBeInTheDocument();
    expect(within(firstRow).getByText("Audio uploaded")).toBeInTheDocument();
    expect(within(firstRow).getByText("Draft")).toBeInTheDocument();

    const secondRow = rows[2];
    expect(within(secondRow).getByText("Published")).toBeInTheDocument();
    expect(within(secondRow).getByText("No audio")).toBeInTheDocument();
    expect(within(secondRow).getByText("0")).toBeInTheDocument();

    expect(screen.getByRole("link", { name: "Edit series" })).toHaveAttribute(
      "href",
      "/series/series-1/edit",
    );
    expect(screen.getByRole("link", { name: "New episode" })).toHaveAttribute(
      "href",
      "/series/series-1/episodes/new",
    );
  });

  it("copies the absolute RSS feed URL", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive", slug: "deep-dive" });

    const { user } = renderApp("/series/series-1");

    expect(await screen.findByRole("heading", { name: "Deep Dive" })).toBeInTheDocument();
    const absoluteUrl = `${window.location.origin}/feeds/deep-dive.xml`;
    expect(screen.getByText(absoluteUrl)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Copy RSS feed URL" }));

    // user-event installs a clipboard stub on navigator.clipboard.
    await expect(navigator.clipboard.readText()).resolves.toBe(absoluteUrl);
    expect(await screen.findByRole("status")).toHaveTextContent("Copied to clipboard");
  });

  it("shows an empty state when the series has no episodes", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive", author: "Ada" });

    renderApp("/series/series-1");

    expect(await screen.findByText("No episodes yet")).toBeInTheDocument();
    for (const link of screen.getAllByRole("link", { name: "New episode" })) {
      expect(link).toHaveAttribute("href", "/series/series-1/episodes/new");
    }
  });
});
