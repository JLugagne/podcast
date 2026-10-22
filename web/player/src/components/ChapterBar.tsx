import type { KeyboardEvent } from "react";
import type { ChapterDTO } from "../api/types";
import { chapterIndexAt, segmentFills, segmentWidths } from "../lib/chapters";
import { formatDuration } from "../lib/format";

export interface ChapterBarProps {
  chapters: ChapterDTO[];
  duration: number;
  position: number;
  /** When provided the bar behaves as an accessible slider. */
  onSeek?: (seconds: number) => void;
  label?: string;
  compact?: boolean;
  className?: string;
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(Math.max(value, minimum), maximum);
}

/**
 * Chapter-segmented progress bar. The interactive form is a real slider:
 * clicking anywhere seeks, arrow keys move 5 seconds, Home/End jump to the
 * edges. The static form is used inside the mini player.
 */
export function ChapterBar({
  chapters,
  duration,
  position,
  onSeek,
  label = "Playback position",
  compact = false,
  className,
}: ChapterBarProps) {
  const interactive = typeof onSeek === "function";
  const widths = segmentWidths(chapters, duration);
  const fills = segmentFills(chapters, duration, position);
  const index = chapterIndexAt(chapters, position);
  const chapterText = index >= 0 ? `chapter ${index + 1}: ${chapters[index].title}` : "no chapter";
  const valueText = `${formatDuration(position)} of ${formatDuration(duration)}, ${chapterText}`;

  const seekFromPoint = (clientX: number, element: HTMLElement) => {
    if (!onSeek || !(duration > 0)) return;
    const rect = element.getBoundingClientRect();
    if (!(rect.width > 0)) return;
    const ratio = clamp((clientX - rect.left) / rect.width, 0, 1);
    onSeek(ratio * duration);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!onSeek) return;
    const step = event.key === "ArrowRight" ? 5 : event.key === "ArrowLeft" ? -5 : null;
    if (step !== null) {
      event.preventDefault();
      event.stopPropagation();
      onSeek(clamp(position + step, 0, duration));
      return;
    }
    if (event.key === "Home") {
      event.preventDefault();
      event.stopPropagation();
      onSeek(0);
    } else if (event.key === "End") {
      event.preventDefault();
      event.stopPropagation();
      onSeek(duration);
    }
  };

  const classes = [
    "chapter-bar",
    compact ? "chapter-bar-compact" : "",
    interactive ? "chapter-bar-interactive" : "chapter-bar-static",
    className ?? "",
  ]
    .filter(Boolean)
    .join(" ");

  const columns = widths.length > 0 ? widths.map((width) => `${width}fr`).join(" ") : "1fr";

  if (!interactive) {
    return (
      <div className={classes} aria-hidden="true">
        <div className="chapter-bar-track" style={{ gridTemplateColumns: columns }}>
          {chapters.map((chapter, segmentIndex) => (
            <span className="chapter-bar-segment" key={`${chapter.start_seconds}-${chapter.title}`}>
              <span
                className="chapter-bar-fill"
                style={{ width: `${fills[segmentIndex] ?? 0}%` }}
              />
            </span>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div
      className={classes}
      role="slider"
      tabIndex={0}
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={Math.round(duration)}
      aria-valuenow={Math.round(position)}
      aria-valuetext={valueText}
      onClick={(event) => seekFromPoint(event.clientX, event.currentTarget)}
      onKeyDown={handleKeyDown}
    >
      <div className="chapter-bar-track" style={{ gridTemplateColumns: columns }}>
        {chapters.map((chapter, segmentIndex) => (
          <span className="chapter-bar-segment" key={`${chapter.start_seconds}-${chapter.title}`}>
            <span className="chapter-bar-fill" style={{ width: `${fills[segmentIndex] ?? 0}%` }} />
          </span>
        ))}
      </div>
    </div>
  );
}
