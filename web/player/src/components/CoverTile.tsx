import { useState } from "react";
import { contrastText } from "../lib/color";
import { initials } from "../lib/format";

export interface CoverTileProps {
  title: string;
  accentColor: string;
  src?: string | null;
  size?: "sm" | "md" | "lg" | "xl";
}

/**
 * Square cover artwork, falling back to an accent-colored tile with the title
 * initials. Decorative: adjacent text always names the series or episode.
 */
export function CoverTile({ title, accentColor, src, size = "md" }: CoverTileProps) {
  const [failedSrc, setFailedSrc] = useState<string | null>(null);
  const showImage = Boolean(src) && failedSrc !== src;
  const classes = [
    "cover-tile",
    size === "sm" ? "cover-tile-sm" : "",
    size === "lg" ? "cover-tile-lg" : "",
    size === "xl" ? "cover-tile-xl" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      className={classes}
      style={{ backgroundColor: accentColor, color: contrastText(accentColor) }}
      aria-hidden="true"
    >
      {showImage ? (
        <img src={src ?? undefined} alt="" onError={() => setFailedSrc(src ?? null)} />
      ) : (
        <span>{initials(title)}</span>
      )}
    </div>
  );
}
