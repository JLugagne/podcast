import { errorMessage } from "../api/errors";
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

export interface ImportOutcome {
  /** Feeds that were downloaded and added to the library. */
  added: StoredSeries[];
  /** Feed URLs skipped because they were already subscribed. */
  skipped: string[];
  /** Feed URLs that could not be added, with the reason. */
  failed: { feed_url: string; message: string }[];
}

/**
 * Subscribes to every feed of a shared subscription link. Feeds already in the
 * library are skipped, and one failing feed does not prevent the others from
 * being added.
 */
export async function importSubscriptions(rawUrls: string[]): Promise<ImportOutcome> {
  const list = listSubscriptions();
  const subscribed = new Set(list.map((item) => item.feed_url));
  const outcome: ImportOutcome = { added: [], skipped: [], failed: [] };
  const pending: string[] = [];

  for (const rawUrl of rawUrls) {
    let feedUrl: string;
    try {
      feedUrl = normalizeUrl(rawUrl);
    } catch (error) {
      outcome.failed.push({ feed_url: rawUrl, message: errorMessage(error) });
      continue;
    }
    if (subscribed.has(feedUrl)) {
      if (!outcome.skipped.includes(feedUrl)) outcome.skipped.push(feedUrl);
    } else if (!pending.includes(feedUrl)) {
      pending.push(feedUrl);
    }
  }

  const fetched = await Promise.all(
    pending.map(async (feedUrl) => {
      try {
        return { series: parseFeed(await fetchFeedText(feedUrl), feedUrl), feedUrl, error: null };
      } catch (error) {
        return { series: null, feedUrl, error };
      }
    }),
  );

  for (const item of fetched) {
    if (item.series) {
      list.push(item.series);
      outcome.added.push(item.series);
    } else {
      outcome.failed.push({ feed_url: item.feedUrl, message: errorMessage(item.error) });
    }
  }

  if (outcome.added.length > 0) {
    try {
      saveSubscriptions(list);
    } catch {
      throw new FeedError("Could not save the library (storage is full or unavailable).");
    }
  }
  return outcome;
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
