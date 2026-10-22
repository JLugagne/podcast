import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { AudioFileDTO } from "../../api/types";
import { seedEpisode, seedSeries } from "../handlers";
import { renderApp } from "../renderApp";

function audioFile(id: string): AudioFileDTO {
  return {
    url: `/api/v1/episodes/${id}/audio`,
    content_type: "audio/mpeg",
    size_bytes: 2048,
    duration_seconds: 100,
    original_name: "episode.mp3",
  };
}

describe("mini player", () => {
  it("docks after playback starts, controls playback and opens the player", async () => {
    const series = seedSeries({ id: "series-1", title: "Deep Dive", slug: "deep-dive" });
    seedEpisode(series.id, {
      id: "episode-1",
      number: 1,
      title: "First episode",
      audio: audioFile("episode-1"),
    });

    const { user, engine } = renderApp("/series/series-1");
    await user.click(await screen.findByRole("button", { name: "Play episode 1: First episode" }));

    const mini = screen.getByRole("region", { name: "Mini player" });
    expect(within(mini).getByText("First episode")).toBeInTheDocument();
    expect(within(mini).getByText("Chapter 1: Intro")).toBeInTheDocument();
    expect(within(mini).queryByText("Now playing")).not.toBeInTheDocument();

    await user.click(within(mini).getByRole("button", { name: "Pause" }));
    expect(engine.pauseCalls).toBe(1);
    await user.click(within(mini).getByRole("button", { name: "Play" }));
    expect(engine.playCalls).toBe(2);

    await user.click(within(mini).getByRole("button", { name: "Forward 30 seconds" }));
    expect(engine.seekCalls.at(-1)).toBe(30);
    expect(within(mini).getByText("Chapter 2: Main topic")).toBeInTheDocument();

    await user.click(within(mini).getByRole("link", { name: "Open player: First episode" }));
    expect(await screen.findByText("Now playing")).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Mini player" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Close player" }));
    expect(await screen.findByRole("heading", { name: "Deep Dive" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Mini player" })).toBeInTheDocument();
  });

  it("is not shown before anything is playing", async () => {
    const series = seedSeries({ id: "series-1", title: "Deep Dive", slug: "deep-dive" });
    seedEpisode(series.id, { id: "episode-1", number: 1, audio: audioFile("episode-1") });

    renderApp("/series/series-1");
    await screen.findByRole("heading", { name: "Deep Dive" });
    expect(screen.queryByRole("region", { name: "Mini player" })).not.toBeInTheDocument();
  });
});
