import type { ReactNode } from "react";

export type IconName =
  | "play"
  | "pause"
  | "rewind"
  | "forward"
  | "previous"
  | "next"
  | "close"
  | "back"
  | "chevron-right"
  | "speed"
  | "moon"
  | "more"
  | "check"
  | "copy"
  | "external"
  | "rss"
  | "list";

const ICONS: Record<IconName, ReactNode> = {
  play: <path d="M8.5 5.8v12.4L19 12z" fill="currentColor" stroke="none" />,
  pause: (
    <>
      <rect x="7" y="5.5" width="3.6" height="13" rx="1.2" fill="currentColor" stroke="none" />
      <rect x="13.4" y="5.5" width="3.6" height="13" rx="1.2" fill="currentColor" stroke="none" />
    </>
  ),
  rewind: (
    <>
      <path d="M11.5 6.2 4.8 12l6.7 5.8z" fill="currentColor" stroke="none" />
      <path d="M19.5 6.2 12.8 12l6.7 5.8z" fill="currentColor" stroke="none" />
    </>
  ),
  forward: (
    <>
      <path d="M12.5 6.2 19.2 12l-6.7 5.8z" fill="currentColor" stroke="none" />
      <path d="M4.5 6.2 11.2 12l-6.7 5.8z" fill="currentColor" stroke="none" />
    </>
  ),
  previous: (
    <>
      <path d="M6 5.5v13" />
      <path d="M19 6.2 9.5 12 19 17.8z" fill="currentColor" stroke="none" />
    </>
  ),
  next: (
    <>
      <path d="M18 5.5v13" />
      <path d="M5 6.2 14.5 12 5 17.8z" fill="currentColor" stroke="none" />
    </>
  ),
  close: <path d="m6 9 6 6 6-6" />,
  back: <path d="m14.5 5.5-6.5 6.5 6.5 6.5" />,
  "chevron-right": <path d="m9.5 5.5 6.5 6.5-6.5 6.5" />,
  speed: (
    <>
      <path d="M4.5 17.5a8 8 0 1 1 15 0" />
      <path d="m12 13.5 4.2-4.2" />
      <circle cx="12" cy="13.5" r="1.4" fill="currentColor" stroke="none" />
    </>
  ),
  moon: <path d="M20 14.3A8.3 8.3 0 0 1 9.7 4a8.3 8.3 0 1 0 10.3 10.3z" />,
  more: (
    <>
      <circle cx="5" cy="12" r="1.5" fill="currentColor" stroke="none" />
      <circle cx="12" cy="12" r="1.5" fill="currentColor" stroke="none" />
      <circle cx="19" cy="12" r="1.5" fill="currentColor" stroke="none" />
    </>
  ),
  check: <path d="m5 12.5 4.5 4.5L19 7" />,
  copy: (
    <>
      <rect x="9" y="9" width="11" height="11" rx="2" />
      <path d="M15 5.5H6.5a2 2 0 0 0-2 2V16" />
    </>
  ),
  external: (
    <>
      <path d="M14 5h5v5" />
      <path d="M19 5l-8 8" />
      <path d="M18 14.5V18a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h3.5" />
    </>
  ),
  rss: (
    <>
      <path d="M5 11a8 8 0 0 1 8 8" />
      <path d="M5 5a14 14 0 0 1 14 14" />
      <circle cx="6" cy="18" r="1.6" fill="currentColor" stroke="none" />
    </>
  ),
  list: (
    <>
      <path d="M9 6h11" />
      <path d="M9 12h11" />
      <path d="M9 18h11" />
      <circle cx="4.5" cy="6" r="1.2" fill="currentColor" stroke="none" />
      <circle cx="4.5" cy="12" r="1.2" fill="currentColor" stroke="none" />
      <circle cx="4.5" cy="18" r="1.2" fill="currentColor" stroke="none" />
    </>
  ),
};

export interface IconProps {
  name: IconName;
  size?: number;
  className?: string;
}

/** Inline stroke icon. Always decorative: the surrounding control is labelled. */
export function Icon({ name, size = 20, className }: IconProps) {
  return (
    <svg
      className={className}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {ICONS[name]}
    </svg>
  );
}
