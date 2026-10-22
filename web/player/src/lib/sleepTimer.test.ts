import { describe, expect, it } from "vitest";
import {
  formatCountdown,
  isSleepTimerExpired,
  type SleepTimer,
  shouldPauseForChapterTimer,
  shouldStopAtEpisodeEnd,
  sleepTimerFromOption,
  sleepTimerLabel,
  sleepTimerRemainingSeconds,
} from "./sleepTimer";

const NOW = 1_000_000;

describe("sleepTimerFromOption", () => {
  it("builds the off timer", () => {
    expect(sleepTimerFromOption("off", { nowMs: NOW, chapterIndex: 0 })).toEqual({ kind: "off" });
  });

  it("builds a minute countdown relative to now", () => {
    expect(sleepTimerFromOption("15", { nowMs: NOW, chapterIndex: 0 })).toEqual({
      kind: "minutes",
      minutes: 15,
      endsAt: NOW + 15 * 60_000,
    });
    expect(sleepTimerFromOption("60", { nowMs: NOW, chapterIndex: 0 })).toEqual({
      kind: "minutes",
      minutes: 60,
      endsAt: NOW + 60 * 60_000,
    });
  });

  it("remembers the chapter that must finish", () => {
    expect(sleepTimerFromOption("chapter", { nowMs: NOW, chapterIndex: 2 })).toEqual({
      kind: "chapter",
      chapterIndex: 2,
    });
    expect(sleepTimerFromOption("chapter", { nowMs: NOW, chapterIndex: -1 })).toEqual({
      kind: "chapter",
      chapterIndex: 0,
    });
  });

  it("builds the end-of-episode timer", () => {
    expect(sleepTimerFromOption("episode", { nowMs: NOW, chapterIndex: 0 })).toEqual({
      kind: "episode",
    });
  });
});

describe("isSleepTimerExpired", () => {
  const timer: SleepTimer = { kind: "minutes", minutes: 15, endsAt: NOW + 900_000 };

  it("expires at the end of the countdown", () => {
    expect(isSleepTimerExpired(timer, NOW + 899_999)).toBe(false);
    expect(isSleepTimerExpired(timer, NOW + 900_000)).toBe(true);
    expect(isSleepTimerExpired(timer, NOW + 900_001)).toBe(true);
  });

  it("never expires the other kinds", () => {
    expect(isSleepTimerExpired({ kind: "off" }, NOW)).toBe(false);
    expect(isSleepTimerExpired({ kind: "chapter", chapterIndex: 0 }, NOW)).toBe(false);
    expect(isSleepTimerExpired({ kind: "episode" }, NOW)).toBe(false);
  });
});

describe("sleepTimerRemainingSeconds", () => {
  it("rounds the remaining time up to whole seconds", () => {
    const timer: SleepTimer = { kind: "minutes", minutes: 15, endsAt: NOW + 899_200 };
    expect(sleepTimerRemainingSeconds(timer, NOW)).toBe(900);
    expect(sleepTimerRemainingSeconds(timer, NOW + 898_000)).toBe(2);
    expect(sleepTimerRemainingSeconds(timer, NOW + 999_000)).toBe(0);
  });

  it("returns null when there is no countdown", () => {
    expect(sleepTimerRemainingSeconds({ kind: "off" }, NOW)).toBeNull();
    expect(sleepTimerRemainingSeconds({ kind: "chapter", chapterIndex: 0 }, NOW)).toBeNull();
  });
});

describe("formatCountdown", () => {
  it("formats minutes and seconds, adding hours when needed", () => {
    expect(formatCountdown(900)).toBe("15:00");
    expect(formatCountdown(59)).toBe("0:59");
    expect(formatCountdown(1)).toBe("0:01");
    expect(formatCountdown(3661)).toBe("1:01:01");
    expect(formatCountdown(-5)).toBe("0:00");
  });
});

describe("shouldPauseForChapterTimer", () => {
  const timer: SleepTimer = { kind: "chapter", chapterIndex: 1 };

  it("pauses at the end of the chosen chapter", () => {
    expect(shouldPauseForChapterTimer(timer, 29, 30)).toBe(false);
    expect(shouldPauseForChapterTimer(timer, 30, 30)).toBe(true);
    expect(shouldPauseForChapterTimer(timer, 31, 30)).toBe(true);
  });

  it("ignores other timer kinds", () => {
    expect(shouldPauseForChapterTimer({ kind: "off" }, 100, 30)).toBe(false);
    expect(shouldPauseForChapterTimer({ kind: "minutes", minutes: 15, endsAt: NOW }, 100, 30)).toBe(
      false,
    );
  });
});

describe("shouldStopAtEpisodeEnd", () => {
  it("suppresses auto-play for every active timer", () => {
    expect(shouldStopAtEpisodeEnd({ kind: "off" })).toBe(false);
    expect(shouldStopAtEpisodeEnd({ kind: "episode" })).toBe(true);
    expect(shouldStopAtEpisodeEnd({ kind: "chapter", chapterIndex: 0 })).toBe(true);
    expect(shouldStopAtEpisodeEnd({ kind: "minutes", minutes: 8, endsAt: NOW })).toBe(true);
  });
});

describe("sleepTimerLabel", () => {
  it("describes the active timer", () => {
    expect(sleepTimerLabel({ kind: "minutes", minutes: 15, endsAt: NOW }, 899)).toBe(
      "Sleep timer 14:59",
    );
    expect(sleepTimerLabel({ kind: "chapter", chapterIndex: 0 }, null)).toBe(
      "Sleep timer end of chapter",
    );
    expect(sleepTimerLabel({ kind: "episode" }, null)).toBe("Sleep timer end of episode");
    expect(sleepTimerLabel({ kind: "off" }, null)).toBe("Sleep timer off");
  });
});
