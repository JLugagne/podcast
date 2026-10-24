import { useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { errorMessage } from "../api/client";
import { useImportSubscriptions } from "../hooks/library";
import { useI18n } from "../i18n";
import { feedUrlsFromSearch } from "../lib/subscriptionLinks";
import { ErrorBanner } from "./ErrorBanner";

/**
 * Subscribes to the feeds of a shared link (`?feed=<url>&feed=<url>`) when the
 * app opens with one. Feeds already in the library are skipped.
 */
export function FeedLinkImport() {
  const { t } = useI18n();
  const location = useLocation();
  const navigate = useNavigate();
  const importFeeds = useImportSubscriptions();
  const [urls] = useState(() => feedUrlsFromSearch(location.search));
  const started = useRef(false);

  useEffect(() => {
    if (started.current || urls.length === 0) return;
    started.current = true;
    // The feeds live in the library now: drop the list from the address bar so
    // reloading the page does not fetch every feed again.
    navigate({ pathname: location.pathname, search: "" }, { replace: true });
    importFeeds.mutate(urls);
  }, [urls, navigate, location.pathname, importFeeds]);

  if (urls.length === 0) return null;

  if (importFeeds.isPending) {
    return (
      <div className="banner" role="status">
        <span className="banner-icon" aria-hidden="true">
          …
        </span>
        <div className="banner-body">
          <span className="banner-title">{t("importingTitle")}</span>
          <span>{t("feedsToCheck", { count: urls.length })}</span>
        </div>
      </div>
    );
  }

  if (importFeeds.isError) {
    return <ErrorBanner title={t("importFailedTitle")} message={errorMessage(importFeeds.error)} />;
  }

  const outcome = importFeeds.data;
  if (!outcome) return null;

  const parts: string[] = [];
  if (outcome.added.length > 0) {
    parts.push(t("importAdded", { count: outcome.added.length }));
  }
  if (outcome.skipped.length > 0) {
    parts.push(t("importSkipped", { count: outcome.skipped.length }));
  }
  if (outcome.failed.length > 0) {
    parts.push(t("importFailed", { count: outcome.failed.length }));
  }
  const failed = outcome.failed.length > 0;

  return (
    <div className={failed ? "banner banner-error" : "banner"} role={failed ? "alert" : "status"}>
      <span className="banner-icon" aria-hidden="true">
        {failed ? "!" : "✓"}
      </span>
      <div className="banner-body">
        <span className="banner-title">{parts.join(" · ")}</span>
        {failed ? (
          <ul className="import-failures">
            {outcome.failed.map((failure) => (
              <li key={failure.feed_url}>
                {failure.feed_url}: {failure.message}
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </div>
  );
}
