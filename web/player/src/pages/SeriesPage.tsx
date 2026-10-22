import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { errorMessage, isNotFound } from "../api/client";
import type { Episode, Progress, Series } from "../api/types";
import { CoverTile } from "../components/CoverTile";
import { EpisodeRow } from "../components/EpisodeRow";
import { ErrorBanner } from "../components/ErrorBanner";
import { Icon } from "../components/Icon";
import { LoadingBlock } from "../components/LoadingBlock";
import { usePublishedEpisodes, useSeries } from "../hooks/catalog";
import {
  progressByEpisode,
  useMarkPlayed,
  useMarkUnplayed,
  useProgressList,
} from "../hooks/listening";
import { formatDuration, pluralize } from "../lib/format";
import { usePlayback } from "../playback/PlaybackProvider";

function absoluteFeedUrl(feedUrl: string): string {
  try {
    return new URL(feedUrl, window.location.origin).href;
  } catch {
    return feedUrl;
  }
}

export function SeriesPage() {
  const { id } = useParams<{ id: string }>();
  const seriesQuery = useSeries(id);
  const episodesQuery = usePublishedEpisodes(id);
  const progressQuery = useProgressList();
  const playback = usePlayback();
  const navigate = useNavigate();
  const markPlayed = useMarkPlayed();
  const markUnplayed = useMarkUnplayed();
  const [copyState, setCopyState] = useState<"idle" | "copied" | "failed">("idle");

  useEffect(() => {
    if (copyState !== "copied") return;
    const timer = window.setTimeout(() => setCopyState("idle"), 2500);
    return () => window.clearTimeout(timer);
  }, [copyState]);

  if (seriesQuery.isPending) {
    return (
      <div className="page">
        <LoadingBlock label="Loading series…" />
      </div>
    );
  }

  if (seriesQuery.isError) {
    const notFound = isNotFound(seriesQuery.error);
    return (
      <div className="page">
        {notFound ? (
          <div className="empty-state">
            <h2>Series not found</h2>
            <p>This series does not exist or has been removed.</p>
            <Link to="/" className="btn btn-primary">
              Back to the library
            </Link>
          </div>
        ) : (
          <>
            <ErrorBanner
              title="Could not load this series"
              message={errorMessage(seriesQuery.error)}
            />
            <button
              type="button"
              className="btn btn-secondary mt-3"
              onClick={() => void seriesQuery.refetch()}
            >
              Try again
            </button>
          </>
        )}
      </div>
    );
  }

  const series: Series = seriesQuery.data;
  const progress = progressByEpisode(progressQuery.data);
  const episodes = [...(episodesQuery.data?.episodes ?? [])].sort(
    (left, right) => left.number - right.number,
  );
  const latest = episodes.length > 0 ? episodes[episodes.length - 1] : null;
  const inProgress = episodes
    .filter((episode) => {
      const row = progress.get(episode.id);
      return Boolean(row && !row.completed && row.position_seconds > 0);
    })
    .sort((left, right) =>
      (progress.get(right.id)?.updated_at ?? "").localeCompare(
        progress.get(left.id)?.updated_at ?? "",
      ),
    );
  const resumeEpisode = inProgress[0] ?? latest;
  const primaryLabel = inProgress.length > 0 ? "Resume" : "Play latest";
  const totalDuration = episodes.reduce(
    (total, episode) => total + (episode.audio?.duration_seconds ?? 0),
    0,
  );

  const savedPosition = (episode: Episode): number => {
    const row: Progress | undefined = progress.get(episode.id);
    if (!row || row.completed) return 0;
    return row.position_seconds;
  };

  const startPlayback = (episode: Episode, positionSeconds: number) => {
    playback.start(episode, series, positionSeconds);
  };

  const openPlayer = (episode: Episode, positionSeconds: number) => {
    startPlayback(episode, positionSeconds);
    navigate("/play");
  };

  const handleToggle = (episode: Episode) => {
    if (playback.episode?.id === episode.id) {
      playback.toggle();
      return;
    }
    startPlayback(episode, savedPosition(episode));
  };

  const copyFeed = async () => {
    try {
      await navigator.clipboard.writeText(absoluteFeedUrl(series.feed_url));
      setCopyState("copied");
    } catch {
      setCopyState("failed");
    }
  };

  const primaryPosition = resumeEpisode ? savedPosition(resumeEpisode) : 0;
  const primaryLabelText = resumeEpisode
    ? `${primaryLabel}: Episode ${resumeEpisode.number}: ${resumeEpisode.title}`
    : primaryLabel;

  return (
    <div className="page series-page">
      <Link to="/" className="back-link">
        <Icon name="back" size={16} />
        <span>Library</span>
      </Link>

      <header className="series-header">
        <div className="series-cover">
          <CoverTile
            title={series.title}
            accentColor={series.accent_color}
            src={series.cover_url}
            size="lg"
          />
        </div>
        <div className="series-header-body">
          <h1 className="page-title">{series.title}</h1>
          {series.author ? <p className="series-author">{series.author}</p> : null}
          {series.description ? <p className="series-description">{series.description}</p> : null}
          <p className="series-stats">
            {pluralize(episodes.length, "episode")}
            {totalDuration > 0 ? ` · ${formatDuration(totalDuration)}` : ""}
          </p>
          {resumeEpisode ? (
            <button
              type="button"
              className="btn btn-primary series-primary"
              aria-label={primaryLabelText}
              onClick={() => openPlayer(resumeEpisode, primaryPosition)}
            >
              <Icon name="play" size={18} />
              <span>{primaryLabel}</span>
            </button>
          ) : null}
        </div>
      </header>

      <section className="feed-panel" aria-label="RSS feed">
        <div className="feed-info">
          <span className="feed-label">
            <Icon name="rss" size={16} />
            <span>RSS feed</span>
          </span>
          <code className="feed-url">{absoluteFeedUrl(series.feed_url)}</code>
        </div>
        <div className="feed-actions">
          <button type="button" className="btn btn-secondary" onClick={() => void copyFeed()}>
            <Icon name="copy" size={16} />
            <span>Copy RSS feed</span>
          </button>
          {copyState === "copied" ? (
            <span className="copy-status" role="status">
              Copied
            </span>
          ) : null}
          {copyState === "failed" ? (
            <span className="copy-status copy-status-error" role="status">
              Could not copy
            </span>
          ) : null}
        </div>
      </section>

      <section className="section" aria-labelledby="episodes-heading">
        <div className="section-heading">
          <h2 id="episodes-heading" className="section-title">
            Episodes
          </h2>
        </div>
        {episodesQuery.isPending ? (
          <LoadingBlock label="Loading episodes…" />
        ) : episodesQuery.isError ? (
          <ErrorBanner
            title="Could not load the episodes"
            message={errorMessage(episodesQuery.error)}
          />
        ) : episodes.length === 0 ? (
          <div className="empty-state">
            <h2>No published episodes</h2>
            <p>Episodes show up here once they are published in the admin.</p>
          </div>
        ) : (
          <ul className="episode-list" aria-label={`Episodes of ${series.title}`}>
            {episodes.map((episode) => (
              <EpisodeRow
                key={episode.id}
                episode={episode}
                progress={progress.get(episode.id)}
                isCurrent={playback.episode?.id === episode.id}
                playing={playback.episode?.id === episode.id && playback.playing}
                onTogglePlay={() => handleToggle(episode)}
                onOpenPlayer={() => openPlayer(episode, savedPosition(episode))}
                onMarkPlayed={() => markPlayed.mutate(episode.id)}
                onMarkUnplayed={() => markUnplayed.mutate(episode.id)}
              />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
