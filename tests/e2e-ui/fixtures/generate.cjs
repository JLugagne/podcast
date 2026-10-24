const { execFileSync } = require("node:child_process");
const { existsSync, mkdirSync } = require("node:fs");
const path = require("node:path");

/** Duration of the generated tone, in seconds (matches the feed fixture). */
const AUDIO_DURATION_SECONDS = 120;

const MEDIA_DIR = path.resolve(__dirname, "generated");
const AUDIO_PATH = path.join(MEDIA_DIR, "tone-120s.mp3");

/** Generates the audio fixture served by the fake podcast host. */
function ensureFixtures() {
  if (existsSync(AUDIO_PATH)) return;
  mkdirSync(MEDIA_DIR, { recursive: true });
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
        `sine=frequency=440:duration=${AUDIO_DURATION_SECONDS}`,
        "-ac",
        "2",
        "-ar",
        "44100",
        "-b:a",
        "96k",
        "-y",
        AUDIO_PATH,
      ],
      { stdio: "ignore" },
    );
  } catch (error) {
    throw new Error(
      `ffmpeg is required to generate the e2e audio fixture at ${AUDIO_PATH}: ${String(error)}`,
    );
  }
}

module.exports = { AUDIO_DURATION_SECONDS, AUDIO_PATH, ensureFixtures };

if (require.main === module) ensureFixtures();
