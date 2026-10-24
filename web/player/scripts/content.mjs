#!/usr/bin/env node
// Build-time content loader for the static podcast site.
//
// It validates the content tree, writes the generated site data next to the
// player build and copies any audio referenced by a relative path.
//
// Usage:
//   node scripts/content.mjs [--check] [--content <dir>] [--out <dir>]
//
// CONTENT_DIR and OUT_DIR environment variables set the defaults.

import { copyFile, mkdir, readdir, readFile, stat, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const DEFAULT_ACCENT_COLOR = "#7A9CFF";
export const MAX_TITLE_LENGTH = 200;
export const MAX_DESCRIPTION_LENGTH = 10_000;
export const MAX_SERIES_DESCRIPTION = 5_000;
export const MAX_CHAPTER_TITLE = 300;
export const MAX_CHAPTERS = 500;
export const MAX_SLUG_LENGTH = 120;

const SLUG_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;
const COLOR_PATTERN = /^#[0-9A-Fa-f]{6}$/;
const SLUG_HINT = "lowercase letters, digits and single hyphens";

/** ContentError carries every validation problem found in one pass. */
export class ContentError extends Error {
  constructor(problems) {
    super(`content validation failed:\n${problems.map((p) => `- ${p}`).join("\n")}`);
    this.name = "ContentError";
    this.problems = problems;
  }
}

/**
 * Loads and validates the content tree rooted at root.
 *
 * Layout:
 *   site.json
 *   series/<slug>/series.json
 *   series/<slug>/episodes/<episode-slug>.json
 */
export async function loadContent(root) {
  const problems = [];
  const site = await readJSON(path.join(root, "site.json"));
  if (!String(site.title ?? "").trim()) problems.push("site.json: title is required");

  const seriesRoot = path.join(root, "series");
  let entries = [];
  try {
    entries = await readdir(seriesRoot, { withFileTypes: true });
  } catch (error) {
    if (error.code !== "ENOENT") throw error;
  }
  const series = [];
  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    series.push(await loadSeries(seriesRoot, entry.name, problems));
  }
  series.sort((a, b) => a.title.localeCompare(b.title) || a.slug.localeCompare(b.slug));

  if (problems.length > 0) throw new ContentError(problems);
  return {
    site: {
      title: site.title.trim(),
      description: site.description ?? "",
      author: site.author ?? "",
    },
    series,
  };
}

async function loadSeries(seriesRoot, slug, problems) {
  const dir = path.join(seriesRoot, slug);
  const raw = await readJSON(path.join(dir, "series.json"));
  const series = {
    slug,
    dir,
    title: String(raw.title ?? "").trim(),
    author: String(raw.author ?? "").trim(),
    description: String(raw.description ?? "").trim(),
    accent_color: String(raw.accent_color ?? "").trim(),
    cover_url: String(raw.cover_url ?? "").trim(),
    episodes: [],
  };

  const where = `series/${slug}`;
  if (!validSlug(slug)) problems.push(`${where}: slug must match ${SLUG_HINT}`);
  if (count(series.title) < 1 || count(series.title) > MAX_TITLE_LENGTH) {
    problems.push(`${where}: title must be 1..${MAX_TITLE_LENGTH} characters`);
  }
  if (count(series.author) > MAX_TITLE_LENGTH) {
    problems.push(`${where}: author must be at most ${MAX_TITLE_LENGTH} characters`);
  }
  if (count(series.description) > MAX_SERIES_DESCRIPTION) {
    problems.push(`${where}: description must be at most ${MAX_SERIES_DESCRIPTION} characters`);
  }
  if (series.accent_color === "") series.accent_color = DEFAULT_ACCENT_COLOR;
  else if (!COLOR_PATTERN.test(series.accent_color)) {
    problems.push(`${where}: accent_color must match #RRGGBB`);
  }
  if (series.cover_url !== "" && !isAbsoluteUrl(series.cover_url)) {
    problems.push(`${where}: cover_url must be an absolute HTTP(S) URL`);
  }

  const episodesDir = path.join(dir, "episodes");
  let entries = [];
  try {
    entries = await readdir(episodesDir, { withFileTypes: true });
  } catch {
    problems.push(`${where}: cannot read episodes directory`);
  }

  const numbers = new Map();
  for (const entry of entries) {
    if (entry.isDirectory() || !entry.name.endsWith(".json")) continue;
    const episodeSlug = entry.name.slice(0, -".json".length);
    const rawEpisode = await readJSON(path.join(episodesDir, entry.name));
    const episode = {
      id: `${slug}/${episodeSlug}`,
      slug: episodeSlug,
      series_slug: slug,
      number: rawEpisode.number,
      title: String(rawEpisode.title ?? "").trim(),
      description: String(rawEpisode.description ?? "").trim(),
      published_at: String(rawEpisode.published_at ?? "").trim(),
      duration_seconds: rawEpisode.duration_seconds ?? 0,
      audio_url: String(rawEpisode.audio_url ?? "").trim(),
      chapters: Array.isArray(rawEpisode.chapters) ? rawEpisode.chapters.map(normalizeChapter) : [],
      audioSource: "",
    };
    await validateEpisode(episode, series, problems);

    if (numbers.has(episode.number)) {
      problems.push(
        `${slug}/${episodeSlug}: episode number ${episode.number} is already used by ${numbers.get(episode.number)}`,
      );
    } else {
      numbers.set(episode.number, episodeSlug);
    }
    series.episodes.push(episode);
  }
  series.episodes.sort((a, b) => a.number - b.number);
  return series;
}

