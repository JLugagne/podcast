export const PLAYBACK_RATES = [0.8, 1, 1.25, 1.5, 1.75, 2] as const;

export type PlaybackRate = (typeof PLAYBACK_RATES)[number];

export function normalizeRate(value: number): PlaybackRate {
  return PLAYBACK_RATES.includes(value as PlaybackRate) ? (value as PlaybackRate) : 1;
}

/** Formats a rate for display: `0.8×`, `1×`, `1.25×`, `2×`. */
export function formatRate(rate: number): string {
  return `${rate}×`;
}
