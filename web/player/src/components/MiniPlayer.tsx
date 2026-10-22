import { Link, useLocation } from "react-router";
import { usePlayback } from "../playback/PlaybackProvider";
import { ChapterBar } from "./ChapterBar";
import { CoverTile } from "./CoverTile";
import { IconButton } from "./IconButton";

/** Playback bar docked at the bottom of every screen except the full player. */
export function MiniPlayer() {
  const {
    episode,
    series,
    chapters,
    chapterIndex,
    currentChapter,
    position,
    duration,
    playing,
    toggle,
    skip,
  } = usePlayback();
  const location = useLocation();

  if (!episode || !series || location.pathname === "/play") return null;

  const chapterLine = currentChapter
    ? `Chapter ${chapterIndex + 1}: ${currentChapter.title}`
    : series.title;

  return (
    <section className="mini-player" aria-label="Mini player">
      <ChapterBar
        chapters={chapters}
        duration={duration}
        position={position}
        compact
        className="mini-player-progress"
      />
      <div className="mini-player-inner">
        <Link className="mini-player-link" to="/play" aria-label={`Open player: ${episode.title}`}>
          <CoverTile
            title={series.title}
            accentColor={series.accent_color}
            src={series.cover_url}
            size="sm"
          />
          <span className="mini-player-text">
            <span className="mini-player-title">{episode.title}</span>
            <span className="mini-player-chapter">{chapterLine}</span>
          </span>
        </Link>
        <div className="mini-player-actions">
          <IconButton
            label={playing ? "Pause" : "Play"}
            icon={playing ? "pause" : "play"}
            variant="primary"
            size="md"
            onClick={toggle}
          />
          <IconButton
            label="Forward 30 seconds"
            icon="forward"
            size="md"
            onClick={() => skip(30)}
          />
        </div>
      </div>
    </section>
  );
}
