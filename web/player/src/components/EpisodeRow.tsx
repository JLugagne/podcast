import type { Episode, Progress } from "../api/types";
import { formatDuration, formatMinutesLeft, pluralize } from "../lib/format";
import { Icon } from "./Icon";
import { IconButton } from "./IconButton";
import { Menu } from "./Menu";

export interface EpisodeRowProps {
  episode: Episode;
  progress?: Progress;
  isCurrent: boolean;
  playing: boolean;
  onTogglePlay: () => void;
  onOpenPlayer: () => void;
  onMarkPlayed: () => void;
  onMarkUnplayed: () => void;
  onRestart: () => void;
  downloaded: boolean;
  /** Download progress 0..1 (or -1 when unknown) while a download runs. */
  downloading?: number;
  downloadError?: string;
  onDownload: () => void;
  onRemoveDownload: () => void;
}

function stateLabel(progress: Progress | undefined, duration: number): string {
  if (progress?.completed) return "Played";
  if (progress && progress.position_seconds > 0) {
    return formatMinutesLeft(Math.max(0, duration - progress.position_seconds));
  }
  return "New";
}

/** One row of the episode list of a series. */
export function EpisodeRow({
  episode,
  progress,
  isCurrent,
  playing,
  onTogglePlay,
  onOpenPlayer,
  onMarkPlayed,
  onMarkUnplayed,
  onRestart,
  downloaded,
  downloading,
  downloadError,
  onDownload,
  onRemoveDownload,
}: EpisodeRowProps) {
  const duration = episode.duration_seconds;
  const positionSeconds = progress?.position_seconds ?? 0;
  const percent = duration > 0 ? Math.min(100, (positionSeconds / duration) * 100) : 0;
  const label = `${playing ? "Pause" : "Play"} episode ${episode.number}: ${episode.title}`;

  return (
    <li className={["episode-row", isCurrent ? "is-current" : ""].filter(Boolean).join(" ")}>
      <span className="episode-row-number" aria-hidden="true">
        {episode.number}
      </span>
      <IconButton
        label={label}
        icon={playing ? "pause" : "play"}
        variant={isCurrent ? "primary" : "surface"}
        size="md"
        onClick={onTogglePlay}
      />
      <div className="episode-row-body">
        <h3 className="episode-row-title">
          <button type="button" className="episode-row-title-button" onClick={onOpenPlayer}>
            {episode.title}
          </button>
        </h3>
        <p className="episode-row-meta">
          <span>{formatDuration(duration)}</span>
          <span aria-hidden="true">·</span>
          {episode.chapter_count > 0 ? (
            <>
              <span>{pluralize(episode.chapter_count, "chapter")}</span>
              <span aria-hidden="true">·</span>
            </>
          ) : null}
          <span>{stateLabel(progress, duration)}</span>
          {downloaded ? (
            <>
              <span aria-hidden="true">·</span>
              <span>Downloaded</span>
            </>
          ) : null}
          {downloading !== undefined ? (
            <>
              <span aria-hidden="true">·</span>
              <span>
                {downloading >= 0 && downloading < 1
                  ? `Downloading ${Math.round(downloading * 100)}%`
                  : "Downloading…"}
              </span>
            </>
          ) : null}
        </p>
        {downloadError ? (
          <p className="episode-row-error" role="alert">
            {downloadError}
          </p>
        ) : null}
        <div className="progress-thin" aria-hidden="true">
          <span className="progress-thin-fill" style={{ width: `${percent}%` }} />
        </div>
      </div>
      <Menu
        label={`More actions for episode ${episode.number}`}
        trigger={<Icon name="more" size={20} />}
      >
        {(close) => (
          <>
            <button
              type="button"
              role="menuitem"
              className="menu-item"
              onClick={() => {
                onMarkPlayed();
                close();
              }}
            >
              Mark as played
            </button>
            <button
              type="button"
              role="menuitem"
              className="menu-item"
              onClick={() => {
                onMarkUnplayed();
                close();
              }}
            >
              Mark as unplayed
            </button>
            <button
              type="button"
              role="menuitem"
              className="menu-item"
              onClick={() => {
                onRestart();
                close();
              }}
            >
              Restart from the beginning
            </button>
            <button
              type="button"
              role="menuitem"
              className="menu-item"
              disabled={downloading !== undefined}
              onClick={() => {
                if (downloaded) onRemoveDownload();
                else onDownload();
                close();
              }}
            >
              {downloaded ? "Remove download" : "Download for offline"}
            </button>
          </>
        )}
      </Menu>
    </li>
  );
}
