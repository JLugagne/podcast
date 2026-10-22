import type { ChapterDTO } from "./types";

/** Version written when serializing chapters for the API. */
export const CHAPTERS_VERSION = "1.2.0";
export const MAX_CHAPTERS = 500;
export const MAX_CHAPTER_TITLE_LENGTH = 300;

/** Podcasting 2.0 accepts any 1.x version. */
const SUPPORTED_VERSION = /^1\.\d+(?:\.\d+)*$/;

export interface ChaptersDocumentChapter {
  startTime: number;
  endTime?: number;
  title: string;
  img?: string;
  url?: string;
  toc?: boolean;
}

export interface ChaptersDocument {
  version: string;
  chapters: ChaptersDocumentChapter[];
}

export interface ChaptersParseResult {
  ok: boolean;
  errors: string[];
  /** Normalized chapters (sorted by start time). Empty when validation failed. */
  chapters: ChapterDTO[];
  /** Normalized document; null when validation failed. */
  document: ChaptersDocument | null;
}

interface DraftChapter {
  startTime: number;
  endTime?: number;
  title: string;
  img?: string;
  url?: string;
  toc: boolean;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function draftToDTO(chapter: DraftChapter): ChapterDTO {
  return {
    start_seconds: chapter.startTime,
    ...(chapter.endTime !== undefined ? { end_seconds: chapter.endTime } : {}),
    title: chapter.title,
    ...(chapter.url !== undefined ? { url: chapter.url } : {}),
    ...(chapter.img !== undefined ? { image_url: chapter.img } : {}),
    toc: chapter.toc,
  };
}

function draftToDocument(chapter: DraftChapter): ChaptersDocumentChapter {
  return {
    startTime: chapter.startTime,
    title: chapter.title,
    toc: chapter.toc,
    ...(chapter.endTime !== undefined ? { endTime: chapter.endTime } : {}),
    ...(chapter.url !== undefined ? { url: chapter.url } : {}),
    ...(chapter.img !== undefined ? { img: chapter.img } : {}),
  };
}

function failure(errors: string[]): ChaptersParseResult {
  return { ok: false, errors, chapters: [], document: null };
}

/**
 * Parses and validates a Podcasting 2.0 JSON chapters document client-side,
 * mirroring the server rules:
 *
 * - `version` is required and must be `1.x`;
 * - `startTime` (number >= 0) and `title` (1..300 chars) are required;
 * - after sorting, start times must be strictly increasing (no duplicates);
 * - the first chapter must start at 0;
 * - `endTime`, when present, must be greater than `startTime`;
 * - between 1 and 500 chapters.
 *
 * Unknown fields are ignored. Input does not need to be pre-sorted.
 */
export function parseChaptersDocument(text: string): ChaptersParseResult {
  const errors: string[] = [];

  if (text.trim().length === 0) {
    return failure(["Paste a chapters JSON document or choose a .json file."]);
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch (error) {
    const detail = error instanceof Error ? error.message : "invalid JSON";
    return failure([`The document is not valid JSON: ${detail}`]);
  }

  if (!isRecord(parsed)) {
    return failure(["The chapters document must be a JSON object."]);
  }

  const version = parsed.version;
  if (typeof version !== "string" || version.trim().length === 0) {
    errors.push('The document is missing the required "version" field.');
  } else if (!SUPPORTED_VERSION.test(version.trim())) {
    errors.push(
      `Unsupported chapters version "${version}"; only 1.x documents are accepted (for example "1.2.0").`,
    );
  }

  const rawChapters = parsed.chapters;
  if (!Array.isArray(rawChapters)) {
    errors.push('The document is missing the required "chapters" array.');
    return failure(errors);
  }

  if (rawChapters.length === 0) {
    errors.push("At least one chapter is required.");
  }
  if (rawChapters.length > MAX_CHAPTERS) {
    errors.push(`Too many chapters: ${rawChapters.length} (the maximum is ${MAX_CHAPTERS}).`);
    return failure(errors);
  }

  const drafts: DraftChapter[] = [];

  rawChapters.forEach((raw, index) => {
    const position = index + 1;
    if (!isRecord(raw)) {
      errors.push(`Chapter ${position}: expected a JSON object.`);
      return;
    }

    const errorsBefore = errors.length;

    const startTime = raw.startTime;
    if (typeof startTime !== "number" || !Number.isFinite(startTime)) {
      errors.push(`Chapter ${position}: "startTime" is required and must be a number.`);
    } else if (startTime < 0) {
      errors.push(`Chapter ${position}: "startTime" must be zero or greater.`);
    }

    const rawTitle = raw.title;
    let title = "";
    if (typeof rawTitle !== "string" || rawTitle.trim().length === 0) {
      errors.push(`Chapter ${position}: "title" is required.`);
    } else {
      title = rawTitle.trim();
      if (title.length > MAX_CHAPTER_TITLE_LENGTH) {
        errors.push(
          `Chapter ${position}: "title" must be at most ${MAX_CHAPTER_TITLE_LENGTH} characters.`,
        );
      }
    }

    let endTime: number | undefined;
    if (raw.endTime !== undefined && raw.endTime !== null) {
      if (typeof raw.endTime !== "number" || !Number.isFinite(raw.endTime)) {
        errors.push(`Chapter ${position}: "endTime" must be a number.`);
      } else {
        endTime = raw.endTime;
        if (typeof startTime === "number" && Number.isFinite(startTime) && endTime <= startTime) {
          errors.push(
            `Chapter ${position}: "endTime" (${endTime}s) must be greater than "startTime" (${startTime}s).`,
          );
        }
      }
    }

    let url: string | undefined;
    if (raw.url !== undefined && raw.url !== null) {
      if (typeof raw.url !== "string") {
        errors.push(`Chapter ${position}: "url" must be a string.`);
      } else if (raw.url.trim().length > 0) {
        url = raw.url.trim();
      }
    }

    let img: string | undefined;
    if (raw.img !== undefined && raw.img !== null) {
      if (typeof raw.img !== "string") {
        errors.push(`Chapter ${position}: "img" must be a string.`);
      } else if (raw.img.trim().length > 0) {
        img = raw.img.trim();
      }
    }

    let toc = true;
    if (raw.toc !== undefined && raw.toc !== null) {
      if (typeof raw.toc !== "boolean") {
        errors.push(`Chapter ${position}: "toc" must be true or false.`);
      } else {
        toc = raw.toc;
      }
    }

    if (errors.length !== errorsBefore) return;
    if (typeof startTime !== "number" || !Number.isFinite(startTime)) return;

    drafts.push({
      startTime,
      title,
      toc,
      ...(endTime !== undefined ? { endTime } : {}),
      ...(url ? { url } : {}),
      ...(img ? { img } : {}),
    });
  });

  if (errors.length > 0) return failure(errors);

  const sorted = [...drafts].sort((a, b) => a.startTime - b.startTime);

  const reportedDuplicates = new Set<number>();
  for (let index = 1; index < sorted.length; index += 1) {
    const start = sorted[index].startTime;
    if (start === sorted[index - 1].startTime && !reportedDuplicates.has(start)) {
      reportedDuplicates.add(start);
      errors.push(
        `Two chapters start at ${start}s; start times must be unique and strictly increasing.`,
      );
    }
  }

  if (sorted.length > 0 && sorted[0].startTime !== 0) {
    errors.push(
      `The first chapter must start at 0 seconds (the earliest starts at ${sorted[0].startTime}s).`,
    );
  }

  if (errors.length > 0) return failure(errors);

  const document: ChaptersDocument = {
    version: typeof parsed.version === "string" ? parsed.version.trim() : CHAPTERS_VERSION,
    chapters: sorted.map(draftToDocument),
  };

  return {
    ok: true,
    errors: [],
    chapters: sorted.map(draftToDTO),
    document,
  };
}

/** Converts episode chapters (API DTO) into a Podcasting 2.0 document. */
export function chaptersToDocument(chapters: ChapterDTO[]): ChaptersDocument {
  const sorted = [...chapters].sort((a, b) => a.start_seconds - b.start_seconds);
  return {
    version: CHAPTERS_VERSION,
    chapters: sorted.map((chapter) => ({
      startTime: chapter.start_seconds,
      title: chapter.title,
      toc: chapter.toc,
      ...(chapter.end_seconds !== undefined && chapter.end_seconds !== null
        ? { endTime: chapter.end_seconds }
        : {}),
      ...(chapter.url ? { url: chapter.url } : {}),
      ...(chapter.image_url ? { img: chapter.image_url } : {}),
    })),
  };
}

/** Serializes episode chapters as a pretty-printed Podcasting 2.0 document. */
export function serializeChapters(chapters: ChapterDTO[]): string {
  return `${JSON.stringify(chaptersToDocument(chapters), null, 2)}\n`;
}

/** Converts a Podcasting 2.0 document into episode chapter DTOs (already parsed). */
export function documentToChapters(document: ChaptersDocument): ChapterDTO[] {
  return [...document.chapters]
    .sort((a, b) => a.startTime - b.startTime)
    .map((chapter) => ({
      start_seconds: chapter.startTime,
      title: chapter.title,
      toc: chapter.toc ?? true,
      ...(chapter.endTime !== undefined ? { end_seconds: chapter.endTime } : {}),
      ...(chapter.url ? { url: chapter.url } : {}),
      ...(chapter.img ? { image_url: chapter.img } : {}),
    }));
}