async function validateEpisode(episode, series, problems) {
  const where = `series/${series.slug}/episodes/${episode.slug}`;
  if (!validSlug(episode.slug)) problems.push(`${where}: slug must match ${SLUG_HINT}`);
  if (!Number.isInteger(episode.number) || episode.number < 1) {
    problems.push(`${where}: number must be at least 1`);
  }
  if (count(episode.title) < 1 || count(episode.title) > MAX_TITLE_LENGTH) {
    problems.push(`${where}: title must be 1..${MAX_TITLE_LENGTH} characters`);
  }
  if (count(episode.description) > MAX_DESCRIPTION_LENGTH) {
    problems.push(`${where}: description must be at most ${MAX_DESCRIPTION_LENGTH} characters`);
  }
  if (!validDate(episode.published_at)) {
    problems.push(`${where}: published_at must be YYYY-MM-DD or RFC 3339`);
  }
  if (
    typeof episode.duration_seconds !== "number" ||
    !Number.isFinite(episode.duration_seconds) ||
    episode.duration_seconds < 0
  ) {
    problems.push(`${where}: duration_seconds must be a positive number`);
  }
  await resolveAudio(episode, series, where, problems);
  validateChapters(episode, where, problems);
}

async function resolveAudio(episode, series, where, problems) {
  if (episode.audio_url === "") {
    problems.push(`${where}: audio_url is required`);
    return;
  }
  if (isAbsoluteUrl(episode.audio_url)) return;
  if (episode.audio_url.includes("://")) {
    problems.push(
      `${where}: audio_url must be an absolute HTTP(S) URL or a path relative to the series directory`,
    );
    return;
  }

  const clean = path.posix.normalize(episode.audio_url.replaceAll("\\", "/"));
  if (clean === "." || clean === ".." || clean.startsWith("../") || path.posix.isAbsolute(clean)) {
    problems.push(`${where}: audio_url must stay inside the series directory`);
    return;
  }
  const source = path.join(series.dir, clean);
  const info = await stat(source).catch(() => null);
  if (!info?.isFile()) {
    problems.push(`${where}: audio_url ${JSON.stringify(episode.audio_url)} does not exist`);
    return;
  }
  episode.audioSource = source;
  episode.audio_url = path.posix.join("media", series.slug, episode.slug, path.basename(clean));
}

function validateChapters(episode, where, problems) {
  if (episode.chapters.length === 0) {
    problems.push(`${where}: chapters is required`);
    return;
  }
  if (episode.chapters.length > MAX_CHAPTERS) {
    problems.push(`${where}: at most ${MAX_CHAPTERS} chapters are allowed`);
  }
  episode.chapters.sort((a, b) => a.startTime - b.startTime);

  episode.chapters.forEach((chapter, index) => {
    const label = `${where}: chapter ${index + 1}`;
    if (!Number.isFinite(chapter.startTime) || chapter.startTime < 0) {
      problems.push(`${label}: startTime must be a positive number`);
    }
    if (index === 0 && chapter.startTime !== 0) {
      problems.push(`${label}: the first chapter must start at 0`);
    }
    if (index > 0 && chapter.startTime === episode.chapters[index - 1].startTime) {
      problems.push(`${label}: startTime duplicates the previous chapter`);
    }
    if (count(chapter.title) < 1 || count(chapter.title) > MAX_CHAPTER_TITLE) {
      problems.push(`${label}: title must be 1..${MAX_CHAPTER_TITLE} characters`);
    }
    if (chapter.endTime != null && chapter.endTime <= chapter.startTime) {
      problems.push(`${label}: endTime must be after startTime`);
    }
    if (episode.duration_seconds > 0 && chapter.startTime >= episode.duration_seconds) {
      problems.push(`${label}: startTime must be before duration_seconds`);
    }
  });
}

function normalizeChapter(raw) {
  const chapter = {
    startTime: Number(raw?.startTime),
    title: String(raw?.title ?? "").trim(),
  };
  if (raw?.endTime != null) chapter.endTime = Number(raw.endTime);
  if (raw?.url) chapter.url = String(raw.url);
  if (raw?.img) chapter.img = String(raw.img);
  if (raw?.toc != null) chapter.toc = Boolean(raw.toc);
  return chapter;
}

