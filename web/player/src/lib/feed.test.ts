import { describe, expect, it } from "vitest";
import { FeedError, parseDuration, parseFeed } from "./feed";

const FEED = `<?xml version="1.0"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:podcast="https://podcastindex.org/namespace/1.0" xmlns:media="http://search.yahoo.com/mrss/">
<channel>
  <title>Le Show</title>
  <itunes:author>Ada</itunes:author>
  <itunes:image href="https://cdn.example/cover.jpg"/>
  <item><title>Newer</title><guid>b</guid><pubDate>Tue, 13 Jan 2026 10:00:00 GMT</pubDate>
    <enclosure url="https://cdn.example/b.mp3" type="audio/mpeg"/>
    <itunes:duration>1:01:05</itunes:duration>
    <itunes:image href="https://cdn.example/newer.jpg"/>
    <podcast:chapters url="https://cdn.example/b.json" type="application/json+chapters"/></item>
  <item><title>Older</title><guid>a</guid><pubDate>Tue, 06 Jan 2026 10:00:00 GMT</pubDate>
    <media:thumbnail url="https://cdn.example/older.jpg"/>
    <enclosure url="https://cdn.example/a.mp3" type="audio/mpeg"/></item>
  <item><title>No audio</title><guid>c</guid></item>
</channel></rss>`;

describe("parseFeed", () => {
  it("builds a series with episodes numbered oldest first", () => {
    const series = parseFeed(FEED, "https://feeds.example/f.xml");

    expect(series.title).toBe("Le Show");
    expect(series.author).toBe("Ada");
    expect(series.cover_url).toBe("https://cdn.example/cover.jpg");
    expect(series.slug).toMatch(/^le-show-[a-z0-9]+$/);
    expect(series.episodes.map((e) => [e.number, e.title])).toEqual([
      [1, "Older"],
      [2, "Newer"],
    ]);
    expect(series.episodes[1].duration_seconds).toBe(3665);
    expect(series.episodes[1].chapters_url).toBe("https://cdn.example/b.json");
    expect(series.total_duration_seconds).toBe(3665);
  });

  it("reads each episode's own artwork and ignores missing ones", () => {
    const series = parseFeed(FEED, "https://feeds.example/f.xml");

    expect(series.episodes[0].image_url).toBe("https://cdn.example/older.jpg");
    expect(series.episodes[1].image_url).toBe("https://cdn.example/newer.jpg");
  });

  it("rejects documents that are not podcast feeds", () => {
    expect(() => parseFeed("<html/>", "https://x.example")).toThrow(FeedError);
    expect(() => parseFeed("not xml <", "https://x.example")).toThrow(FeedError);
  });

  it("rejects feeds without playable episodes", () => {
    const empty = "<rss><channel><title>T</title></channel></rss>";
    expect(() => parseFeed(empty, "https://x.example")).toThrow(/no playable episodes/);
  });
});

describe("parseDuration", () => {
  it("handles seconds, mm:ss and hh:mm:ss", () => {
    expect(parseDuration("90")).toBe(90);
    expect(parseDuration("12:30")).toBe(750);
    expect(parseDuration("1:00:00")).toBe(3600);
    expect(parseDuration("abc")).toBe(0);
    expect(parseDuration(undefined)).toBe(0);
  });
});
