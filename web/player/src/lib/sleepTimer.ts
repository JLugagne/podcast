export type SleepTimerOption = "off" | "15" | "30" | "45" | "60" | "chapter" | "episode";

export type SleepTimer =
  | { kind: "off" }
  | { kind: "minutes"; minutes: number; endsAt: number }
  | { kind: "chapter"; chapterIndex: number }
  | { kind: "episode" };

export interface SleepTimerOptionDescriptor {
  value: SleepTimerOption;
  label: string;
}

export const SLEEP_TIMER_OPTIONS: SleepTimerOptionDescriptor[] = [
  { value: "off", label: "Off" },
  { value: "15", label: "15 minutes" },
  { value: "30", label: "30 minutes" },
  { value: "45", label: "45 minutes" },
  { value: "60", label: "60 minutes" },
  { value: "chapter", label: "End of chapter" },
  { value: "episode", label: "End of episode" },
];

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
      return `Sleep timer ${formatCountdown(remainingSeconds ?? timer.minutes * 60)}`;
    case "chapter":
      return "Sleep timer end of chapter";
    case "episode":
      return "Sleep timer end of episode";
    default:
      return "Sleep timer off";
  }
}
