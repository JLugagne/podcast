import type { EpisodeStatus } from "../api/types";

export function StatusBadge({ status }: { status: EpisodeStatus }) {
  const published = status === "published";
  return (
    <span className={published ? "badge badge-published" : "badge badge-draft"}>
      {published ? "Published" : "Draft"}
    </span>
  );
}
