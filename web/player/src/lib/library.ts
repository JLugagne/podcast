import { FeedError, parseFeed, type StoredSeries } from "./feed";
import { readStorage, writeStorage } from "./storage";

const LIBRARY_KEY = "library";
const PROXY_KEY = "cors-proxy";

/** Subscriptions are stored as one JSON document in localStorage. */
export function listSubscriptions(): StoredSeries[] {
  const raw = readStorage(LIBRARY_KEY);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    return Array.isArray(parsed) ? (parsed as StoredSeries[]) : [];
  } catch {
    return [];
  }
}

function saveSubscriptions(list: StoredSeries[]): void {
  writeStorage(LIBRARY_KEY, JSON.stringify(list));
}

/**
 * Optional CORS proxy prefix, for feeds that do not send CORS headers.
 * The feed URL is appended (URL-encoded) to it.
 */
export function getProxy(): string {
  return readStorage(PROXY_KEY) ?? "";
}

export function setProxy(value: string): void {
  writeStorage(PROXY_KEY, value.trim());
}

/** Downloads a feed directly, then through the proxy when one is configured. */
export async function fetchFeedText(feedUrl: string): Promise<string> {
  let direct: Response | null = null;
  try {
    direct = await fetch(feedUrl, {
      headers: { Accept: "application/rss+xml, application/xml, text/xml" },
    });
  } catch {
    direct = null;
  }
  if (direct?.ok) return direct.text();

  const proxy = getProxy();
  if (proxy) {
    try {
      const proxied = await fetch(`${proxy}${encodeURIComponent(feedUrl)}`);
      if (proxied.ok) return proxied.text();
    } catch {
      // Fall through to the error below.
    }
  }
  if (direct) throw new FeedError(`The feed server answered ${direct.status}.`);
  throw new FeedError(
    "Could not download this feed. The server may block browser requests (CORS); set a CORS proxy in the settings.",
  );
}

function normalizeUrl(raw: string): string {
  let url: URL;
  try {
    url = new URL(raw.trim());
  } catch {
    throw new FeedError("Enter a valid feed URL.");
  }
  if (url.protocol !== "https:" && url.protocol !== "http:") {
    throw new FeedError("Enter a valid feed URL.");
  }
  return url.toString();
}

export async function addSubscription(rawUrl: string): Promise<StoredSeries> {
  const feedUrl = normalizeUrl(rawUrl);
  const list = listSubscriptions();
  if (list.some((item) => item.feed_url === feedUrl)) {
    throw new FeedError("This podcast is already in your library.");
  }
  const series = parseFeed(await fetchFeedText(feedUrl), feedUrl);
  try {
    saveSubscriptions([...list, series]);
  } catch {
    throw new FeedError("Could not save the library (storage is full or unavailable).");
  }
  return series;
}

export async function refreshSubscription(slug: string): Promise<StoredSeries> {
  const list = listSubscriptions();
  const current = list.find((item) => item.slug === slug);
  if (!current) throw new FeedError("This podcast is not in your library.");
  const next = parseFeed(await fetchFeedText(current.feed_url), current.feed_url);
  saveSubscriptions(list.map((item) => (item.slug === slug ? next : item)));
  return next;
}

export function removeSubscription(slug: string): void {
  saveSubscriptions(listSubscriptions().filter((item) => item.slug !== slug));
}
