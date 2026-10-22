import { describe, expect, it } from "vitest";
import type { ChapterDTO } from "../api/types";
import {
  chapterEnd,
  chapterIndexAt,
  nextChapterTarget,
  previousChapterTarget,
  RESTART_THRESHOLD_SECONDS,
  segmentFills,
  segmentWidths,
  sortChapters,
} from "./chapters";

const chapters: ChapterDTO[] = [
  { start_seconds: 0, title: "Intro", toc: true },
  { start_seconds: 30, title: "Topic", toc: true },
  { start_seconds: 80, title: "Outro", toc: true },
];

describe("chapterIndexAt", () => {
  it("returns the chapter containing the time", () => {
    expect(chapterIndexAt(chapters, 0)).toBe(0);
    expect(chapterIndexAt(chapters, 29.9)).toBe(0);
    expect(chapterIndexAt(chapters, 30)).toBe(1);
    expect(chapterIndexAt(chapters, 79.9)).toBe(1);
    expect(chapterIndexAt(chapters, 80)).toBe(2);
    expect(chapterIndexAt(chapters, 500)).toBe(2);
  });

  it("returns -1 before the first chapter", () => {
    expect(chapterIndexAt(chapters, -1)).toBe(-1);
    expect(chapterIndexAt([], 10)).toBe(-1);
  });

  it("works on unsorted input through the sorted view", () => {
    const unordered: ChapterDTO[] = [chapters[2], chapters[0], chapters[1]];
    expect(chapterIndexAt(sortChapters(unordered), 35)).toBe(1);
  });
});

describe("chapterEnd", () => {
  it("uses the next chapter start, then the episode duration", () => {
    expect(chapterEnd(chapters, 0, 100)).toBe(30);
    expect(chapterEnd(chapters, 1, 100)).toBe(80);
    expect(chapterEnd(chapters, 2, 100)).toBe(100);
  });

  it("prefers an explicit end time", () => {
    const explicit: ChapterDTO[] = [
      { start_seconds: 0, title: "Intro", toc: true, end_seconds: 12 },
      { start_seconds: 12, title: "Body", toc: true, end_seconds: 40 },
    ];
    expect(chapterEnd(explicit, 0, 100)).toBe(12);
    expect(chapterEnd(explicit, 1, 100)).toBe(40);
  });
});

describe("segmentWidths", () => {
  it("spans the whole episode", () => {
    const widths = segmentWidths(chapters, 100);
    expect(widths).toEqual([30, 50, 20]);
    expect(widths.reduce((sum, width) => sum + width, 0)).toBeCloseTo(100, 6);
  });

  it("gives the last chapter the remaining time", () => {
    const widths = segmentWidths(chapters, 120);
    expect(widths).toEqual([25, 100 * (50 / 120), 100 * (40 / 120)]);
    expect(widths.reduce((sum, width) => sum + width, 0)).toBeCloseTo(100, 6);
  });

  it("handles an unknown duration and an empty chapter list", () => {
    expect(segmentWidths([], 100)).toEqual([]);
    const widths = segmentWidths(chapters, 0);
    expect(widths.reduce((sum, width) => sum + width, 0)).toBeCloseTo(100, 6);
  });

  it("keeps rounding from losing a sliver of the bar", () => {
    const uneven: ChapterDTO[] = [
      { start_seconds: 0, title: "A", toc: true },
      { start_seconds: 1, title: "B", toc: true },
      { start_seconds: 2, title: "C", toc: true },
    ];
    const widths = segmentWidths(uneven, 7);
    expect(widths.reduce((sum, width) => sum + width, 0)).toBeCloseTo(100, 6);
  });
});

describe("segmentFills", () => {
  it("fills each segment according to the position", () => {
    expect(segmentFills(chapters, 100, 0)).toEqual([0, 0, 0]);
    expect(segmentFills(chapters, 100, 15)).toEqual([50, 0, 0]);
    expect(segmentFills(chapters, 100, 40)).toEqual([100, 20, 0]);
    expect(segmentFills(chapters, 100, 100)).toEqual([100, 100, 100]);
    expect(segmentFills(chapters, 100, 400)).toEqual([100, 100, 100]);
  });

  it("returns one entry per chapter", () => {
    expect(segmentFills([], 100, 10)).toEqual([]);
  });
});

describe("previousChapterTarget", () => {
  it("restarts the current chapter when past the threshold", () => {
    expect(previousChapterTarget(chapters, RESTART_THRESHOLD_SECONDS + 0.5)).toEqual({
      index: 0,
      seconds: 0,
    });
    expect(previousChapterTarget(chapters, 40)).toEqual({ index: 1, seconds: 30 });
  });

  it("steps back to the previous chapter near the start", () => {
    expect(previousChapterTarget(chapters, 1)).toEqual({ index: 0, seconds: 0 });
    expect(previousChapterTarget(chapters, 31)).toEqual({ index: 0, seconds: 0 });
    expect(previousChapterTarget(chapters, 81)).toEqual({ index: 1, seconds: 30 });
  });

  it("stays on the first chapter", () => {
    expect(previousChapterTarget(chapters, 0)).toEqual({ index: 0, seconds: 0 });
  });

  it("falls back to the beginning without chapters", () => {
    expect(previousChapterTarget([], 20)).toEqual({ index: -1, seconds: 0 });
  });
});

describe("nextChapterTarget", () => {
  it("moves to the next chapter start", () => {
    expect(nextChapterTarget(chapters, 0)).toEqual({ index: 1, seconds: 30 });
    expect(nextChapterTarget(chapters, 35)).toEqual({ index: 2, seconds: 80 });
  });

  it("returns null on the last chapter", () => {
    expect(nextChapterTarget(chapters, 80)).toBeNull();
    expect(nextChapterTarget([], 0)).toBeNull();
  });
});
