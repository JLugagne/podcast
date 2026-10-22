import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { seedEpisode, seedSeries, store } from "../handlers";
import { renderApp } from "../renderApp";

function deleteSection(): HTMLElement {
  const heading = screen.getByRole("heading", { name: "Delete series" });
  return heading.closest("section") as HTMLElement;
}

describe("series form", () => {
  it("derives the slug from the title and creates the series", async () => {
    const { user } = renderApp("/series/new");
    await screen.findByRole("heading", { name: "New series" });

    await user.type(screen.getByLabelText(/^Title/), "My Great Show");
    expect((screen.getByLabelText("Slug") as HTMLInputElement).value).toBe("my-great-show");

    await user.type(screen.getByLabelText("Author"), "Ada");
    await user.type(screen.getByLabelText("Description"), "A show about things.");
    await user.click(screen.getByRole("button", { name: "Create series" }));

    expect(await screen.findByText("No episodes yet")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "My Great Show" })).toBeInTheDocument();
    expect(store.series).toHaveLength(1);
    expect(store.series[0]).toMatchObject({
      slug: "my-great-show",
      author: "Ada",
      description: "A show about things.",
    });
  });

  it("keeps a manually edited slug when the title changes", async () => {
    const { user } = renderApp("/series/new");
    await screen.findByRole("heading", { name: "New series" });

    const title = screen.getByLabelText(/^Title/);
    await user.type(title, "My Great Show");

    const slug = screen.getByLabelText("Slug");
    await user.clear(slug);
    await user.type(slug, "custom-slug");
    await user.type(title, " Again");
    expect((screen.getByLabelText("Slug") as HTMLInputElement).value).toBe("custom-slug");

    await user.click(screen.getByRole("button", { name: "Create series" }));

    expect(await screen.findByText("No episodes yet")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "My Great Show Again" })).toBeInTheDocument();
    expect(store.series[0].slug).toBe("custom-slug");
  });

  it("sends the accent color chosen with the picker", async () => {
    const { user } = renderApp("/series/new");
    await screen.findByRole("heading", { name: "New series" });

    await user.type(screen.getByLabelText(/^Title/), "Colorful Show");
    fireEvent.change(screen.getByLabelText("Pick the accent color"), {
      target: { value: "#00ff00" },
    });
    expect((screen.getByLabelText("Accent color") as HTMLInputElement).value).toBe("#00ff00");

    const accentText = screen.getByLabelText("Accent color");
    await user.clear(accentText);
    await user.type(accentText, "#123456");
    expect((screen.getByLabelText("Pick the accent color") as HTMLInputElement).value).toBe(
      "#123456",
    );

    await user.click(screen.getByRole("button", { name: "Create series" }));
    expect(await screen.findByText("No episodes yet")).toBeInTheDocument();
    expect(store.series[0].accent_color).toBe("#123456");
  });

  it("returns to the series detail when cancelling the form", async () => {
    seedSeries({ id: "series-1", title: "Deep Dive" });
    const { user } = renderApp("/series/series-1/edit");
    await screen.findByLabelText(/^Title/);

    await user.click(screen.getByRole("link", { name: "Cancel" }));

    expect(await screen.findByText("No episodes yet")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Deep Dive" })).toBeInTheDocument();
  });

  it("saves an edited series with its version", async () => {
    seedSeries({ id: "series-1", title: "Old Title", version: 3 });
    const { user } = renderApp("/series/series-1/edit");

    const title = await screen.findByLabelText(/^Title/);
    await user.clear(title);
    await user.type(title, "New Title");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByText("No episodes yet")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "New Title" })).toBeInTheDocument();
    const request = store.requests.find(
      (entry) => entry.method === "PUT" && entry.path === "/api/v1/series/series-1",
    );
    expect(request?.body).toMatchObject({ title: "New Title", version: 3 });
    expect(store.series[0].version).toBe(4);
  });

  it("shows a reload hint when the series was changed elsewhere", async () => {
    const series = seedSeries({ id: "series-1", title: "Old Title", version: 1 });
    const { user } = renderApp("/series/series-1/edit");

    const title = await screen.findByLabelText(/^Title/);
    series.version = 2;

    await user.clear(title);
    await user.type(title, "New Title");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByText("This series changed elsewhere")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reload the page" })).toBeInTheDocument();
  });

  it("uploads the cover as multipart and removes it", async () => {
    seedSeries({ id: "series-1", title: "Show" });
    const { user } = renderApp("/series/series-1/edit");

    const input = await screen.findByLabelText(/upload cover image/i);
    await user.upload(input, new File(["png"], "cover.png", { type: "image/png" }));

    await screen.findByRole("button", { name: "Remove cover" });
    const request = store.requests.find(
      (entry) => entry.method === "PUT" && entry.path === "/api/v1/series/series-1/cover",
    );
    expect(request?.contentType).toContain("multipart/form-data");
    expect(request?.body).toMatchObject({ field: "file", fileName: "cover.png", size: 3 });
    expect(store.series[0].cover_url).toBe("/api/v1/series/series-1/cover");
    const preview = document.querySelector(".cover-tile img");
    expect(preview).not.toBeNull();
    expect(preview?.getAttribute("src")).toContain("/api/v1/series/series-1/cover?v=");

    await user.click(screen.getByRole("button", { name: "Remove cover" }));
    expect(await screen.findByText(/No cover uploaded/)).toBeInTheDocument();
    expect(store.series[0].cover_url).toBeNull();
  });

  it("disables series deletion while episodes exist", async () => {
    seedSeries({ id: "series-1", title: "Show" });
    seedEpisode("series-1");

    renderApp("/series/series-1/edit");
    await screen.findByLabelText(/^Title/);

    const button = within(deleteSection()).getByRole("button", { name: "Delete series" });
    expect(button).toBeDisabled();
    expect(screen.getByText(/still has 1 episode/)).toBeInTheDocument();
  });

  it("deletes a series after confirmation", async () => {
    seedSeries({ id: "series-1", title: "Show" });
    const { user } = renderApp("/series/series-1/edit");
    await screen.findByLabelText(/^Title/);

    await user.click(within(deleteSection()).getByRole("button", { name: "Delete series" }));
    const dialog = screen.getByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "Delete series" }));

    await waitFor(() => expect(store.series).toHaveLength(0));
    expect(await screen.findByText("No series yet")).toBeInTheDocument();
  });
});
