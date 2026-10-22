const SKELETON_KEYS = ["skeleton-1", "skeleton-2", "skeleton-3", "skeleton-4", "skeleton-5"];

export function LoadingBlock({ label = "Loading…" }: { label?: string }) {
  return (
    <div className="loading-block" role="status">
      <span className="spinner" aria-hidden="true" />
      <span>{label}</span>
    </div>
  );
}

export function SkeletonList({ count = 3 }: { count?: number }) {
  return (
    <div className="skeleton-list" aria-hidden="true">
      {SKELETON_KEYS.slice(0, Math.min(count, SKELETON_KEYS.length)).map((key) => (
        <div className="skeleton-row" key={key} />
      ))}
    </div>
  );
}
