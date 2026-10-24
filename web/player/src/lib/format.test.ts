import { describe, expect, it } from "vitest";
import { formatDuration, formatMinutesLeft, initials } from "./format";

describe("formatDuration", () => {
  it("formats positions under an hour as m:ss", () => {
    expect(formatDuration(0)).toBe("0:00");
    expect(formatDuration(5)).toBe("0:05");
    expect(formatDuration(65)).toBe("1:05");
    expect(formatDuration(599.9)).toBe("9:59");
  });

  it("adds hours once the duration reaches one", () => {
    expect(formatDuration(3600)).toBe("1:00:00");
    expect(formatDuration(3725)).toBe("1:02:05");
  });

  it("returns a dash for unusable values", () => {
    expect(formatDuration(null)).toBe("—");
    expect(formatDuration(undefined)).toBe("—");
    expect(formatDuration(-1)).toBe("—");
    expect(formatDuration(Number.NaN)).toBe("—");
  });
});

describe("formatMinutesLeft", () => {
  it("rounds up to whole minutes", () => {
    expect(formatMinutesLeft(0)).toBe("0 min left");
    expect(formatMinutesLeft(1)).toBe("1 min left");
    expect(formatMinutesLeft(61)).toBe("2 min left");
    expect(formatMinutesLeft(600)).toBe("10 min left");
  });

  it("rejects negative or unusable input", () => {
    expect(formatMinutesLeft(-4)).toBe("—");
    expect(formatMinutesLeft(null)).toBe("—");
  });
});

describe("initials", () => {
  it("uses the first letters of the first and last words", () => {
    expect(initials("Deep Dive")).toBe("DD");
    expect(initials("the long form podcast")).toBe("TP");
    expect(initials("Solo")).toBe("S");
    expect(initials("   ")).toBe("?");
  });
});
