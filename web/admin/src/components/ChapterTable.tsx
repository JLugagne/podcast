import type { ChapterDTO } from "../api/types";
import { formatTimecode } from "../lib/format";

export interface ChapterTableProps {
  chapters: ChapterDTO[];
  /** When provided, rows and the start-time button seek the audio player. */
  onSeek?: (seconds: number) => void;
  caption?: string;
}

function linkLabel(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, "");
  } catch {
    return "Open link";
  }
}

export function ChapterTable({ chapters, onSeek, caption }: ChapterTableProps) {
  return (
    <div className="table-wrap">
      <table className="data-table">
        <caption className="sr-only">{caption ?? "Chapter list"}</caption>
        <thead>
          <tr>
            <th scope="col">Start</th>
            <th scope="col">Title</th>
            <th scope="col">End</th>
            <th scope="col">Link</th>
          </tr>
        </thead>
        <tbody>
          {chapters.map((chapter, index) => (
            <tr
              // biome-ignore lint/suspicious/noArrayIndexKey: start times may repeat while the user edits an invalid document, so the index keeps keys unique
              key={`${chapter.start_seconds}-${index}`}
              className={onSeek ? "chapter-row" : undefined}
              onClick={onSeek ? () => onSeek(chapter.start_seconds) : undefined}
            >
              <td className="cell-number">
                {onSeek ? (
                  <button
                    type="button"
                    className="chapter-seek"
                    aria-label={`Play chapter ${index + 1} at ${formatTimecode(chapter.start_seconds)}`}
                    onClick={(event) => {
                      event.stopPropagation();
                      onSeek(chapter.start_seconds);
                    }}
                  >
                    {formatTimecode(chapter.start_seconds)}
                  </button>
                ) : (
                  formatTimecode(chapter.start_seconds)
                )}
              </td>
              <td>{chapter.title}</td>
              <td className="chapter-end">
                {chapter.end_seconds === undefined || chapter.end_seconds === null
                  ? "—"
                  : formatTimecode(chapter.end_seconds)}
              </td>
              <td>
                {chapter.url ? (
                  <a
                    className="chapter-link"
                    href={chapter.url}
                    target="_blank"
                    rel="noreferrer"
                    onClick={(event) => event.stopPropagation()}
                  >
                    {linkLabel(chapter.url)}
                  </a>
                ) : (
                  "—"
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