/** Writes the generated site data into outDir and copies relative audio. */
export async function buildSite(contentRoot, outDir, options = {}) {
  const { fallback = true } = options;
  const catalog = await loadContent(contentRoot);
  await mkdir(outDir, { recursive: true });

  for (const series of catalog.series) {
    for (const episode of series.episodes) {
      if (!episode.audioSource) continue;
      const destination = path.join(outDir, ...episode.audio_url.split("/"));
      await mkdir(path.dirname(destination), { recursive: true });
      await copyFile(episode.audioSource, destination);
    }
  }

  const index = {
    site: catalog.site,
    series: catalog.series.map((series) => ({
      slug: series.slug,
      title: series.title,
      author: series.author,
      description: series.description,
      accent_color: series.accent_color,
      cover_url: series.cover_url,
      episode_count: series.episodes.length,
      total_duration_seconds: series.episodes.reduce((total, e) => total + e.duration_seconds, 0),
      episodes: series.episodes.map((episode) => ({
        id: episode.id,
        slug: episode.slug,
        series_slug: episode.series_slug,
        number: episode.number,
        title: episode.title,
        description: episode.description,
        published_at: episode.published_at,
        duration_seconds: episode.duration_seconds,
        chapter_count: episode.chapters.length,
        audio_url: episode.audio_url,
      })),
    })),
  };
  await writeJSON(path.join(outDir, "data", "index.json"), index);

  for (const series of catalog.series) {
    for (const { audioSource: _audioSource, ...episode } of series.episodes) {
      await writeJSON(
        path.join(outDir, "data", "episodes", series.slug, `${episode.slug}.json`),
        episode,
      );
    }
  }

  if (fallback) {
    const html = await readFile(path.join(outDir, "index.html"));
    await writeFile(path.join(outDir, "404.html"), html);
  }
  await writeFile(path.join(outDir, ".nojekyll"), "");

  return {
    series: catalog.series.length,
    episodes: catalog.series.reduce((total, series) => total + series.episodes.length, 0),
  };
}

async function readJSON(file) {
  let data;
  try {
    data = await readFile(file, "utf8");
  } catch (error) {
    throw new Error(`read ${file}: ${error.message}`, { cause: error });
  }
  try {
    return JSON.parse(data);
  } catch (error) {
    throw new Error(`parse ${file}: ${error.message}`, { cause: error });
  }
}

async function writeJSON(file, value) {
  await mkdir(path.dirname(file), { recursive: true });
  await writeFile(file, `${JSON.stringify(value, null, 2)}\n`);
}

function validSlug(slug) {
  return slug.length <= MAX_SLUG_LENGTH && SLUG_PATTERN.test(slug);
}

function isAbsoluteUrl(raw) {
  try {
    const parsed = new URL(raw);
    return (parsed.protocol === "http:" || parsed.protocol === "https:") && parsed.host !== "";
  } catch {
    return false;
  }
}

function validDate(raw) {
  if (/^\d{4}-\d{2}-\d{2}$/.test(raw)) return !Number.isNaN(Date.parse(`${raw}T00:00:00Z`));
  return !Number.isNaN(Date.parse(raw));
}

/** Counts characters (code points), matching the previous Go implementation. */
function count(value) {
  return [...String(value)].length;
}

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const PLAYER_DIR = path.resolve(SCRIPT_DIR, "..");
const REPO_DIR = path.resolve(PLAYER_DIR, "..", "..");

async function main(argv) {
  let check = false;
  let fallback = true;
  let contentDir = process.env.CONTENT_DIR ?? path.join(REPO_DIR, "content");
  let outDir = process.env.OUT_DIR ?? path.join(PLAYER_DIR, "dist");

  for (let index = 0; index < argv.length; index += 1) {
    switch (argv[index]) {
      case "--check":
        check = true;
        break;
      case "--no-fallback":
        fallback = false;
        break;
      case "--content":
        contentDir = argv[++index];
        break;
      case "--out":
        outDir = argv[++index];
        break;
      default:
        throw new Error(`unknown argument ${JSON.stringify(argv[index])}`);
    }
  }

  try {
    if (check) {
      const catalog = await loadContent(path.resolve(contentDir));
      const episodes = catalog.series.reduce((total, series) => total + series.episodes.length, 0);
      console.log(`content is valid: ${catalog.series.length} series, ${episodes} episodes`);
      return;
    }
    const result = await buildSite(path.resolve(contentDir), path.resolve(outDir), { fallback });
    console.log(
      `wrote ${result.series} series and ${result.episodes} episodes to ${path.resolve(outDir)}`,
    );
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  await main(process.argv.slice(2));
}
