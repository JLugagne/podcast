import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { seedEpisode, seedSeries, store } from "../handlers";
import { renderApp } from "../renderApp";

describe("create episode", () => {
  it("creates an episode with the next number and opens the editor", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive" });
    seedEpisode("series-1", { number: 1, title: "Existing" });

    const { user } = renderApp("/series/series-1/episodes/new");
    await screen.findByRole("heading", { name: "New episode" });

    await user.type(screen.getByLabelText(/^Title/), "Pilot");
    await user.type(screen.getByLabelText("Description"), "The first one.");
    await user.click(screen.getByRole("button", { name: "Create episode" }));

    expect(await screen.findByRole("heading", { name: "Pilot" })).toBeInTheDocument();

    const request = store.requests.find(
      (entry) => entry.method === "POST" && entry.path === "/api/v1/series/series-1/episodes",
    );
    expect(request?.body).toEqual({ title: "Pilot", description: "The first one." });

    const created = store.episodes.find((episode) => episode.title === "Pilot");
    expect(created?.number).toBe(2);
    expect(created?.status).toBe("draft");
  });

  it("returns to the series detail when cancelling", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive" });
    const { user } = renderApp("/series/series-1/episodes/new");
    await screen.findByRole("heading", { name: "New episode" });

    await user.click(screen.getByRole("link", { name: "Cancel" }));

    expect(await screen.findByText("No episodes yet")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Deep Dive" })).toBeInTheDocument();
  });

  it("sends an explicit episode number when provided", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive" });

    const { user } = renderApp("/series/series-1/episodes/new");
    await screen.findByRole("heading", { name: "New episode" });

    await user.type(screen.getByLabelText(/^Number/), "7");
    await user.type(screen.getByLabelText(/^Title/), "Episode Seven");
    await user.click(screen.getByRole("button", { name: "Create episode" }));

    expect(await screen.findByRole("heading", { name: "Episode Seven" })).toBeInTheDocument();
    expect(store.episodes[0].number).toBe(7);
  });
});
