import { BrowserRouter, Link, Route, Routes } from "react-router";
import { EpisodeEditorPage } from "./pages/EpisodeEditorPage";
import { NewEpisodePage } from "./pages/NewEpisodePage";
import { SeriesDetailPage } from "./pages/SeriesDetailPage";
import { SeriesFormPage } from "./pages/SeriesFormPage";
import { SeriesListPage } from "./pages/SeriesListPage";

export function App() {
  return (
    <BrowserRouter basename="/admin">
      <SkipLink />
      <AppHeader />
      <main id="main-content">
        <AppRoutes />
      </main>
    </BrowserRouter>
  );
}

/** Routes without a router, so tests can mount them inside a MemoryRouter. */
export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<SeriesListPage />} />
      <Route path="/series/new" element={<SeriesFormPage />} />
      <Route path="/series/:id" element={<SeriesDetailPage />} />
      <Route path="/series/:id/edit" element={<SeriesFormPage />} />
      <Route path="/series/:id/episodes/new" element={<NewEpisodePage />} />
      <Route path="/episodes/:id" element={<EpisodeEditorPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}

function SkipLink() {
  return (
    <a className="skip-link" href="#main-content">
      Skip to content
    </a>
  );
}

export function AppHeader() {
  return (
    <header className="app-header">
      <div className="app-header-inner">
        <Link to="/" className="brand">
          <span className="brand-dot" aria-hidden="true" />
          Podcast admin
        </Link>
        <a className="app-header-link" href="/">
          Open listener app
        </a>
      </div>
    </header>
  );
}

function NotFoundPage() {
  return (
    <div className="page">
      <div className="empty-state">
        <h2>Page not found</h2>
        <p>The page you requested does not exist.</p>
        <Link to="/" className="btn btn-primary">
          Back to series
        </Link>
      </div>
    </div>
  );
}
