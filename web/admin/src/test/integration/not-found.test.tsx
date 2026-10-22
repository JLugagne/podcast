import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderApp } from "../renderApp";

describe("not found", () => {
  it("shows an error for an unknown series", async () => {
    renderApp("/series/unknown");

    expect(await screen.findByText("Could not load this series")).toBeInTheDocument();
    expect(screen.getByText("The series was not found.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to series" })).toHaveAttribute("href", "/");
  });

  it("shows an error for an unknown episode", async () => {
    renderApp("/episodes/unknown");

    expect(await screen.findByText("Could not load this episode")).toBeInTheDocument();
    expect(screen.getByText("The episode was not found.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to series" })).toHaveAttribute("href", "/");
  });

  it("shows the 404 page for an unknown route", async () => {
    renderApp("/missing");

    expect(await screen.findByText("Page not found")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to series" })).toHaveAttribute("href", "/");
  });
});
