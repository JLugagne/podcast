import { useState } from "react";
import { Link } from "react-router";
import { errorMessage } from "../api/client";
import { Button } from "../components/Button";
import { CoverTile } from "../components/CoverTile";
import { ErrorBanner } from "../components/ErrorBanner";
import { TextField } from "../components/Field";
import { SkeletonGrid } from "../components/LoadingBlock";
import { useSeriesList } from "../hooks/series";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import { withCacheBust } from "../lib/urls";

export function SeriesListPage() {
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebouncedValue(search);
  const { data, isPending, isError, error, refetch, isFetching } = useSeriesList(debouncedSearch);

  const hasSearch = debouncedSearch.trim().length > 0;

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1 className="page-title">Series</h1>
          <p className="page-subtitle">Manage your podcast series, episodes and chapters.</p>
        </div>
        <Link to="/series/new" className="btn btn-primary">
          New series
        </Link>
      </header>

      <div className="search-field">
        <TextField
          id="series-search"
          label="Search series"
          type="search"
          placeholder="Search by title, author or slug"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          hint={isFetching && !isPending ? "Updating results…" : undefined}
        />
      </div>

      <div className="section">
        {isPending ? (
          <SkeletonGrid />
        ) : isError ? (
          <div className="stack">
            <ErrorBanner title="Could not load series" message={errorMessage(error)} />
            <div>
              <Button variant="secondary" onClick={() => void refetch()}>
                Try again
              </Button>
            </div>
          </div>
        ) : data.series.length === 0 ? (
          hasSearch ? (
            <div className="empty-state">
              <h2>No series match “{debouncedSearch.trim()}”</h2>
              <p>Try a different search term.</p>
              <Button variant="secondary" onClick={() => setSearch("")}>
                Clear search
              </Button>
            </div>
          ) : (
            <div className="empty-state">
              <h2>No series yet</h2>
              <p>Create your first series to start publishing episodes.</p>
              <Link to="/series/new" className="btn btn-primary">
                New series
              </Link>
            </div>
          )
        ) : (
          <ul className="series-grid" aria-label="Series">
            {data.series.map((series) => (
              <li key={series.id}>
                <Link className="series-card" to={`/series/${series.id}`}>
                  <CoverTile
                    title={series.title}
                    accentColor={series.accent_color}
                    src={withCacheBust(series.cover_url, series.version)}
                  />
                  <div className="series-card-body">
                    <span className="series-card-title">{series.title}</span>
                    <span className="series-card-author">
                      {series.author.trim() ? `by ${series.author}` : "Unknown author"}
                    </span>
                    <span className="series-card-meta">
                      <span>
                        {series.episode_count} episode{series.episode_count === 1 ? "" : "s"}
                      </span>
                      <span>{series.published_count} published</span>
                    </span>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
