// Generates the PWA icons into public/ (plain PNG encoder, no dependencies).
import { mkdir, writeFile } from "node:fs/promises";
import { deflateSync } from "node:zlib";

const BG = [17, 19, 24];
const FG = [122, 156, 255];

function crc32(buffer) {
  let crc = 0xffffffff;
  for (const byte of buffer) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit += 1) crc = (crc >>> 1) ^ (0xedb88320 & -(crc & 1));
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const body = Buffer.concat([Buffer.from(type), data]);
  const out = Buffer.alloc(body.length + 8);
  out.writeUInt32BE(data.length, 0);
  body.copy(out, 4);
  out.writeUInt32BE(crc32(body), body.length + 4);
  return out;
}

/** A dot (the brand mark) with rings, on the app background. `safe` shrinks it for maskable icons. */
function render(size, safe) {
  const rows = [];
  const center = size / 2;
  const radius = size * (safe ? 0.2 : 0.26);
  for (let y = 0; y < size; y += 1) {
    const row = Buffer.alloc(1 + size * 3);
    for (let x = 0; x < size; x += 1) {
      const distance = Math.hypot(x + 0.5 - center, y + 0.5 - center);
      const ring = Math.abs(distance - radius * 1.7) < size * 0.018;
      const on = distance <= radius || ring;
      const color = on ? FG : BG;
      row[1 + x * 3] = color[0];
      row[2 + x * 3] = color[1];
      row[3 + x * 3] = color[2];
    }
    rows.push(row);
  }
  const header = Buffer.alloc(13);
  header.writeUInt32BE(size, 0);
  header.writeUInt32BE(size, 4);
  header.set([8, 2, 0, 0, 0], 8);
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    chunk("IHDR", header),
    chunk("IDAT", deflateSync(Buffer.concat(rows))),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

await mkdir("public", { recursive: true });
await writeFile("public/icon-192.png", render(192, false));
await writeFile("public/icon-512.png", render(512, false));
await writeFile("public/icon-maskable-512.png", render(512, true));
