export type AudioDurationReader = (file: File) => Promise<number>;

/**
 * Default duration reader: loads the file in a detached <audio> element and
 * reads its metadata before the upload starts.
 */
export const domAudioDurationReader: AudioDurationReader = (file) =>
  new Promise<number>((resolve) => {
    const objectUrl = URL.createObjectURL(file);
    const audio = document.createElement("audio");
    let settled = false;

    const finish = (duration: number) => {
      if (settled) return;
      settled = true;
      window.clearTimeout(timer);
      audio.removeAttribute("src");
      audio.load();
      URL.revokeObjectURL(objectUrl);
      resolve(Number.isFinite(duration) && duration > 0 ? duration : 0);
    };

    const timer = window.setTimeout(() => finish(0), 15000);
    audio.preload = "metadata";
    audio.onloadedmetadata = () => finish(audio.duration);
    audio.onerror = () => finish(0);
    audio.src = objectUrl;
  });

let activeDurationReader: AudioDurationReader = domAudioDurationReader;

/** Replaces duration detection, or restores the DOM reader with `null`. */
export function setAudioDurationReader(reader: AudioDurationReader | null): void {
  activeDurationReader = reader ?? domAudioDurationReader;
}

/** Reads the duration of an audio file in the browser without uploading it. */
export function readAudioDuration(file: File): Promise<number> {
  return activeDurationReader(file);
}

/** True when the browser reports an audio MIME type for the file. */
export function isAudioFile(file: File): boolean {
  return file.type.startsWith("audio/");
}

const ALLOWED_COVER_TYPES = new Set(["image/jpeg", "image/png", "image/webp"]);

/** Mirrors the cover content types accepted by the API. */
export function isCoverImage(file: File): boolean {
  return ALLOWED_COVER_TYPES.has(file.type);
}
