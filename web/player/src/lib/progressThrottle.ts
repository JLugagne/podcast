/** How often progress is sent to the API while playback is running. */
export const PROGRESS_SAVE_INTERVAL_MS = 10_000;

/**
 * Decides whether a progress update is due. Forced saves (pause, seek,
 * chapter change) bypass the interval; regular saves are throttled.
 */
export class ProgressThrottle {
  private lastSavedAt: number;

  constructor(
    private readonly intervalMs: number = PROGRESS_SAVE_INTERVAL_MS,
    startAtMs = 0,
  ) {
    this.lastSavedAt = startAtMs;
  }

  shouldSave(nowMs: number): boolean {
    return nowMs - this.lastSavedAt >= this.intervalMs;
  }

  markSaved(nowMs: number): void {
    this.lastSavedAt = nowMs;
  }

  reset(nowMs = 0): void {
    this.lastSavedAt = nowMs;
  }
}
