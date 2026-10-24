import { t } from "../i18n";

export type SleepTimerOption = "off" | "15" | "30" | "45" | "60" | "chapter" | "episode";

export type SleepTimer =
  | { kind: "off" }
  | { kind: "minutes"; minutes: number; endsAt: number }
  | { kind: "chapter"; chapterIndex: number }
  | { kind: "episode" };

export const SLEEP_TIMER_OPTIONS: SleepTimerOption[] = [
  "off",
  "15",
  "30",
  "45",
  "60",
  "chapter",
  "episode",
];

/** Label of one choice in the sleep timer menu. */
export function sleepTimerOptionLabel(option: SleepTimerOption): string {
  switch (option) {
    case "off":
      return t("sleepOff");
    case "15":
      return t("minutes15");
    case "30":
      return t("minutes30");
    case "45":
      return t("minutes45");
    case "60":
      return t("minutes60");
    case "chapter":
      return t("endOfChapter");
    case "episode":
      return t("endOfEpisode");
  }
}

export const MINUTE_OPTIONS = [15, 30, 45, 60] as const;

export interface SleepTimerContext {
  nowMs: number;
  chapterIndex: number;
}

/** Builds a timer from a menu choice; minutes are relative to `nowMs`. */
export function sleepTimerFromOption(
  option: SleepTimerOption,
  context: SleepTimerContext,
): SleepTimer {
  if (option === "off") return { kind: "off" };
  if (option === "chapter") {
    return { kind: "chapter", chapterIndex: Math.max(0, context.chapterIndex) };
  }
  if (option === "episode") return { kind: "episode" };
  const minutes = Number(option);
  return { kind: "minutes", minutes, endsAt: context.nowMs + minutes * 60_000 };
}

export function isSleepTimerExpired(timer: SleepTimer, nowMs: number): boolean {
  return timer.kind === "minutes" && nowMs >= timer.endsAt;
}

/** Seconds remaining for a countdown timer, or null for the other kinds. */
export function sleepTimerRemainingSeconds(timer: SleepTimer, nowMs: number): number | null {
  if (timer.kind !== "minutes") return null;
  return Math.max(0, Math.ceil((timer.endsAt - nowMs) / 1000));
}

/** Formats a countdown as `m:ss`, or `h:mm:ss` past an hour. */
export function formatCountdown(seconds: number): string {
  const total = Math.max(0, Math.ceil(seconds));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const remainder = total % 60;
  const paddedSeconds = String(remainder).padStart(2, "0");
  if (hours > 0) return `${hours}:${String(minutes).padStart(2, "0")}:${paddedSeconds}`;
  return `${minutes}:${paddedSeconds}`;
}

/** True when the chapter chosen for an "end of chapter" timer has finished. */
export function shouldPauseForChapterTimer(
  timer: SleepTimer,
  positionSeconds: number,
  chapterEndSeconds: number,
): boolean {
  if (timer.kind !== "chapter") return false;
  return positionSeconds + 0.5 >= chapterEndSeconds;
}

/** Any active timer suppresses auto-play at the end of an episode. */
export function shouldStopAtEpisodeEnd(timer: SleepTimer): boolean {
  return timer.kind !== "off";
}

/** Accessible label for the sleep timer trigger. */
export function sleepTimerLabel(timer: SleepTimer, remainingSeconds: number | null): string {
  switch (timer.kind) {
    case "minutes":
      return t("sleepTimer", {
        time: formatCountdown(remainingSeconds ?? timer.minutes * 60),
      });
    case "chapter":
      return t("sleepTimerChapter");
    case "episode":
      return t("sleepTimerEpisode");
    default:
      return t("sleepTimerOff");
  }
}
