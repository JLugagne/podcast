function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

/** Formats a position or length as `h:mm:ss` (hours are always shown). */
export function formatTimecode(seconds: number): string {
  if (!Number.isFinite(seconds)) return "—";
  const total = Math.max(0, Math.floor(seconds));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const remainder = total % 60;
  return `${hours}:${pad2(minutes)}:${pad2(remainder)}`;
}

/** Formats a duration as `m:ss`, or `h:mm:ss` once it reaches an hour. */
export function formatDuration(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined) return "—";
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  if (seconds >= 3600) return formatTimecode(seconds);
  const total = Math.floor(seconds);
  return `${Math.floor(total / 60)}:${pad2(total % 60)}`;
}

/** Formats a byte count using binary units (1 KB = 1024 B). */
export function formatBytes(bytes: number | null | undefined): string {
  if (bytes === null || bytes === undefined) return "—";
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  if (bytes < 1024) return `${Math.floor(bytes)} B`;

  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const rounded = Math.round(value * 10) / 10;
  return `${rounded} ${units[unit]}`;
}

/** Formats an RFC3339 timestamp for display. */
export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

/** Returns up to two initials from a title, for artwork placeholders. */
export function initials(title: string): string {
  const words = title.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return "?";
  const first = words[0][0] ?? "";
  const second = words.length > 1 ? (words[words.length - 1][0] ?? "") : "";
  return `${first}${second}`.toUpperCase();
}
