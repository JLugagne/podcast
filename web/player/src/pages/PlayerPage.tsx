import { useEffect, useMemo, useState } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router";
import type { Episode } from "../api/types";
import { ChapterBar } from "../components/ChapterBar";
import { ChapterList } from "../components/ChapterList";
import { CoverTile } from "../components/CoverTile";
import { ErrorBanner } from "../components/ErrorBanner";
import { Icon } from "../components/Icon";
import { IconButton } from "../components/IconButton";
import { SleepTimerMenu } from "../components/SleepTimerMenu";
import { SpeedMenu } from "../components/SpeedMenu";
import { useCatalog } from "../hooks/catalog";
import { progressByEpisode, useProgressList } from "../hooks/listening";
import { formatDuration } from "../lib/format";
import { isEditableTarget, isInteractiveTarget } from "../lib/keyboard";
import { usePlayback } from "../playback/PlaybackProvider";

interface PlayerArtworkProps {
  imageUrl: string | null;
  seriesTitle: string;
  accentColor: string;
  coverUrl: string | null;
}

function PlayerArtwork({ imageUrl, seriesTitle, accentColor, coverUrl }: PlayerArtworkProps) {
  const [failedUrl, setFailedUrl] = useState<string | null>(null);
  const showImage = Boolean(imageUrl) && failedUrl !== imageUrl;

  return (
    <div className="player-artwork">
      {showImage ? (
        <img src={imageUrl ?? undefined} alt="" onError={() => setFailedUrl(imageUrl ?? null)} />
      ) : (
        <CoverTile title={seriesTitle} accentColor={accentColor} src={coverUrl} size="xl" />
      )}
    </div>
  );
}

