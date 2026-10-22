function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

/** Formats a position, duration or chapter length as `m:ss`, or `h:mm:ss` past an hour. */
export function formatDuration(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined) return "—";
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  const total = Math.floor(seconds);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const remainder = total % 60;
  if (hours > 0) return `${hours}:${pad2(minutes)}:${pad2(remainder)}`;
  return `${minutes}:${pad2(remainder)}`;
}

/** Human friendly remaining time, suitable for an episode's state line. */
export function formatMinutesLeft(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined) return "—";
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  const minutes = Math.ceil(seconds / 60);
  if (minutes <= 0) return "0 min left";
  return `${minutes} min left`;
}

/** Returns up to two initials from a title, for artwork placeholders. */
export function initials(title: string): string {
  const words = title.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return "?";
  const first = words[0][0] ?? "";
  const second = words.length > 1 ? (words[words.length - 1][0] ?? "") : "";
  return `${first}${second}`.toUpperCase();
}

/** Formats a count with a singular / plural noun: `1 episode`, `2 episodes`. */
export function pluralize(count: number, singular: string, plural = `${singular}s`): string {
  return `${count} ${count === 1 ? singular : plural}`;
}
