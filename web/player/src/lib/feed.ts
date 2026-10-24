import type { Episode, Series } from "../api/types";

/** A subscription as persisted in localStorage. */
export interface StoredSeries extends Series {
  feed_url: string;
  refreshed_at: string;
}

/** The most recent episodes kept per feed, to stay within the localStorage quota. */
export const MAX_EPISODES_PER_FEED = 200;
const MAX_DESCRIPTION_LENGTH = 600;

export class FeedError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "FeedError";
  }
}

const ITUNES_NS = "http://www.itunes.com/dtds/podcast-1.0.dtd";
const PODCAST_NS = "https://podcastindex.org/namespace/1.0";

/** Small stable hash, used to build ids from feed URLs and GUIDs. */
export function shortHash(value: string): string {
  let hash = 5381;
  for (let index = 0; index < value.length; index += 1) {
    hash = ((hash << 5) + hash + value.charCodeAt(index)) >>> 0;
  }
  return hash.toString(36);
}

function slugify(value: string): string {
  return value
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 40);
}

export function seriesSlugFor(title: string, feedUrl: string): string {
  const base = slugify(title) || "podcast";
  return `${base}-${shortHash(feedUrl)}`;
}

/** A deterministic accent color so every show keeps the same tile color. */
export function accentColorFor(seed: string): string {
  const hue = (Number.parseInt(shortHash(seed), 36) % 360) / 360;
  const [saturation, lightness] = [0.62, 0.68];
  const q = lightness + saturation - lightness * saturation;
  const p = 2 * lightness - q;
  const channel = (offset: number): string => {
    let t = hue + offset;
    if (t < 0) t += 1;
    if (t > 1) t -= 1;
    let value = p;
    if (t < 1 / 6) value = p + (q - p) * 6 * t;
    else if (t < 1 / 2) value = q;
    else if (t < 2 / 3) value = p + (q - p) * (2 / 3 - t) * 6;
    return Math.round(value * 255)
      .toString(16)
      .padStart(2, "0");
  };
  return `#${channel(1 / 3)}${channel(0)}${channel(-1 / 3)}`.toUpperCase();
}

/** Parses "1:02:03", "12:30" or "3600" into seconds. */
export function parseDuration(raw: string | null | undefined): number {
  if (!raw) return 0;
  const parts = raw.trim().split(":");
  if (parts.length === 0 || parts.length > 3) return 0;
  let total = 0;
  for (const part of parts) {
    const value = Number(part);
    if (!Number.isFinite(value) || value < 0) return 0;
    total = total * 60 + value;
  }
  return total;
}

function childText(parent: Element, name: string, namespace?: string): string {
  for (const child of Array.from(parent.children)) {
    const matches = namespace
      ? child.localName === name && child.namespaceURI === namespace
      : child.localName === name && !child.namespaceURI;
    if (matches) return (child.textContent ?? "").trim();
  }
  return "";
}

function childElement(parent: Element, name: string, namespace?: string): Element | null {
  for (const child of Array.from(parent.children)) {
    const matches = namespace
      ? child.localName === name && child.namespaceURI === namespace
      : child.localName === name && !child.namespaceURI;
    if (matches) return child;
  }
  return null;
}

function httpsOnly(raw: string | null | undefined): string | null {
  if (!raw) return null;
  return /^https?:\/\//i.test(raw.trim()) ? raw.trim() : null;
}

function stripHtml(raw: string): string {
  const text = raw
    .replace(/<[^>]*>/g, " ")
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/\s+/g, " ")
    .trim();
  return text.length > MAX_DESCRIPTION_LENGTH
    ? `${text.slice(0, MAX_DESCRIPTION_LENGTH - 1)}…`
    : text;
}

function parseDate(raw: string): string {
  const time = Date.parse(raw);
  return Number.isNaN(time) ? new Date(0).toISOString() : new Date(time).toISOString();
}

/**
 * Parses an RSS 2.0 podcast feed. Episodes are numbered oldest first (or by
 * `itunes:episode` when every item carries one) and ordered ascending, which is
 * what the series page expects.
 */
export function parseFeed(xml: string, feedUrl: string): StoredSeries {
  const doc = new DOMParser().parseFromString(xml, "application/xml");
  if (doc.querySelector("parsererror")) throw new FeedError("This URL is not a valid XML feed.");
  const channel = doc.querySelector("rss > channel");
  if (!channel) throw new FeedError("This XML document is not a podcast RSS feed.");

  const title = childText(channel, "title") || feedUrl;
  const slug = seriesSlugFor(title, feedUrl);
  const coverUrl =
    httpsOnly(childElement(channel, "image", ITUNES_NS)?.getAttribute("href")) ??
    httpsOnly(childText(childElement(channel, "image") ?? channel, "url"));

  interface Draft {
    guid: string;
    title: string;
    description: string;
    published_at: string;
    duration_seconds: number;
    audio_url: string;
    chapters_url: string | null;
    itunes_number: number | null;
  }

  const drafts: Draft[] = [];
  for (const item of Array.from(channel.children).filter((node) => node.localName === "item")) {
    const audioUrl = httpsOnly(childElement(item, "enclosure")?.getAttribute("url"));
    if (!audioUrl) continue;
    const guid = childText(item, "guid") || audioUrl;
    const number = Number.parseInt(childText(item, "episode", ITUNES_NS), 10);
    drafts.push({
      guid,
      title: childText(item, "title") || "Untitled episode",
      description: stripHtml(
        childText(item, "description") || childText(item, "summary", ITUNES_NS),
      ),
      published_at: parseDate(childText(item, "pubDate")),
      duration_seconds: parseDuration(childText(item, "duration", ITUNES_NS)),
      audio_url: audioUrl,
      chapters_url: httpsOnly(childElement(item, "chapters", PODCAST_NS)?.getAttribute("url")),
      itunes_number: Number.isFinite(number) && number > 0 ? number : null,
    });
  }
  if (drafts.length === 0) throw new FeedError("This feed has no playable episodes.");

  drafts.sort((left, right) => left.published_at.localeCompare(right.published_at));
  const recent = drafts.slice(-MAX_EPISODES_PER_FEED);
  const useItunes =
    recent.every((draft) => draft.itunes_number !== null) &&
    new Set(recent.map((draft) => draft.itunes_number)).size === recent.length;
  const offset = drafts.length - recent.length;

  const seen = new Set<string>();
  const episodes: Episode[] = recent.map((draft, index) => {
    let episodeSlug = shortHash(draft.guid);
    while (seen.has(episodeSlug)) episodeSlug = `${episodeSlug}x`;
    seen.add(episodeSlug);
    return {
      id: `${slug}/${episodeSlug}`,
      slug: episodeSlug,
      series_slug: slug,
      number: useItunes ? (draft.itunes_number as number) : offset + index + 1,
      title: draft.title,
      description: draft.description,
      published_at: draft.published_at,
      duration_seconds: draft.duration_seconds,
      chapter_count: 0,
      audio_url: draft.audio_url,
      chapters_url: draft.chapters_url,
    };
  });
  episodes.sort((left, right) => left.number - right.number);

  return {
    slug,
    feed_url: feedUrl,
    refreshed_at: new Date().toISOString(),
    title,
    author: childText(channel, "author", ITUNES_NS) || childText(channel, "managingEditor"),
    description: stripHtml(childText(channel, "description")),
    accent_color: accentColorFor(feedUrl),
    cover_url: coverUrl,
    episode_count: episodes.length,
    total_duration_seconds: episodes.reduce(
      (total, episode) => total + episode.duration_seconds,
      0,
    ),
    episodes,
  };
}
