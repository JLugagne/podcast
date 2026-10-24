import { screen, waitFor } from "@testing-library/react";
import { HttpResponse, http } from "msw/http";
import { describe, expect, it, vi } from "vitest";
import { seedEpisode, seedProgress, seedSeries } from "../handlers";
import { renderApp } from "../renderApp";
import { server } from "../server";

describe("library", () => {
  it("shows an empty state when there is no content", async () => {
    renderApp("/");

    expect(await screen.findByText("Nothing to listen to yet")).toBeInTheDocument();
    expect(screen.getByText(/Paste the RSS feed URL of a podcast/)).toBeInTheDocument();
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

  it("adds a podcast from its feed URL and keeps it in localStorage", async () => {
    server.use(http.get("https://feeds.example/show.xml", () => rss("Fresh Show")));

    const { user } = renderApp("/");
    await user.type(
      await screen.findByLabelText("Podcast feed URL"),
      "https://feeds.example/show.xml",
    );
    await user.click(screen.getByRole("button", { name: "Add" }));

    expect(await screen.findByRole("link", { name: /Fresh Show/ })).toBeInTheDocument();
    expect(screen.getByText(/2 episodes/)).toBeInTheDocument();
    expect(window.localStorage.getItem("podcast.player.library")).toContain("Fresh Show");
  });

  it("shows an error when the URL is not a podcast feed", async () => {
    server.use(http.get("https://feeds.example/bad.xml", () => HttpResponse.text("<html></html>")));

    const { user } = renderApp("/");
    await user.type(
      await screen.findByLabelText("Podcast feed URL"),
      "https://feeds.example/bad.xml",
    );
    await user.click(screen.getByRole("button", { name: "Add" }));

    expect(await screen.findByText("Could not add this podcast")).toBeInTheDocument();
    expect(window.localStorage.getItem("podcast.player.library")).toBeNull();
  });

  it("removes a podcast from its page", async () => {
    const series = seedSeries({ slug: "alpha-show", title: "Alpha Show" });
    seedEpisode(series.slug, { id: "episode-1" });
    vi.spyOn(window, "confirm").mockReturnValue(true);

    const { user } = renderApp("/series/alpha-show");
    await user.click(await screen.findByRole("button", { name: "Remove" }));

    expect(await screen.findByText("Nothing to listen to yet")).toBeInTheDocument();
  });

  it("subscribes to every feed of a shared link", async () => {
    server.use(
      http.get("https://feeds.example/one.xml", () => rss("Show One")),
      http.get("https://feeds.example/two.xml", () => rss("Show Two")),
    );

    renderApp(
      "/?feed=https%3A%2F%2Ffeeds.example%2Fone.xml&feed=https%3A%2F%2Ffeeds.example%2Ftwo.xml",
    );

    expect(await screen.findByRole("link", { name: /Show One/ })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Show Two/ })).toBeInTheDocument();
    expect(await screen.findByText("2 podcasts added")).toBeInTheDocument();
    expect(window.localStorage.getItem("podcast.player.library")).toContain("Show Two");
  });

  it("does not subscribe again to feeds already in the library", async () => {
    const series = seedSeries({ slug: "alpha-show", title: "Alpha Show" });
    seedEpisode(series.slug, { id: "episode-1" });
    const refetch = vi.fn(() => rss("Alpha Show"));
    server.use(
      http.get("https://feeds.test/alpha-show.xml", refetch),
      http.get("https://feeds.example/two.xml", () => rss("Show Two")),
    );

    renderApp(
      "/?feed=https%3A%2F%2Ffeeds.test%2Falpha-show.xml&feed=https%3A%2F%2Ffeeds.example%2Ftwo.xml",
    );

    expect(
      await screen.findByText("1 podcast added · 1 podcast already in your library"),
    ).toBeInTheDocument();
    expect(refetch).not.toHaveBeenCalled();
    const stored = JSON.parse(window.localStorage.getItem("podcast.player.library") ?? "[]");
    expect(stored).toHaveLength(2);
  });

  it("adds the working feeds when one feed of the link fails", async () => {
    server.use(
      http.get("https://feeds.example/good.xml", () => rss("Good Show")),
      http.get("https://feeds.example/bad.xml", () => HttpResponse.text("<html></html>")),
    );

    renderApp(
      "/?feed=https%3A%2F%2Ffeeds.example%2Fbad.xml&feed=https%3A%2F%2Ffeeds.example%2Fgood.xml",
    );

    expect(await screen.findByRole("link", { name: /Good Show/ })).toBeInTheDocument();
    expect(
      await screen.findByText("1 podcast added · 1 podcast could not be added"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/bad\.xml: This XML document is not a podcast RSS feed\./),
    ).toBeInTheDocument();
  });

  it("copies a link that subscribes to every podcast in the library", async () => {
    const series = seedSeries({ slug: "alpha-show", title: "Alpha Show" });
    seedEpisode(series.slug, { id: "episode-1" });
    const { user } = renderApp("/");

    const writeText = vi.spyOn(navigator.clipboard, "writeText");
    await user.click(await screen.findByRole("button", { name: "Share library" }));

    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText).toHaveBeenCalledWith(
      "http://localhost:3000/?feed=https%3A%2F%2Ffeeds.test%2Falpha-show.xml",
    );
    expect(await screen.findByRole("button", { name: "Link copied" })).toBeInTheDocument();
  });
});

function rss(title: string): Response {
  return HttpResponse.text(
    `<?xml version="1.0"?><rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"><channel>
      <title>${title}</title><itunes:author>Ada</itunes:author>
      <item><title>One</title><guid>g1</guid><pubDate>Mon, 05 Jan 2026 10:00:00 GMT</pubDate>
        <enclosure url="https://cdn.example/1.mp3" type="audio/mpeg"/><itunes:duration>10:00</itunes:duration></item>
      <item><title>Two</title><guid>g2</guid><pubDate>Mon, 12 Jan 2026 10:00:00 GMT</pubDate>
        <enclosure url="https://cdn.example/2.mp3" type="audio/mpeg"/></item>
    </channel></rss>`,
    { headers: { "content-type": "application/rss+xml" } },
  );
}
