import { Link, useNavigate } from "react-router";
import { errorMessage } from "../api/client";
import type { ContinueItem, Episode, Series } from "../api/types";
import { ChapterBar } from "../components/ChapterBar";
import { CoverTile } from "../components/CoverTile";
import { ErrorBanner } from "../components/ErrorBanner";
import { Icon } from "../components/Icon";
import { LoadingBlock } from "../components/LoadingBlock";
import { useEpisode, usePublishedEpisodes, useSeriesList } from "../hooks/catalog";
import { progressByEpisode, useContinueListening, useProgressList } from "../hooks/listening";
import { chapterIndexAt, sortChapters } from "../lib/chapters";
import { formatMinutesLeft, pluralize } from "../lib/format";
import { usePlayback } from "../playback/PlaybackProvider";

function ContinueCard({ item }: { item: ContinueItem }) {
  const { progress, episode } = item;
  const episodeQuery = useEpisode(episode.id);
  const playback = usePlayback();
  const navigate = useNavigate();

  const detail = episodeQuery.data;
  const chapters = sortChapters(detail?.chapters ?? []);
  const chapterIndex =
    chapters.length > 0 ? chapterIndexAt(chapters, progress.position_seconds) : -1;
  const duration =
    episode.duration_seconds > 0
      ? episode.duration_seconds
      : (detail?.audio?.duration_seconds ?? 0);
  const chapterLine =
    chapterIndex >= 0
      ? `Chapter ${chapterIndex + 1}: ${chapters[chapterIndex].title}`
      : `Episode ${episode.number}`;
  const series = {
    id: episode.series_id,
    slug: episode.series_slug,
    title: episode.series_title,
    accent_color: episode.accent_color,
    cover_url: episode.cover_url,
  };

  const resume = () => {
    if (!detail) return;
    playback.start(detail, series, progress.position_seconds);
    navigate("/play");
  };

  return (
    <li className="continue-card-wrap">
      <button
        type="button"
        className="continue-card"
        aria-label={`Resume ${episode.series_title} episode ${episode.number}: ${episode.title}`}
        onClick={resume}
        disabled={!detail}
      >
        <CoverTile
          title={episode.series_title}
          accentColor={episode.accent_color}
          src={episode.cover_url}
          size="md"
        />
        <span className="continue-body">
          <span className="continue-kicker">
            {episode.series_title} · Episode {episode.number}
          </span>
          <span className="continue-title">{episode.title}</span>
          <span className="continue-chapter">{chapterLine}</span>
          {chapters.length > 0 ? (
            <ChapterBar
              chapters={chapters}
              duration={duration}
              position={progress.position_seconds}
              compact
            />
          ) : (
            <span className="progress-thin progress-thin-block" aria-hidden="true">
              <span
                className="progress-thin-fill"
                style={{
                  width: `${
                    duration > 0 ? Math.min(100, (progress.position_seconds / duration) * 100) : 0
                  }%`,
                }}
              />
            </span>
          )}
          <span className="continue-time">
            {formatMinutesLeft(Math.max(0, duration - progress.position_seconds))}
          </span>
        </span>
      </button>
    </li>
  );
}

function SeriesCard({ series }: { series: Series }) {
  const progressQuery = useProgressList();
  const episodesQuery = usePublishedEpisodes(series.id);
  const progress = progressByEpisode(progressQuery.data);
  const unplayed = (episodesQuery.data?.episodes ?? []).filter(
    (episode: Episode) => !progress.get(episode.id)?.completed,
  ).length;

  return (
    <li>
      <Link to={`/series/${series.id}`} className="library-series-card">
        <CoverTile
          title={series.title}
          accentColor={series.accent_color}
          src={series.cover_url}
          size="md"
        />
        <span className="library-series-body">
          <span className="library-series-title">{series.title}</span>
          <span className="library-series-author">{series.author}</span>
          <span className="library-series-meta">
            {pluralize(series.published_count, "episode")}
            {episodesQuery.isSuccess ? ` · ${unplayed} unplayed` : ""}
          </span>
        </span>
        <span className="library-series-chevron" aria-hidden="true">
          <Icon name="chevron-right" size={18} />
        </span>
      </Link>
    </li>
  );
}

export function LibraryPage() {
  const seriesQuery = useSeriesList();
  const continueQuery = useContinueListening(5);

  if (seriesQuery.isPending) {
    return (
      <div className="page">
        <h1 className="page-title">Listen</h1>
        <LoadingBlock label="Loading your library…" />
      </div>
    );
  }

  if (seriesQuery.isError) {
    return (
      <div className="page">
        <h1 className="page-title">Listen</h1>
        <ErrorBanner
          title="Could not load your library"
          message={errorMessage(seriesQuery.error)}
        />
        <button
          type="button"
          className="btn btn-secondary mt-3"
          onClick={() => void seriesQuery.refetch()}
        >
          Try again
        </button>
      </div>
    );
  }

  const publishedSeries = seriesQuery.data.series.filter((series) => series.published_count > 0);
  const continueItems = continueQuery.data?.items ?? [];

  return (
    <div className="page">
      <header className="library-header">
        <h1 className="page-title">Listen</h1>
        <p className="page-subtitle">Episodes from your published series.</p>
      </header>

      {publishedSeries.length === 0 ? (
        <div className="empty-state">
          <span className="empty-state-icon" aria-hidden="true">
            <Icon name="list" size={26} />
          </span>
          <h2>Nothing published yet</h2>
          <p>
            Episodes appear here once they are published in the admin. Publish an episode with audio
            and chapters, then it becomes available for listening.
          </p>
          <Link to="/admin/" className="btn btn-primary">
            Open the admin
          </Link>
        </div>
      ) : (
        <>
          {continueItems.length > 0 ? (
            <section className="section" aria-labelledby="continue-heading">
              <h2 id="continue-heading" className="section-title">
                Continue listening
              </h2>
              <ul className="continue-list">
                {continueItems.map((item) => (
                  <ContinueCard key={item.episode.id} item={item} />
                ))}
              </ul>
            </section>
          ) : null}

          <section className="section" aria-labelledby="series-heading">
            <h2 id="series-heading" className="section-title">
              Series
            </h2>
            <ul className="library-series-list">
              {publishedSeries.map((series) => (
                <SeriesCard key={series.id} series={series} />
              ))}
            </ul>
          </section>

          <footer className="app-footer">
            <Link to="/admin/">Manage episodes in the admin</Link>
          </footer>
        </>
      )}
    </div>
  );
}
