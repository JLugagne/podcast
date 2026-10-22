import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CoverTile } from "./CoverTile";

describe("CoverTile", () => {
  it.each([
    ["sm", "cover-tile-sm"],
    ["md", "cover-tile-md"],
    ["lg", "cover-tile-lg"],
    ["xl", "cover-tile-xl"],
  ] as const)("applies the %s size class", (size, className) => {
    const { container } = render(
      <CoverTile title="Harbour Lights" accentColor="#F2A541" size={size} />,
    );
    expect(container.firstElementChild).toHaveClass("cover-tile", className);
  });

  it("uses the md size by default", () => {
    const { container } = render(<CoverTile title="Harbour Lights" accentColor="#F2A541" />);
    expect(container.firstElementChild).toHaveClass("cover-tile-md");
  });
});
