import { readFileSync } from "node:fs";
import type { BrowserContext, Route } from "@playwright/test";
import { AUDIO_DURATION_SECONDS, AUDIO_PATH, ensureFixtures } from "./generate.cjs";

/** The origin that stands in for the user's podcast host (R2 bucket). */
export const MEDIA_ORIGIN = "https://media.e2e.test";
export const FEED_URL = `${MEDIA_ORIGIN}/harbour-lights/feed.xml`;

export const seriesTitle = "Harbour Lights";
export const episodes = [
  {
    title: "The harbour at dawn",
    date: "Thu, 01 Oct 2026 06:00:00 GMT",
    chapters: ["Opening bell", "Harbour at dawn", "Deep current", "Finale"],
  },
  {
    title: "Tides and tables",
    date: "Thu, 08 Oct 2026 06:00:00 GMT",
    chapters: ["First light", "Tide tables", "The long swell", "Sign-off"],
  },
  {
    title: "Night watch",
    date: "Thu, 15 Oct 2026 06:00:00 GMT",
    chapters: ["Night watch", "Fog signal", "Sea glass", "Last call"],
  },
];
/** Chapters start every 30 seconds of the 120 second tone. */
export const chapterStarts = [0, 30, 60, 90];

const CORS = {
  "access-control-allow-origin": "*",
  "access-control-expose-headers": "Content-Length, Content-Range, Accept-Ranges, Content-Type",
};

function feedXml(): string {
  const items = episodes
    .map(
      (episode, index) => `<item>
  <title>${episode.title}</title>
  <guid>harbour-${index + 1}</guid>
  <pubDate>${episode.date}</pubDate>
  <description>Episode ${index + 1} of the harbour sessions.</description>
  <enclosure url="${MEDIA_ORIGIN}/harbour-lights/tone.mp3" length="0" type="audio/mpeg"/>
  <itunes:duration>${AUDIO_DURATION_SECONDS}</itunes:duration>
  <podcast:chapters url="${MEDIA_ORIGIN}/harbour-lights/ep-${index + 1}.json" type="application/json+chapters"/>
</item>`,
    )
    .join("\n");
  return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:podcast="https://podcastindex.org/namespace/1.0">
<channel>
  <title>${seriesTitle}</title>
  <itunes:author>Mira Vale</itunes:author>
  <description>Field recordings from a working harbour, recorded at first light.</description>
${items}
</channel>
</rss>`;
}

function chaptersJson(index: number): string {
  return JSON.stringify({
    version: "1.2.0",
    chapters: episodes[index].chapters.map((title, position) => ({
      startTime: chapterStarts[position],
      title,
    })),
  });
}

export interface HarbourHost {
  /** While true, the audio file answers with a network failure (simulates being offline). */
  audioBlocked: boolean;
  /** Audio requests received so far. */
  audioRequests: number;
}

/** Serves the feed, chapters and audio of the fake host, with CORS and Range support. */
export async function installHarbourHost(context: BrowserContext): Promise<HarbourHost> {
  ensureFixtures();
  const audio = readFileSync(AUDIO_PATH);
  const host: HarbourHost = { audioBlocked: false, audioRequests: 0 };

  const serveAudio = async (route: Route): Promise<void> => {
    host.audioRequests += 1;
    if (host.audioBlocked) return route.abort("internetdisconnected");
    const range = /bytes=(\d*)-(\d*)/.exec(route.request().headers().range ?? "");
    if (!range) {
      return route.fulfill({
        status: 200,
        body: audio,
        headers: { ...CORS, "content-type": "audio/mpeg", "accept-ranges": "bytes" },
      });
    }
    const start = range[1] ? Number(range[1]) : 0;
    const end = Math.min(audio.length - 1, range[2] ? Number(range[2]) : audio.length - 1);
    return route.fulfill({
      status: 206,
      body: audio.subarray(start, end + 1),
      headers: {
        ...CORS,
        "content-type": "audio/mpeg",
        "accept-ranges": "bytes",
        "content-range": `bytes ${start}-${end}/${audio.length}`,
      },
    });
  };

  await context.route(`${MEDIA_ORIGIN}/**`, async (route) => {
    const { pathname } = new URL(route.request().url());
    if (route.request().method() === "OPTIONS") {
      return route.fulfill({
        status: 204,
        headers: { ...CORS, "access-control-allow-headers": "*", "access-control-allow-methods": "GET, HEAD" },
      });
    }
    if (pathname.endsWith("/feed.xml")) {
      return route.fulfill({
        status: 200,
        body: feedXml(),
        headers: { ...CORS, "content-type": "application/rss+xml" },
      });
    }
    const chapters = /\/ep-(\d+)\.json$/.exec(pathname);
    if (chapters) {
      return route.fulfill({
        status: 200,
        body: chaptersJson(Number(chapters[1]) - 1),
        headers: { ...CORS, "content-type": "application/json" },
      });
    }
    if (pathname.endsWith("/tone.mp3")) return serveAudio(route);
    return route.fulfill({ status: 404, headers: CORS });
  });
  return host;
}
