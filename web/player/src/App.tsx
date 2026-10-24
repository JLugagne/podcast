import { BrowserRouter, Link, Route, Routes, useLocation } from "react-router";
import { MiniPlayer } from "./components/MiniPlayer";
import { type Locale, useI18n } from "./i18n";
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
  const { t } = useI18n();
  const location = useLocation();
  const isPlayer = location.pathname === "/play";

  return (
    <>
      <a className="skip-link" href="#main-content">
        {t("skipToContent")}
      </a>
      {isPlayer ? null : (
        <header className="app-header">
          <div className="app-header-inner">
            <Link to="/" className="brand">
              <span className="brand-dot" aria-hidden="true" />
              {t("appName")}
            </Link>
            <LanguagePicker />
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

/** Lets the reader override the language detected from the browser. */
function LanguagePicker() {
  const { locale, t, setLocale } = useI18n();

  return (
    <select
      className="language-picker"
      aria-label={t("language")}
      value={locale}
      onChange={(event) => setLocale(event.target.value as Locale)}
    >
      <option value="en">English</option>
      <option value="fr">Français</option>
    </select>
  );
}

function NotFoundPage() {
  const { t } = useI18n();

  return (
    <div className="page">
      <div className="empty-state">
        <h1>{t("pageNotFound")}</h1>
        <p>{t("pageNotFoundBody")}</p>
        <Link to="/" className="btn btn-primary">
          {t("backToLibrary")}
        </Link>
      </div>
    </div>
  );
}
