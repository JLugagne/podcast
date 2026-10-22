import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import { AppHeader } from "../../App";

describe("app header", () => {
  it("links to the listener application", () => {
    render(
      <MemoryRouter>
        <AppHeader />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Open listener app" })).toHaveAttribute("href", "/");
  });
});
