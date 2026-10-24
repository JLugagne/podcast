import { render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ChapterDTO } from "../api/types";
import { ChapterList } from "./ChapterList";

const chapters: ChapterDTO[] = [0, 60, 120, 180].map((start, index) => ({
  start_seconds: start,
  title: `Chapter title ${index + 1}`,
  toc: true,
}));

function renderList(position: number) {
  return <ChapterList chapters={chapters} duration={240} position={position} onSelect={() => {}} />;
}

describe("ChapterList", () => {
  afterEach(() => {
    Reflect.deleteProperty(Element.prototype, "scrollIntoView");
  });

  it("scrolls the active chapter into view when the position jumps to another chapter", () => {
    const scrolled: string[] = [];
    Element.prototype.scrollIntoView = vi.fn(function (this: Element) {
      scrolled.push(this.getAttribute("aria-label") ?? "");
    });

    const { rerender } = render(renderList(5));
    expect(scrolled.at(-1)).toBe("Chapter 1: Chapter title 1");

    rerender(renderList(130));
    expect(scrolled.at(-1)).toBe("Chapter 3: Chapter title 3");

    const calls = scrolled.length;
    rerender(renderList(140));
    expect(scrolled).toHaveLength(calls);
  });
});
