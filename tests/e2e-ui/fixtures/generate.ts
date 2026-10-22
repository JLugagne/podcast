import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { deflateSync } from "node:zlib";

/** Directory holding the generated binary fixtures. */
export const GENERATED_DIR = path.resolve(__dirname, "generated");

export const COVER_PATH = path.join(GENERATED_DIR, "cover.png");

export const VALID_CHAPTERS_PATH = path.resolve(__dirname, "valid-chapters.json");
export const INVALID_CHAPTERS_PATH = path.resolve(__dirname, "invalid-chapters.json");

/** Duration of the generated tone, in seconds. */
export const AUDIO_DURATION_SECONDS = 20;

const MP3_PATH = path.join(GENERATED_DIR, "tone-20s.mp3");
const WAV_PATH = path.join(GENERATED_DIR, "tone-20s.wav");

export interface AudioFixture {
  path: string;
  name: string;
}

/**
 * Returns the generated audio fixture. ffmpeg produces a real MP3 when it is
 * available; otherwise a valid PCM WAV is written by {@link ensureFixtures}.
 */
export function getAudioFixture(): AudioFixture {
  return audioFixture(MP3_PATH, WAV_PATH, AUDIO_DURATION_SECONDS);
}

/** Duration of the longer tone used by the listener journey, in seconds. */
export const LISTEN_AUDIO_DURATION_SECONDS = 120;

/**
 * Returns a longer tone for the listener journey, where playback has to stay
 * alive across many steps. Generated on demand next to the 20 second fixture.
 */
export function getListenAudioFixture(): AudioFixture {
  return audioFixture(
    path.join(GENERATED_DIR, `tone-${LISTEN_AUDIO_DURATION_SECONDS}s.mp3`),
    path.join(GENERATED_DIR, `tone-${LISTEN_AUDIO_DURATION_SECONDS}s.wav`),
    LISTEN_AUDIO_DURATION_SECONDS,
  );
}

/** Returns the generated fixture, creating it when it is missing. */
function audioFixture(mp3Path: string, wavPath: string, seconds: number): AudioFixture {
  if (!existsSync(mp3Path) && !existsSync(wavPath)) {
    mkdirSync(GENERATED_DIR, { recursive: true });
    generateAudio(mp3Path, wavPath, seconds);
  }
  if (existsSync(mp3Path)) return { path: mp3Path, name: path.basename(mp3Path) };
  return { path: wavPath, name: path.basename(wavPath) };
}

/** Creates the audio and cover fixtures when they are missing. */
export function ensureFixtures(): void {
  mkdirSync(GENERATED_DIR, { recursive: true });
  getAudioFixture();
  if (!existsSync(COVER_PATH)) writePng(COVER_PATH, 512);
}

function hasFfmpeg(): boolean {
  try {
    execFileSync("ffmpeg", ["-version"], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

function generateAudio(mp3Path: string, wavPath: string, seconds: number): void {
  if (!hasFfmpeg()) {
    writeWav(wavPath, seconds);
    return;
  }

  try {
    execFileSync(
      "ffmpeg",
      [
        "-hide_banner",
        "-loglevel",
        "error",
        "-f",
        "lavfi",
        "-i",
        `sine=frequency=440:duration=${seconds}`,
        "-ac",
        "2",
        "-ar",
        "44100",
        "-b:a",
        "96k",
        "-y",
        mp3Path,
      ],
      { stdio: "ignore" },
    );
  } catch {
    writeWav(wavPath, seconds);
  }
}

/** Writes a 16-bit mono PCM WAV containing a 440 Hz sine tone. */
function writeWav(file: string, seconds: number): void {
  const sampleRate = 8000;
  const sampleCount = sampleRate * seconds;
  const bytesPerSample = 2;
  const dataSize = sampleCount * bytesPerSample;
  const buffer = Buffer.alloc(44 + dataSize);

  buffer.write("RIFF", 0, "ascii");
  buffer.writeUInt32LE(36 + dataSize, 4);
  buffer.write("WAVE", 8, "ascii");
  buffer.write("fmt ", 12, "ascii");
  buffer.writeUInt32LE(16, 16);
  buffer.writeUInt16LE(1, 20); // PCM
  buffer.writeUInt16LE(1, 22); // mono
  buffer.writeUInt32LE(sampleRate, 24);
  buffer.writeUInt32LE(sampleRate * bytesPerSample, 28);
  buffer.writeUInt16LE(bytesPerSample, 32);
  buffer.writeUInt16LE(16, 34);
  buffer.write("data", 36, "ascii");
  buffer.writeUInt32LE(dataSize, 40);

  for (let index = 0; index < sampleCount; index += 1) {
    const sample = Math.round(Math.sin((2 * Math.PI * 440 * index) / sampleRate) * 0.3 * 32767);
    buffer.writeInt16LE(sample, 44 + index * bytesPerSample);
  }

  writeFileSync(file, buffer);
}

const CRC_TABLE = (() => {
  const table = new Uint32Array(256);
  for (let n = 0; n < 256; n += 1) {
    let value = n;
    for (let bit = 0; bit < 8; bit += 1) {
      value = value & 1 ? 0xedb88320 ^ (value >>> 1) : value >>> 1;
    }
    table[n] = value >>> 0;
  }
  return table;
})();

function crc32(data: Buffer): number {
  let crc = 0xffffffff;
  for (const byte of data) {
    crc = CRC_TABLE[(crc ^ byte) & 0xff] ^ (crc >>> 8);
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function pngChunk(type: string, data: Buffer): Buffer {
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length, 0);
  const typeAndData = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(typeAndData), 0);
  return Buffer.concat([length, typeAndData, crc]);
}

/** Writes a square RGBA PNG with a warm vertical gradient. */
function writePng(file: string, size: number): void {
  const stride = size * 4 + 1;
  const raw = Buffer.alloc(stride * size);

  for (let y = 0; y < size; y += 1) {
    const row = y * stride;
    raw[row] = 0; // filter: none
    for (let x = 0; x < size; x += 1) {
      const offset = row + 1 + x * 4;
      const blend = y / Math.max(1, size - 1);
      raw[offset] = Math.round(242 - blend * 70); // R
      raw[offset + 1] = Math.round(165 - blend * 60); // G
      raw[offset + 2] = Math.round(65 + blend * 70); // B
      raw[offset + 3] = 255; // A
    }
  }

  const header = Buffer.alloc(13);
  header.writeUInt32BE(size, 0);
  header.writeUInt32BE(size, 4);
  header[8] = 8; // bit depth
  header[9] = 6; // color type: RGBA
  header[10] = 0; // compression
  header[11] = 0; // filter
  header[12] = 0; // interlace

  writeFileSync(
    file,
    Buffer.concat([
      Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
      pngChunk("IHDR", header),
      pngChunk("IDAT", deflateSync(raw)),
      pngChunk("IEND", Buffer.alloc(0)),
    ]),
  );
}
