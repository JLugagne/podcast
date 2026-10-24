import { Link, useNavigate, useParams } from "react-router";
import { errorMessage } from "../api/client";
import type { Episode, PlaybackSeries, Progress } from "../api/types";
import { CoverTile } from "../components/CoverTile";
import { EpisodeRow } from "../components/EpisodeRow";
import { ErrorBanner } from "../components/ErrorBanner";
import { Icon } from "../components/Icon";
import { LoadingBlock } from "../components/LoadingBlock";
import { useCatalog } from "../hooks/catalog";
import {
  progressByEpisode,
  useMarkPlayed,
  useMarkUnplayed,
  useProgressList,
} from "../hooks/listening";
import { formatDuration, pluralize } from "../lib/format";
import { usePlayback } from "../playback/PlaybackProvider";

export function SeriesPage() {
  const { slug } = useParams<{ slug: string }>();
  const catalogQuery = useCatalog();
  const progressQuery = useProgressList();
  const playback = usePlayback();
  const navigate = useNavigate();
  const markPlayed = useMarkPlayed();
  const markUnplayed = useMarkUnplayed();

  if (catalogQuery.isPending) {
    return (
      <div className="page">
        <LoadingBlock label="Loading series…" />
      </div>
    );
  }

  if (catalogQuery.isError) {
    return (
      <div className="page">
        <ErrorBanner
          title="Could not load this series"
          message={errorMessage(catalogQuery.error)}
        />
        <button
          type="button"
          className="btn btn-secondary mt-3"
          onClick={() => void catalogQuery.refetch()}
        >
          Try again
        </button>
      </div>
    );
  }

  const series = catalogQuery.data.series.find((item) => item.slug === slug);
  if (!series) {
    return (
      <div className="page">
        <div className="empty-state">
          <h2>Series not found</h2>
          <p>This series does not exist or has been removed.</p>
          <Link to="/" className="btn btn-primary">
            Back to the library
          </Link>
        </div>
      </div>
    );
  }

  const playbackSeries: PlaybackSeries = {
    slug: series.slug,
    title: series.title,
    accent_color: series.accent_color,
    cover_url: series.cover_url,
  };
  const progress = progressByEpisode(progressQuery.data);
  const episodes = series.episodes;
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

  const savedPosition = (episode: Episode): number => {
    const row: Progress | undefined = progress.get(episode.id);
    if (!row || row.completed) return 0;
    return row.position_seconds;
  };

  const startPlayback = (episode: Episode, positionSeconds: number) => {
    playback.start(episode, playbackSeries, positionSeconds);
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
            {series.total_duration_seconds > 0
              ? ` · ${formatDuration(series.total_duration_seconds)}`
              : ""}
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

      <section className="section" aria-labelledby="episodes-heading">
        <div className="section-heading">
          <h2 id="episodes-heading" className="section-title">
            Episodes
          </h2>
        </div>
        {episodes.length === 0 ? (
          <div className="empty-state">
            <h2>No episodes</h2>
            <p>Add an episode JSON to this series to see it here.</p>
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
                onMarkPlayed={() => markPlayed.mutate(episode)}
                onMarkUnplayed={() => markUnplayed.mutate(episode.id)}
              />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
