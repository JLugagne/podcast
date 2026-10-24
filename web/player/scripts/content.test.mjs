import { mkdir, mkdtemp, readFile, stat, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { buildSite, ContentError, DEFAULT_ACCENT_COLOR, loadContent } from "./content.mjs";

async function writeTree(files) {
  const root = await mkdtemp(path.join(tmpdir(), "podcast-content-"));
  for (const [name, body] of Object.entries(files)) {
    const target = path.join(root, ...name.split("/"));
    await mkdir(path.dirname(target), { recursive: true });
    await writeFile(target, body);
  }
  return root;
}

function validTree() {
  return {
    "site.json": `{"title":"Podcasts","description":"Shows","author":"Ada"}`,
    "series/alpha/series.json": `{"title":"Alpha","author":"Ada","description":"First show"}`,
    "series/alpha/media/one.mp3": "audio",
    "series/alpha/episodes/01-first.json": `{
      "number": 1,
      "title": "First",
      "description": "An episode",
      "published_at": "2026-10-24",
      "duration_seconds": 120,
      "audio_url": "media/one.mp3",
      "chapters": [
        {"startTime": 30, "title": "Second"},
        {"startTime": 0, "title": "First"}
      ]
    }`,
    "series/zeta/series.json": `{"title":"Zeta","author":"Ada","description":"Second show","accent_color":"#112233"}`,
    "series/zeta/episodes/01-only.json": `{
      "number": 1,
      "title": "Only",
      "published_at": "2026-10-25T10:00:00Z",
      "duration_seconds": 60,
      "audio_url": "https://cdn.example.com/only.mp3",
      "chapters": [{"startTime": 0, "title": "Intro", "toc": false}]
    }`,
  };
}

describe("loadContent", () => {
  it("loads, validates and normalizes a content tree", async () => {
    const catalog = await loadContent(await writeTree(validTree()));

    expect(catalog.site.title).toBe("Podcasts");
    expect(catalog.series.map((series) => series.slug)).toEqual(["alpha", "zeta"]);

    const [alpha, zeta] = catalog.series;
    expect(alpha.accent_color).toBe(DEFAULT_ACCENT_COLOR);
    expect(alpha.episodes[0]).toMatchObject({
      id: "alpha/01-first",
      series_slug: "alpha",
      slug: "01-first",
      audio_url: "media/alpha/01-first/one.mp3",
    });
    expect(alpha.episodes[0].audioSource).not.toBe("");
    expect(alpha.episodes[0].chapters.map((chapter) => chapter.title)).toEqual(["First", "Second"]);
    expect(zeta.episodes[0].audio_url).toBe("https://cdn.example.com/only.mp3");
    expect(zeta.episodes[0].audioSource).toBe("");
  });

  it("reports every validation problem in one pass", async () => {
    const files = validTree();
    files["series/broken/series.json"] = `{"title":"","accent_color":"blue"}`;
    files["series/broken/episodes/Bad_Slug.json"] = `{
      "number": 0,
      "title": "",
      "published_at": "yesterday",
      "duration_seconds": -1,
      "audio_url": "../escape.mp3",
      "chapters": [
        {"startTime": 10, "title": ""},
        {"startTime": 10, "title": "Duplicate"}
      ]
    }`;

    await expect(loadContent(await writeTree(files))).rejects.toThrow(ContentError);

    const error = await loadContent(await writeTree(files)).catch((reason) => reason);
    const message = error.message;
    for (const expected of [
      "slug must match",
      "title must be 1..200 characters",
      "accent_color must match",
      "number must be at least 1",
      "published_at must be",
      "duration_seconds must be",
      "must stay inside the series directory",
      "the first chapter must start at 0",
      "startTime duplicates the previous chapter",
    ]) {
      expect(message).toContain(expected);
    }
  });

  it("rejects duplicate episode numbers", async () => {
    const files = validTree();
    files["series/alpha/episodes/02-second.json"] = `{
      "number": 1,
      "title": "Second",
      "published_at": "2026-10-24",
      "duration_seconds": 60,
      "audio_url": "https://cdn.example.com/second.mp3",
      "chapters": [{"startTime": 0, "title": "Intro"}]
    }`;

    await expect(loadContent(await writeTree(files))).rejects.toThrow(/already used by/);
  });

  it("rejects a chapter starting after the episode duration", async () => {
    const files = validTree();
    files["series/alpha/episodes/01-first.json"] = `{
      "number": 1,
      "title": "First",
      "published_at": "2026-10-24",
      "duration_seconds": 20,
      "audio_url": "https://cdn.example.com/first.mp3",
      "chapters": [{"startTime": 0, "title": "A"}, {"startTime": 30, "title": "B"}]
    }`;

    await expect(loadContent(await writeTree(files))).rejects.toThrow(/before duration_seconds/);
  });

  it("fails when site.json is missing", async () => {
    const root = await mkdtemp(path.join(tmpdir(), "podcast-content-"));
    await expect(loadContent(root)).rejects.toThrow(/site.json/);
  });
});

describe("buildSite", () => {
  it("writes the index, episode documents, media and the fallback files", async () => {
    const content = await writeTree(validTree());
    const out = await mkdtemp(path.join(tmpdir(), "podcast-site-"));
    await writeFile(path.join(out, "index.html"), "<!doctype html>");

    const result = await buildSite(content, out);
    expect(result).toEqual({ series: 2, episodes: 2 });

    const index = JSON.parse(await readFile(path.join(out, "data", "index.json"), "utf8"));
    expect(index.site.title).toBe("Podcasts");
    expect(index.series[0].episode_count).toBe(1);
    expect(index.series[0].episodes[0].chapter_count).toBe(2);

    const episode = JSON.parse(
      await readFile(path.join(out, "data", "episodes", "alpha", "01-first.json"), "utf8"),
    );
    expect(episode.id).toBe("alpha/01-first");
    expect(episode.audio_url).toBe("media/alpha/01-first/one.mp3");
    expect(episode.audioSource).toBeUndefined();
    expect(episode.chapters).toHaveLength(2);

    expect(await readFile(path.join(out, "media", "alpha", "01-first", "one.mp3"), "utf8")).toBe(
      "audio",
    );
    expect((await stat(path.join(out, "404.html"))).isFile()).toBe(true);
    expect((await stat(path.join(out, ".nojekyll"))).isFile()).toBe(true);
  });

  it("requires the player build to be present", async () => {
    const content = await writeTree(validTree());
    const out = await mkdtemp(path.join(tmpdir(), "podcast-site-"));
    await expect(buildSite(content, out)).rejects.toThrow(/index\.html/);
  });
});
