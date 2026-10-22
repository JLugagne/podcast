import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ChapterDTO } from "../api/types";
import { ChapterTable } from "./ChapterTable";

const chapters: ChapterDTO[] = [
  { start_seconds: 0, title: "Intro", toc: true },
  {
    start_seconds: 95.5,
    end_seconds: 120,
    title: "Main topic",
    url: "https://www.example.com/notes",
    toc: true,
  },
];

describe("ChapterTable", () => {
  it("renders chapters with timecodes, end times and link labels", () => {
    render(<ChapterTable chapters={chapters} />);

    expect(screen.getByText("Intro")).toBeInTheDocument();
    expect(screen.getByText("0:01:35")).toBeInTheDocument();
    expect(screen.getByText("0:02:00")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "example.com" })).toHaveAttribute(
      "href",
      "https://www.example.com/notes",
    );
  });

  it("calls onSeek once when a chapter start time is activated", async () => {
    const onSeek = vi.fn();
    const user = userEvent.setup();
    render(<ChapterTable chapters={chapters} onSeek={onSeek} />);

    await user.click(screen.getByRole("button", { name: /play chapter 2 at 0:01:35/i }));

    expect(onSeek).toHaveBeenCalledTimes(1);
    expect(onSeek).toHaveBeenCalledWith(95.5);
  });

  it("seeks when a row is clicked", async () => {
    const onSeek = vi.fn();
    const user = userEvent.setup();
    render(<ChapterTable chapters={chapters} onSeek={onSeek} />);

    await user.click(screen.getByText("Intro"));

    expect(onSeek).toHaveBeenCalledWith(0);
  });

  it("renders start times as plain text without a seek handler", () => {
    render(<ChapterTable chapters={chapters} />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
