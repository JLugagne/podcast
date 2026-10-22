import { describe, expect, it } from "vitest";
import {
  chaptersToDocument,
  documentToChapters,
  parseChaptersDocument,
  serializeChapters,
} from "./chapters";
import type { ChapterDTO } from "./types";

function documentWith(chapters: unknown[], version: unknown = "1.2.0"): string {
  return JSON.stringify({ version, chapters });
}

describe("parseChaptersDocument", () => {
  it("accepts a valid document and sorts chapters by start time", () => {
    const result = parseChaptersDocument(
      JSON.stringify({
        version: "1.2.0",
        chapters: [
          { startTime: 95.5, title: "Outro", endTime: 120 },
          { startTime: 0, title: "Intro", img: "https://example.com/cover.png" },
        ],
        podcast: "ignored",
      }),
    );

    expect(result.ok).toBe(true);
    expect(result.errors).toEqual([]);
    expect(result.chapters.map((chapter) => chapter.start_seconds)).toEqual([0, 95.5]);
    expect(result.chapters[0]).toMatchObject({
      title: "Intro",
      toc: true,
      image_url: "https://example.com/cover.png",
    });
    expect(result.chapters[1]).toMatchObject({ title: "Outro", end_seconds: 120 });
    expect(result.document?.chapters.map((chapter) => chapter.startTime)).toEqual([0, 95.5]);
  });

  it.each(["1.0", "1.1.0", "1.2.0"])("accepts chapters version %s", (version) => {
    const result = parseChaptersDocument(documentWith([{ startTime: 0, title: "Intro" }], version));
    expect(result.ok).toBe(true);
  });

  it("rejects a missing version", () => {
    const result = parseChaptersDocument(
      JSON.stringify({ chapters: [{ startTime: 0, title: "Intro" }] }),
    );
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/version/i);
    expect(result.chapters).toEqual([]);
  });

  it.each(["2.0", "1", "0.9", "v1.2.0"])("rejects unsupported version %s", (version) => {
    const result = parseChaptersDocument(documentWith([{ startTime: 0, title: "Intro" }], version));
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/version/i);
  });

  it("rejects invalid JSON", () => {
    const result = parseChaptersDocument("{ not json");
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toMatch(/not valid JSON/i);
  });

  it("rejects a non-object document", () => {
    const result = parseChaptersDocument("[]");
    expect(result.ok).toBe(false);
    expect(result.errors[0]).toMatch(/JSON object/i);
  });

  it("requires at least one chapter", () => {
    const result = parseChaptersDocument(documentWith([]));
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/at least one chapter/i);
  });

  it("rejects more than 500 chapters", () => {
    const chapters = Array.from({ length: 501 }, (_, index) => ({
      startTime: index,
      title: `Chapter ${index + 1}`,
    }));
    const result = parseChaptersDocument(documentWith(chapters));
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/Too many chapters/i);
  });

  it("accepts exactly 500 chapters", () => {
    const chapters = Array.from({ length: 500 }, (_, index) => ({
      startTime: index,
      title: `Chapter ${index + 1}`,
    }));
    const result = parseChaptersDocument(documentWith(chapters));
    expect(result.ok).toBe(true);
    expect(result.chapters).toHaveLength(500);
  });

  it("requires the first chapter to start at 0", () => {
    const result = parseChaptersDocument(
      documentWith([
        { startTime: 10, title: "Intro" },
        { startTime: 20, title: "Body" },
      ]),
    );
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/must start at 0 seconds/i);
  });

  it("rejects duplicate start times", () => {
    const result = parseChaptersDocument(
      documentWith([
        { startTime: 0, title: "Intro" },
        { startTime: 0, title: "Again" },
      ]),
    );
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/unique/i);
  });

  it("requires a title and a numeric startTime", () => {
    const result = parseChaptersDocument(
      documentWith([{ startTime: "0", title: "" }, { startTime: 5 }]),
    );
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/"startTime"/);
    expect(result.errors.join(" ")).toMatch(/"title"/);
  });

  it("rejects titles longer than 300 characters", () => {
    const result = parseChaptersDocument(documentWith([{ startTime: 0, title: "x".repeat(301) }]));
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/at most 300/);
  });

  it("rejects an endTime that is not after startTime", () => {
    const result = parseChaptersDocument(
      documentWith([{ startTime: 10, title: "Intro", endTime: 10 }]),
    );
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/endTime/);
  });

  it("rejects a negative startTime", () => {
    const result = parseChaptersDocument(documentWith([{ startTime: -1, title: "Intro" }]));
    expect(result.ok).toBe(false);
    expect(result.errors.join(" ")).toMatch(/zero or greater/);
  });
});

describe("chapters conversion", () => {
  const chapters: ChapterDTO[] = [
    {
      start_seconds: 30,
      end_seconds: 60,
      title: "Second",
      url: "https://example.com/second",
      image_url: "https://example.com/second.png",
      toc: false,
    },
    { start_seconds: 0, title: "First", toc: true },
  ];

  it("converts DTO chapters to a Podcasting 2.0 document", () => {
    const document = chaptersToDocument(chapters);

    expect(document.version).toBe("1.2.0");
    expect(document.chapters).toEqual([
      { startTime: 0, title: "First", toc: true },
      {
        startTime: 30,
        title: "Second",
        toc: false,
        endTime: 60,
        url: "https://example.com/second",
        img: "https://example.com/second.png",
      },
    ]);
  });

  it("round-trips through serialize and parse", () => {
    const serialized = serializeChapters(chapters);
    expect(serialized.endsWith("\n")).toBe(true);

    const parsed = parseChaptersDocument(serialized);
    expect(parsed.ok).toBe(true);
    expect(parsed.chapters).toEqual([
      { start_seconds: 0, title: "First", toc: true },
      {
        start_seconds: 30,
        end_seconds: 60,
        title: "Second",
        url: "https://example.com/second",
        image_url: "https://example.com/second.png",
        toc: false,
      },
    ]);
  });

  it("converts a document to DTO chapters and defaults toc to true", () => {
    const chaptersFromDocument = documentToChapters({
      version: "1.0",
      chapters: [
        { startTime: 20, title: "Later" },
        { startTime: 0, title: "Earlier", toc: false },
      ],
    });

    expect(chaptersFromDocument).toEqual([
      { start_seconds: 0, title: "Earlier", toc: false },
      { start_seconds: 20, title: "Later", toc: true },
    ]);
  });
});
