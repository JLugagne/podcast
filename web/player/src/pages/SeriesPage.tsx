import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { errorMessage } from "../api/client";
import type { Episode, PlaybackSeries, Progress } from "../api/types";
import { CoverTile } from "../components/CoverTile";
import { EpisodeRow } from "../components/EpisodeRow";
import { ErrorBanner } from "../components/ErrorBanner";
import { Icon } from "../components/Icon";
import { LoadingBlock } from "../components/LoadingBlock";
import { useCatalog } from "../hooks/catalog";
import { deleteDownload, startDownload, useDownloads } from "../hooks/downloads";
import { useRefreshPodcast, useRemovePodcast } from "../hooks/library";
import {
  progressByEpisode,
  useMarkPlayed,
  useMarkUnplayed,
  useProgressList,
} from "../hooks/listening";
import { useI18n } from "../i18n";
import { type EpisodeView, readEpisodeView, writeEpisodeView } from "../lib/episodeView";
import { formatDuration } from "../lib/format";
import { usePlayback } from "../playback/PlaybackProvider";

export function SeriesPage() {
  const { t } = useI18n();
  const { slug } = useParams<{ slug: string }>();
  const catalogQuery = useCatalog();
  const progressQuery = useProgressList();
  const playback = usePlayback();
  const navigate = useNavigate();
  const markPlayed = useMarkPlayed();
  const markUnplayed = useMarkUnplayed();
  const downloads = useDownloads();
  const refresh = useRefreshPodcast();
  const remove = useRemovePodcast();
  const [view, setView] = useState<EpisodeView>(readEpisodeView);

  const changeView = (next: EpisodeView) => {
    setView(next);
    writeEpisodeView(next);
  };

  if (catalogQuery.isPending) {
    return (
      <div className="page">
        <LoadingBlock label={t("loadingSeries")} />
      </div>
    );
  }

  if (catalogQuery.isError) {
    return (
      <div className="page">
        <ErrorBanner title={t("couldNotLoadSeries")} message={errorMessage(catalogQuery.error)} />
        <button
          type="button"
          className="btn btn-secondary mt-3"
          onClick={() => void catalogQuery.refetch()}
        >
          {t("tryAgain")}
        </button>
      </div>
    );
  }

  const series = catalogQuery.data.series.find((item) => item.slug === slug);
  if (!series) {
    return (
      <div className="page">
        <div className="empty-state">
          <h2>{t("seriesNotFound")}</h2>
          <p>{t("seriesNotFoundBody")}</p>
          <Link to="/" className="btn btn-primary">
            {t("backToLibrary")}
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
  const primaryLabel = inProgress.length > 0 ? t("resume") : t("playLatest");

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
    ? t("resumeEpisodeLabel", {
        action: primaryLabel,
        number: resumeEpisode.number,
        title: resumeEpisode.title,
      })
    : primaryLabel;

  return (
    <div className="page series-page">
      <Link to="/" className="back-link">
        <Icon name="back" size={16} />
        <span>{t("libraryLink")}</span>
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
            {t("episodes", { count: episodes.length })}
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
          <div className="series-actions">
            <button
              type="button"
              className="btn btn-secondary"
              disabled={refresh.isPending}
              onClick={() => refresh.mutate(series.slug)}
            >
              {refresh.isPending ? t("refreshing") : t("refresh")}
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              onClick={() => {
                if (!window.confirm(t("removeConfirm", { title: series.title }))) return;
                remove.mutate(series.slug, { onSuccess: () => navigate("/") });
              }}
            >
              {t("remove")}
            </button>
          </div>
          {refresh.isError ? (
            <ErrorBanner title={t("couldNotRefresh")} message={errorMessage(refresh.error)} />
          ) : null}
        </div>
      </header>

      <section className="section" aria-labelledby="episodes-heading">
        <div className="section-heading">
          <h2 id="episodes-heading" className="section-title">
            {t("episodesHeading")}
          </h2>
          {episodes.length > 0 ? (
            <div className="episode-view-toggle">
              <button
                type="button"
                className={view === "list" ? "is-active" : undefined}
                aria-pressed={view === "list"}
                aria-label={t("listView")}
                onClick={() => changeView("list")}
              >
                <Icon name="list" size={18} />
              </button>
              <button
                type="button"
                className={view === "grid" ? "is-active" : undefined}
                aria-pressed={view === "grid"}
                aria-label={t("gridView")}
                onClick={() => changeView("grid")}
              >
                <Icon name="grid" size={18} />
              </button>
            </div>
          ) : null}
        </div>
        {episodes.length === 0 ? (
          <div className="empty-state">
            <h2>{t("noEpisodes")}</h2>
            <p>{t("noPlayableEpisodes")}</p>
          </div>
        ) : (
          <ul
            className={view === "grid" ? "episode-list is-grid" : "episode-list"}
            aria-label={t("episodesOf", { title: series.title })}
          >
            {episodes.map((episode) => (
              <EpisodeRow
                key={episode.id}
                episode={episode}
                progress={progress.get(episode.id)}
                isCurrent={playback.episode?.id === episode.id}
                playing={playback.episode?.id === episode.id && playback.playing}
                view={view}
                accentColor={series.accent_color}
                seriesCoverUrl={series.cover_url}
                onTogglePlay={() => handleToggle(episode)}
                onOpenPlayer={() => openPlayer(episode, savedPosition(episode))}
                onMarkPlayed={() => markPlayed.mutate(episode)}
                onMarkUnplayed={() => markUnplayed.mutate(episode.id)}
                onRestart={() => {
                  markUnplayed.mutate(episode.id);
                  startPlayback(episode, 0);
                }}
                downloaded={downloads.downloaded.has(episode.id)}
                downloading={downloads.active.get(episode.id)}
                downloadError={downloads.errors.get(episode.id)}
                onDownload={() => void startDownload(episode)}
                onRemoveDownload={() => void deleteDownload(episode.id)}
              />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
