import { type FormEvent, useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import { errorMessage } from "../api/client";
import type { ContinueItem, Progress, Series } from "../api/types";
import { ChapterBar } from "../components/ChapterBar";
import { CoverTile } from "../components/CoverTile";
import { ErrorBanner } from "../components/ErrorBanner";
import { FeedLinkImport } from "../components/FeedLinkImport";
import { Icon } from "../components/Icon";
import { LoadingBlock } from "../components/LoadingBlock";
import { ShareLibraryButton } from "../components/ShareLibraryButton";
import { useCatalog, useEpisode } from "../hooks/catalog";
import { useAddPodcast } from "../hooks/library";
import { progressByEpisode, useContinueListening, useProgressList } from "../hooks/listening";
import { useI18n } from "../i18n";
import { chapterIndexAt, sortChapters } from "../lib/chapters";
import { formatMinutesLeft } from "../lib/format";
import { getProxy, setProxy } from "../lib/library";
import { usePlayback } from "../playback/PlaybackProvider";

function ContinueCard({ item }: { item: ContinueItem }) {
  const { t } = useI18n();
  const { progress, episode, series } = item;
  const episodeQuery = useEpisode(episode.id);
  const playback = usePlayback();
  const navigate = useNavigate();

  const detail = episodeQuery.data;
  const chapters = sortChapters(detail?.chapters ?? []);
  const chapterIndex =
    chapters.length > 0 ? chapterIndexAt(chapters, progress.position_seconds) : -1;
  const duration = episode.duration_seconds;
  const chapterLine =
    chapterIndex >= 0
      ? t("chapterLine", { number: chapterIndex + 1, title: chapters[chapterIndex].title })
      : t("episodeLine", { number: episode.number });

  const resume = () => {
    playback.start(episode, series, progress.position_seconds);
    navigate("/play");
  };

  return (
    <li className="continue-card-wrap">
      <button
        type="button"
        className="continue-card"
        aria-label={t("resumeEpisode", {
          series: series.title,
          number: episode.number,
          title: episode.title,
        })}
        onClick={resume}
      >
        <CoverTile
          title={series.title}
          accentColor={series.accent_color}
          src={series.cover_url}
          size="md"
        />
        <span className="continue-body">
          <span className="continue-kicker">
            {t("episodeKicker", { series: series.title, number: episode.number })}
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

function SeriesCard({ series, progress }: { series: Series; progress: Map<string, Progress> }) {
  const { t } = useI18n();
  const unplayed = series.episodes.filter((episode) => !progress.get(episode.id)?.completed).length;

  return (
    <li>
      <Link to={`/series/${series.slug}`} className="library-series-card">
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
            {t("episodes", { count: series.episode_count })} · {t("unplayed", { count: unplayed })}
          </span>
        </span>
        <span className="library-series-chevron" aria-hidden="true">
          <Icon name="chevron-right" size={18} />
        </span>
      </Link>
    </li>
  );
}

function AddPodcastForm({ onClose }: { onClose: () => void }) {
  const { t } = useI18n();
  const [url, setUrl] = useState("");
  const [proxy, setProxyValue] = useState(getProxy);
  const inputRef = useRef<HTMLInputElement>(null);
  const add = useAddPodcast();

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const submit = (event: FormEvent) => {
    event.preventDefault();
    setProxy(proxy);
    add.mutate(url, { onSuccess: () => setUrl("") });
  };

  return (
    <form className="add-form" id="add-podcast-form" onSubmit={submit} aria-label={t("addPodcast")}>
      <label className="add-form-label" htmlFor="feed-url">
        {t("feedUrlLabel")}
      </label>
      <div className="add-form-row">
        <input
          id="feed-url"
          ref={inputRef}
          className="add-form-input"
          type="url"
          inputMode="url"
          placeholder={t("feedUrlPlaceholder")}
          value={url}
          onChange={(event) => setUrl(event.target.value)}
          required
        />
        <button type="submit" className="btn btn-primary" disabled={add.isPending}>
          {add.isPending ? t("adding") : t("add")}
        </button>
        <button type="button" className="btn btn-secondary" onClick={onClose}>
          {t("cancel")}
        </button>
      </div>
      {add.isError ? (
        <ErrorBanner title={t("couldNotAddPodcast")} message={errorMessage(add.error)} />
      ) : null}
      <details className="add-form-advanced">
        <summary>{t("feedBlockedSummary")}</summary>
        <label className="add-form-label" htmlFor="cors-proxy">
          {t("corsProxyLabel")}
        </label>
        <input
          id="cors-proxy"
          className="add-form-input"
          type="url"
          placeholder={t("corsProxyPlaceholder")}
          value={proxy}
          onChange={(event) => setProxyValue(event.target.value)}
        />
      </details>
    </form>
  );
}

export function LibraryPage() {
  const { t } = useI18n();
  const [adding, setAdding] = useState(false);
  const catalogQuery = useCatalog();
  const progressQuery = useProgressList();
  const continueQuery = useContinueListening(5);

  if (catalogQuery.isPending) {
    return (
      <div className="page">
        <h1 className="page-title">{t("listen")}</h1>
        <LoadingBlock label={t("loadingLibrary")} />
      </div>
    );
  }

  if (catalogQuery.isError) {
    return (
      <div className="page">
        <h1 className="page-title">{t("listen")}</h1>
        <ErrorBanner title={t("couldNotLoadLibrary")} message={errorMessage(catalogQuery.error)} />
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

  const series = catalogQuery.data.series.filter((item) => item.episode_count > 0);
  const progress = progressByEpisode(progressQuery.data);
  const continueItems = continueQuery.data.items;

  return (
    <div className="page">
      <header className="library-header">
        <div className="library-header-row">
          <div>
            <h1 className="page-title">{t("listen")}</h1>
            <p className="page-subtitle">{t("librarySubtitle")}</p>
          </div>
          <div className="library-actions">
            <button
              type="button"
              className="btn btn-secondary btn-small"
              aria-expanded={adding}
              aria-controls={adding ? "add-podcast-form" : undefined}
              onClick={() => setAdding(true)}
            >
              <Icon name="rss" size={16} />
              <span>{t("addPodcast")}</span>
            </button>
            {series.length > 0 ? <ShareLibraryButton /> : null}
          </div>
        </div>
      </header>

      <FeedLinkImport />

      {adding ? <AddPodcastForm onClose={() => setAdding(false)} /> : null}

      {series.length === 0 ? (
        <div className="empty-state">
          <span className="empty-state-icon" aria-hidden="true">
            <Icon name="list" size={26} />
          </span>
          <h2>{t("emptyLibraryTitle")}</h2>
          <p>{t("emptyLibraryBody")}</p>
          <button type="button" className="btn btn-primary" onClick={() => setAdding(true)}>
            {t("addFirstPodcast")}
          </button>
        </div>
      ) : (
        <>
          {continueItems.length > 0 ? (
            <section className="section" aria-labelledby="continue-heading">
              <h2 id="continue-heading" className="section-title">
                {t("continueListening")}
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
              {t("seriesHeading")}
            </h2>
            <ul className="library-series-list">
              {series.map((item) => (
                <SeriesCard key={item.slug} series={item} progress={progress} />
              ))}
            </ul>
          </section>
        </>
      )}
    </div>
  );
}
