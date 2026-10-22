/** Appends a version query parameter so replaced assets bypass the browser cache. */
export function withCacheBust(url: string | null | undefined, version: number): string | null {
  if (!url) return null;
  const separator = url.includes("?") ? "&" : "?";
  return `${url}${separator}v=${version}`;
}
