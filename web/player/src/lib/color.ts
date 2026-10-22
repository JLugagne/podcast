/**
 * Text color that stays readable on a series accent color. Falls back to the
 * dark ink used elsewhere when the hex cannot be parsed.
 */
export function contrastText(hex: string): string {
  const match = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
  if (!match) return "#111318";
  const value = Number.parseInt(match[1], 16);
  const red = (value >> 16) & 0xff;
  const green = (value >> 8) & 0xff;
  const blue = value & 0xff;
  const luminance = (0.2126 * red + 0.7152 * green + 0.0722 * blue) / 255;
  return luminance > 0.55 ? "#111318" : "#F4F1EA";
}
