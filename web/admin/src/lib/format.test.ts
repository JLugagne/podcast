import { describe, expect, it } from "vitest";
import { formatBytes, formatDuration, formatTimecode, initials } from "./format";

describe("formatTimecode", () => {
  it("formats positions as h:mm:ss", () => {
    expect(formatTimecode(0)).toBe("0:00:00");
    expect(formatTimecode(5)).toBe("0:00:05");
    expect(formatTimecode(65)).toBe("0:01:05");
    expect(formatTimecode(95.5)).toBe("0:01:35");
    expect(formatTimecode(3661)).toBe("1:01:01");
    expect(formatTimecode(25 * 3600 + 61)).toBe("25:01:01");
  });

  it("clamps negatives and handles invalid input", () => {
    expect(formatTimecode(-3)).toBe("0:00:00");
    expect(formatTimecode(Number.NaN)).toBe("—");
  });
});

describe("formatDuration", () => {
  it("uses m:ss below an hour and h:mm:ss above", () => {
    expect(formatDuration(0)).toBe("0:00");
    expect(formatDuration(9)).toBe("0:09");
    expect(formatDuration(95)).toBe("1:35");
    expect(formatDuration(3599)).toBe("59:59");
    expect(formatDuration(3600)).toBe("1:00:00");
    expect(formatDuration(5432.9)).toBe("1:30:32");
  });

  it("returns a dash for missing or invalid values", () => {
    expect(formatDuration(null)).toBe("—");
    expect(formatDuration(undefined)).toBe("—");
    expect(formatDuration(-1)).toBe("—");
    expect(formatDuration(Number.POSITIVE_INFINITY)).toBe("—");
  });
});

describe("formatBytes", () => {
  it("formats byte counts with binary units", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(1024)).toBe("1 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
    expect(formatBytes(1024 * 1024)).toBe("1 MB");
    expect(formatBytes(2.5 * 1024 * 1024)).toBe("2.5 MB");
    expect(formatBytes(1024 * 1024 * 1024)).toBe("1 GB");
  });

  it("returns a dash for invalid values", () => {
    expect(formatBytes(null)).toBe("—");
    expect(formatBytes(-5)).toBe("—");
    expect(formatBytes(Number.NaN)).toBe("—");
  });
});

describe("initials", () => {
  it("takes the first and last word initials", () => {
    expect(initials("My Great Podcast")).toBe("MP");
    expect(initials("Solo")).toBe("S");
    expect(initials("  spaced   out  ")).toBe("SO");
  });

  it("falls back for empty titles", () => {
    expect(initials("")).toBe("?");
    expect(initials("   ")).toBe("?");
  });
});
