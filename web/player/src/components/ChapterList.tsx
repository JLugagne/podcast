import { useEffect, useRef } from "react";
import type { ChapterDTO } from "../api/types";
import { chapterEnd, chapterIndexAt, sortChapters } from "../lib/chapters";
import { formatDuration } from "../lib/format";
import { Icon } from "./Icon";

export interface ChapterListProps {
  chapters: ChapterDTO[];
  duration: number;
  position: number;
  onSelect: (seconds: number) => void;
}

/** Chapters of the current episode; the active chapter is highlighted. */
export function ChapterList({ chapters, duration, position, onSelect }: ChapterListProps) {
  const ordered = sortChapters(chapters);
  const activeIndex = chapterIndexAt(ordered, position);
  const activeRef = useRef<HTMLButtonElement | null>(null);
  const listRef = useRef<HTMLOListElement | null>(null);

  // Keep the active chapter visible, whether it changed by playback, a chapter
  // tap or a click on the timeline.
  // biome-ignore lint/correctness/useExhaustiveDependencies: scroll only when the active chapter changes.
  useEffect(() => {
    const row = activeRef.current;
    if (!row || typeof row.scrollIntoView !== "function") return;
    // Do not drag the page around while the list is off screen.
    const bounds = listRef.current?.getBoundingClientRect();
    if (bounds && (bounds.bottom < 0 || bounds.top > window.innerHeight)) return;
    const reduceMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
    row.scrollIntoView({ block: "nearest", behavior: reduceMotion ? "auto" : "smooth" });
  }, [activeIndex]);

  if (ordered.length === 0) {
    return <p className="muted text-small">This episode has no chapters.</p>;
  }

  return (
    <ol className="chapter-list" aria-label="Chapters" ref={listRef}>
      {ordered.map((chapter, index) => {
        const end = chapterEnd(ordered, index, duration);
        const played = position >= end;
        const active = index === activeIndex;
        return (
          <li key={`${chapter.start_seconds}-${chapter.title}`}>
            <button
              type="button"
              ref={active ? activeRef : undefined}
              className={["chapter-row", active ? "is-active" : "", played ? "is-played" : ""]
                .filter(Boolean)
                .join(" ")}
              aria-label={`Chapter ${index + 1}: ${chapter.title}`}
              aria-current={active ? "true" : undefined}
              onClick={() => onSelect(chapter.start_seconds)}
            >
              <span className="chapter-row-number" aria-hidden="true">
                {index + 1}
              </span>
              <span className="chapter-row-body">
                <span className="chapter-row-title">{chapter.title}</span>
                <span className="chapter-row-meta" aria-hidden="true">
                  {formatDuration(chapter.start_seconds)} ·{" "}
                  {formatDuration(Math.max(0, end - chapter.start_seconds))}
                </span>
              </span>
              {played ? (
                <span className="chapter-row-check" aria-hidden="true">
                  <Icon name="check" size={16} />
                </span>
              ) : null}
            </button>
          </li>
        );
      })}
    </ol>
  );
}
