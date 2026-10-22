import { useState } from "react";
import { initials } from "../lib/format";

export interface CoverTileProps {
  title: string;
  accentColor: string;
  src?: string | null;
  size?: "sm" | "md" | "lg";
}

/**
 * Square cover artwork, falling back to a colored tile with the title initials.
 * Decorative: the adjacent heading always names the series.
 */
export function CoverTile({ title, accentColor, src, size = "md" }: CoverTileProps) {
  const [failedSrc, setFailedSrc] = useState<string | null>(null);
  const showImage = Boolean(src) && failedSrc !== src;
  const classes = [
    "cover-tile",
    size === "sm" ? "cover-tile-sm" : "",
    size === "lg" ? "cover-tile-lg" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes} style={{ backgroundColor: accentColor }} aria-hidden="true">
      {showImage ? (
        <img src={src ?? undefined} alt="" onError={() => setFailedSrc(src ?? null)} />
      ) : (
        <span>{initials(title)}</span>
      )}
    </div>
  );
}
