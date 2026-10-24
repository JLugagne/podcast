import type { Episode, Progress } from "../api/types";
import { useI18n } from "../i18n";
import { formatDuration, formatMinutesLeft } from "../lib/format";
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
  const { t } = useI18n();
  const stateLabel = (row: Progress | undefined, length: number): string => {
    if (row?.completed) return t("played");
    if (row && row.position_seconds > 0) {
      return formatMinutesLeft(Math.max(0, length - row.position_seconds));
    }
    return t("newEpisode");
  };
  const duration = episode.duration_seconds;
  const positionSeconds = progress?.position_seconds ?? 0;
  const percent = duration > 0 ? Math.min(100, (positionSeconds / duration) * 100) : 0;
  const label = playing
    ? t("pauseEpisode", { number: episode.number, title: episode.title })
    : t("playEpisode", { number: episode.number, title: episode.title });

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
              <span>{t("chapters", { count: episode.chapter_count })}</span>
              <span aria-hidden="true">·</span>
            </>
          ) : null}
          <span>{stateLabel(progress, duration)}</span>
          {downloaded ? (
            <>
              <span aria-hidden="true">·</span>
              <span>{t("downloaded")}</span>
            </>
          ) : null}
          {downloading !== undefined ? (
            <>
              <span aria-hidden="true">·</span>
              <span>
                {downloading >= 0 && downloading < 1
                  ? t("downloadingPercent", { percent: Math.round(downloading * 100) })
                  : t("downloading")}
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
        label={t("moreActions", { number: episode.number })}
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
              {t("markPlayed")}
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
              {t("markUnplayed")}
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
              {t("restartFromBeginning")}
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
              {downloaded ? t("removeDownload") : t("downloadForOffline")}
            </button>
          </>
        )}
      </Menu>
    </li>
  );
}
