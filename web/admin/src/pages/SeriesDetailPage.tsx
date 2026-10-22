import { useState } from "react";
import { Link, useParams } from "react-router";
import { errorMessage } from "../api/client";
import { CoverTile } from "../components/CoverTile";
import { ErrorBanner } from "../components/ErrorBanner";
import { LoadingBlock } from "../components/LoadingBlock";
import { StatusBadge } from "../components/StatusBadge";
import { useEpisodes } from "../hooks/episodes";
import { useSeries } from "../hooks/series";
import { copyText } from "../lib/clipboard";
import { formatDateTime, formatDuration } from "../lib/format";
import { withCacheBust } from "../lib/urls";

export function SeriesDetailPage() {
  const { id } = useParams<{ id: string }>();
  const seriesQuery = useSeries(id);
  const episodesQuery = useEpisodes(id);
  const [copied, setCopied] = useState(false);

  if (seriesQuery.isPending) {
    return (
      <div className="page">
        <LoadingBlock label="Loading series…" />
      </div>
    );
  }

  if (seriesQuery.isError || !seriesQuery.data) {
    return (
      <div className="page">
        <ErrorBanner
          title="Could not load this series"
          message={
            seriesQuery.isError ? errorMessage(seriesQuery.error) : "The series was not found."
          }
        />
        <p className="mt-4">
          <Link to="/">Back to series</Link>
        </p>
      </div>
    );
  }

  const series = seriesQuery.data;
  const episodes = episodesQuery.data?.episodes ?? [];
  const feedUrl = `${window.location.origin}${series.feed_url}`;

  async function handleCopyFeed() {
    setCopied(await copyText(feedUrl));
  }

  return (
    <div className="page">
      <nav className="breadcrumbs" aria-label="Breadcrumb">
        <Link to="/">Series</Link>
        <span aria-hidden="true">/</span>
        <span aria-current="page">{series.title}</span>
      </nav>

      <header className="detail-header">
        <CoverTile
          title={series.title}
          accentColor={series.accent_color}
          src={withCacheBust(series.cover_url, series.version)}
          size="lg"
        />
        <div>
          <h1 className="detail-title">{series.title}</h1>
          <p className="detail-meta">
            {series.author.trim() ? `by ${series.author}` : "Unknown author"} ·{" "}
            <span className="mono">{series.slug}</span>
          </p>
          {series.description.trim() ? (
            <p className="detail-description">{series.description}</p>
          ) : null}

          <div className="stat-row">
            <div className="stat">
              <div className="stat-value">{series.episode_count}</div>
              <div className="stat-label">Episodes</div>
            </div>
            <div className="stat">
              <div className="stat-value">{series.published_count}</div>
              <div className="stat-label">Published</div>
            </div>
            <div className="stat">
              <div className="stat-value">{formatDuration(series.total_duration_seconds)}</div>
              <div className="stat-label">Total audio</div>
            </div>
            <div className="stat">
              <div className="stat-value text-small">{formatDateTime(series.updated_at)}</div>
              <div className="stat-label">Last updated</div>
            </div>
          </div>

          <div className="row mt-4">
            <Link className="btn btn-secondary" to={`/series/${series.id}/edit`}>
              Edit series
            </Link>
            <Link className="btn btn-primary" to={`/series/${series.id}/episodes/new`}>
              New episode
            </Link>
            <button type="button" className="btn btn-secondary" onClick={handleCopyFeed}>
              Copy RSS feed URL
            </button>
          </div>

          <p className="detail-meta feed-url">
            <span aria-hidden="true">RSS feed </span>
            <span className="mono">{feedUrl}</span>
          </p>
          {copied ? (
            <p className="text-small" role="status">
              Copied to clipboard
            </p>
          ) : null}
        </div>
      </header>

      <section className="section" aria-labelledby="episodes-heading">
        <div className="section-heading">
          <h2 className="section-title" id="episodes-heading">
            Episodes
          </h2>
          {episodes.length > 0 ? (
            <span className="section-hint">
              {episodes.length} episode{episodes.length === 1 ? "" : "s"}
            </span>
          ) : null}
        </div>

        {episodesQuery.isPending ? (
          <LoadingBlock label="Loading episodes…" />
        ) : episodesQuery.isError ? (
          <ErrorBanner
            title="Could not load episodes"
            message={errorMessage(episodesQuery.error)}
          />
        ) : episodes.length === 0 ? (
          <div className="empty-state">
            <h2>No episodes yet</h2>
            <p>Add the first episode of this series to upload audio and chapters.</p>
            <Link className="btn btn-primary" to={`/series/${series.id}/episodes/new`}>
              New episode
            </Link>
          </div>
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <caption className="sr-only">Episodes of {series.title}</caption>
              <thead>
                <tr>
                  <th scope="col">#</th>
                  <th scope="col">Title</th>
                  <th scope="col">Status</th>
                  <th scope="col">Duration</th>
                  <th scope="col">Chapters</th>
                  <th scope="col">Audio</th>
                </tr>
              </thead>
              <tbody>
                {episodes.map((episode) => (
                  <tr key={episode.id}>
                    <td className="cell-number">{episode.number}</td>
                    <td>
                      <Link className="table-link" to={`/episodes/${episode.id}`}>
                        {episode.title}
                      </Link>
                    </td>
                    <td>
                      <StatusBadge status={episode.status} />
                    </td>
                    <td className="cell-number">
                      {episode.audio ? formatDuration(episode.audio.duration_seconds) : "—"}
                    </td>
                    <td className="cell-number">{episode.chapters.length}</td>
                    <td>
                      {episode.audio ? (
                        <>
                          <span aria-hidden="true">✓</span>
                          <span className="sr-only">Audio uploaded</span>
                        </>
                      ) : (
                        <>
                          <span className="muted" aria-hidden="true">
                            ✗
                          </span>
                          <span className="sr-only">No audio</span>
                        </>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
