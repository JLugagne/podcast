import type { ChapterDTO } from "../api/types";

/** How far into a chapter the listener must be for "previous" to restart it. */
export const RESTART_THRESHOLD_SECONDS = 3;

/** Returns the chapters ordered by start time (the API already sorts them). */
export function sortChapters(chapters: ChapterDTO[]): ChapterDTO[] {
  return [...chapters].sort((left, right) => left.start_seconds - right.start_seconds);
}

/**
 * Index of the chapter that contains `time` (the last chapter starting at or
 * before it), or -1 when the time precedes the first chapter.
 */
export function chapterIndexAt(chapters: ChapterDTO[], time: number): number {
  let index = -1;
  for (let position = 0; position < chapters.length; position += 1) {
    if (time >= chapters[position].start_seconds) index = position;
    else break;
  }
  return index;
}

/**
 * End of a chapter: its explicit end, otherwise the start of the next chapter,
 * otherwise the end of the episode.
 */
export function chapterEnd(chapters: ChapterDTO[], index: number, durationSeconds: number): number {
  const chapter = chapters[index];
  if (!chapter) return 0;
  const next = chapters[index + 1];
  const end = chapter.end_seconds ?? next?.start_seconds ?? durationSeconds;
  return Math.max(end, chapter.start_seconds);
}

interface SegmentBounds {
  start: number;
  end: number;
}

function segmentBounds(chapters: ChapterDTO[], durationSeconds: number): SegmentBounds[] {
  if (chapters.length === 0) return [];
  const lastStart = chapters[chapters.length - 1].start_seconds;
  const total = durationSeconds > 0 ? durationSeconds : Math.max(lastStart + 1, 1);
  return chapters.map((chapter, index) => {
    const start = Math.min(Math.max(chapter.start_seconds, 0), total);
    const end =
      index === chapters.length - 1
        ? total
        : Math.min(Math.max(chapters[index + 1].start_seconds, start), total);
    return { start, end };
  });
}

/**
 * Width percentages of the chapter segments on a progress bar. The widths
 * always sum to 100 so the bar spans the whole episode.
 */
export function segmentWidths(chapters: ChapterDTO[], durationSeconds: number): number[] {
  const bounds = segmentBounds(chapters, durationSeconds);
  if (bounds.length === 0) return [];
  const total = bounds[bounds.length - 1].end - bounds[0].start;
  if (!(total > 0)) return bounds.map(() => 100 / bounds.length);
  const widths = bounds.map(({ start, end }) => ((end - start) / total) * 100);
  const sum = widths.reduce((acc, width) => acc + width, 0);
  const last = widths.length - 1;
  widths[last] = Math.max(0, widths[last] + (100 - sum));
  return widths;
}

/**
 * Played fraction of each chapter segment, as a percentage of that segment's
 * own width (0 = untouched, 100 = fully played).
 */
export function segmentFills(
  chapters: ChapterDTO[],
  durationSeconds: number,
  positionSeconds: number,
): number[] {
  const bounds = segmentBounds(chapters, durationSeconds);
  return bounds.map(({ start, end }) => {
    if (positionSeconds >= end) return 100;
    if (positionSeconds <= start) return 0;
    if (!(end > start)) return 0;
    return ((positionSeconds - start) / (end - start)) * 100;
  });
}

export interface ChapterSeekTarget {
  index: number;
  seconds: number;
}

/**
 * Target of the "previous chapter" control: restart the current chapter when
 * the listener is more than three seconds into it, otherwise step back to the
 * previous chapter (or to the start of the episode).
 */
export function previousChapterTarget(
  chapters: ChapterDTO[],
  positionSeconds: number,
): ChapterSeekTarget | null {
  if (chapters.length === 0) return { index: -1, seconds: 0 };
  const index = Math.max(0, chapterIndexAt(chapters, positionSeconds));
  const chapterStart = chapters[index].start_seconds;
  if (positionSeconds - chapterStart > RESTART_THRESHOLD_SECONDS) {
    return { index, seconds: chapterStart };
  }
  if (index > 0) return { index: index - 1, seconds: chapters[index - 1].start_seconds };
  return { index: 0, seconds: chapters[0].start_seconds };
}

/** Target of the "next chapter" control, or null on the last chapter. */
export function nextChapterTarget(
  chapters: ChapterDTO[],
  positionSeconds: number,
): ChapterSeekTarget | null {
  if (chapters.length === 0) return null;
  const next = chapterIndexAt(chapters, positionSeconds) + 1;
  if (next >= chapters.length) return null;
  return { index: next, seconds: chapters[next].start_seconds };
}