export function PlayerPage() {
  const playback = usePlayback();
  const {
    episode,
    series,
    chapters,
    chapterIndex,
    currentChapter,
    position,
    duration,
    playing,
    autoPlayNext,
    error,
  } = playback;
  const { toggle, skip, seekTo, previousChapter, nextChapter, restart, setAutoPlayNext, start } =
    playback;
  const navigate = useNavigate();
  const location = useLocation();
  const catalogQuery = useCatalog();
  const progressQuery = useProgressList();
  const progress = progressByEpisode(progressQuery.data);

  useEffect(() => {
    if (!episode || !series) return;
    document.title = playing
      ? `▶ ${episode.title} — ${series.title}`
      : `${episode.title} — ${series.title}`;
    return () => {
      document.title = "Podcasts";
    };
  }, [episode, series, playing]);

  useEffect(() => {
    if (!episode) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (isEditableTarget(event.target)) return;
      if (event.key === " ") {
        if (isInteractiveTarget(event.target)) return;
        event.preventDefault();
        toggle();
        return;
      }
      if (event.key === "ArrowLeft") {
        event.preventDefault();
        if (event.shiftKey) previousChapter();
        else skip(-15);
        return;
      }
      if (event.key === "ArrowRight") {
        event.preventDefault();
        if (event.shiftKey) nextChapter();
        else skip(30);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [episode, toggle, skip, previousChapter, nextChapter]);

  const seriesData = useMemo(() => {
    if (!series) return null;
    return catalogQuery.data?.series.find((item) => item.slug === series.slug) ?? null;
  }, [catalogQuery.data, series]);

  const nextEpisode = useMemo<Episode | null>(() => {
    if (!episode || !seriesData) return null;
    const index = seriesData.episodes.findIndex((candidate) => candidate.id === episode.id);
    return index >= 0 ? (seriesData.episodes[index + 1] ?? null) : null;
  }, [episode, seriesData]);

  if (!episode || !series) return <Navigate to="/" replace />;

  const closePlayer = () => {
    if (location.key === "default") navigate("/");
    else navigate(-1);
  };

  const playNext = () => {
    if (!nextEpisode) return;
    const row = progress.get(nextEpisode.id);
    start(nextEpisode, series, row && !row.completed ? row.position_seconds : 0);
  };

  const remaining = duration > 0 ? Math.max(0, duration - position) : 0;

  return (
    <div className="player-page">
      <header className="player-top">
        <button type="button" className="icon-button icon-button-surface" onClick={closePlayer}>
          <Icon name="close" size={22} />
          <span className="sr-only">Close player</span>
        </button>
        <span className="player-top-label">Now playing</span>
        <span className="player-top-spacer" aria-hidden="true" />
      </header>

      {error ? <ErrorBanner title="Playback problem" message={error} /> : null}

      <div className="player-layout">
        <section className="player-main" aria-label="Player">
          <PlayerArtwork
            key={currentChapter?.image_url ?? "cover"}
            imageUrl={currentChapter?.image_url ?? null}
            seriesTitle={series.title}
            accentColor={series.accent_color}
            coverUrl={series.cover_url}
          />

          <div className="player-heading">
            <Link to={`/series/${series.slug}`} className="player-series-link">
              {series.title}
            </Link>
            <h1 className="player-episode-title">{episode.title}</h1>
            <p className="player-chapter-count">
              {chapters.length > 0 && chapterIndex >= 0
                ? `Chapter ${chapterIndex + 1} of ${chapters.length}`
                : "No chapters"}
            </p>
            <h2 className="player-chapter-title">{currentChapter?.title ?? episode.title}</h2>
            {currentChapter?.url ? (
              <a
                className="chapter-link-button"
                href={currentChapter.url}
                target="_blank"
                rel="noreferrer"
              >
                <Icon name="external" size={16} />
                <span>Open chapter link</span>
              </a>
            ) : null}
          </div>

          <ChapterBar chapters={chapters} duration={duration} position={position} onSeek={seekTo} />

          <div className="player-times">
            <span>{formatDuration(position)}</span>
            <span>{duration > 0 ? `-${formatDuration(remaining)}` : "—"}</span>
          </div>

          <div className="player-controls">
            <IconButton
              label="Previous chapter"
              icon="previous"
              size="md"
              onClick={previousChapter}
            />
            <IconButton label="Back 15 seconds" icon="rewind" size="md" onClick={() => skip(-15)} />
            <IconButton
              label={playing ? "Pause" : "Play"}
              icon={playing ? "pause" : "play"}
              variant="primary"
              size="lg"
              className="play-button"
              onClick={toggle}
            />
            <IconButton
              label="Forward 30 seconds"
              icon="forward"
              size="md"
              onClick={() => skip(30)}
            />
            <IconButton label="Next chapter" icon="next" size="md" onClick={nextChapter} />
          </div>

          <div className="player-options">
            <SpeedMenu />
            <SleepTimerMenu />
          </div>
        </section>

        <aside className="player-side" aria-label="Chapters and queue">
          <section className="player-side-section" aria-labelledby="chapters-heading">
            <div className="section-heading">
              <h2 id="chapters-heading" className="section-title">
                Chapters
              </h2>
              <button type="button" className="btn btn-secondary btn-small" onClick={restart}>
                Restart from the beginning
              </button>
            </div>
            <ChapterList
              chapters={chapters}
              duration={duration}
              position={position}
              onSelect={seekTo}
            />
          </section>

          <section className="player-side-section" aria-labelledby="up-next-heading">
            <div className="section-heading">
              <h2 id="up-next-heading" className="section-title">
                Up next
              </h2>
            </div>
            <label className="toggle-row">
              <input
                type="checkbox"
                checked={autoPlayNext}
                onChange={(event) => setAutoPlayNext(event.target.checked)}
              />
              <span>Autoplay next episode</span>
            </label>
            {nextEpisode ? (
              <button
                type="button"
                className="up-next-card"
                aria-label={`Play next: ${nextEpisode.title}`}
                onClick={playNext}
              >
                <span className="up-next-kicker">Episode {nextEpisode.number}</span>
                <span className="up-next-title">{nextEpisode.title}</span>
                <span className="up-next-meta">{formatDuration(nextEpisode.duration_seconds)}</span>
              </button>
            ) : (
              <p className="muted text-small">This is the latest episode of the series.</p>
            )}
          </section>
        </aside>
      </div>
    </div>
  );
}
