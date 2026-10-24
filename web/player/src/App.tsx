import { BrowserRouter, Link, Route, Routes, useLocation } from "react-router";
import { MiniPlayer } from "./components/MiniPlayer";
import { LibraryPage } from "./pages/LibraryPage";
import { PlayerPage } from "./pages/PlayerPage";
import { SeriesPage } from "./pages/SeriesPage";
import { PlaybackProvider } from "./playback/PlaybackProvider";

export function App() {
  return (
    <BrowserRouter basename={import.meta.env.BASE_URL}>
      <PlaybackProvider>
        <AppShell />
      </PlaybackProvider>
    </BrowserRouter>
  );
}

/** Routes without a router, so tests can mount them inside a MemoryRouter. */
export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<LibraryPage />} />
      <Route path="/series/:slug" element={<SeriesPage />} />
      <Route path="/play" element={<PlayerPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}

/** Header, page content and the docked mini player. */
export function AppShell() {
  const location = useLocation();
  const isPlayer = location.pathname === "/play";

  return (
    <>
      <a className="skip-link" href="#main-content">
        Skip to content
      </a>
      {isPlayer ? null : (
        <header className="app-header">
          <div className="app-header-inner">
            <Link to="/" className="brand">
              <span className="brand-dot" aria-hidden="true" />
              Podcasts
            </Link>
          </div>
        </header>
      )}
      <main id="main-content" className={isPlayer ? "app-main app-main-player" : "app-main"}>
        <AppRoutes />
      </main>
      <MiniPlayer />
    </>
  );
}

function NotFoundPage() {
  return (
    <div className="page">
      <div className="empty-state">
        <h1>Page not found</h1>
        <p>The page you requested does not exist.</p>
        <Link to="/" className="btn btn-primary">
          Back to the library
        </Link>
      </div>
    </div>
  );
}
