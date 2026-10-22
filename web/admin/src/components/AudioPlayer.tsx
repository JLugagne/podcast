import type { RefObject } from "react";

export interface AudioPlayerProps {
  src: string;
  audioRef?: RefObject<HTMLAudioElement | null>;
  label?: string;
}

export function AudioPlayer({ src, audioRef, label = "Episode audio" }: AudioPlayerProps) {
  return (
    // biome-ignore lint/a11y/useMediaCaption: podcast audio files carry no caption track; the chapter list is the textual outline
    <audio
      ref={audioRef}
      className="audio-player"
      controls
      preload="metadata"
      src={src}
      aria-label={label}
    >
      Your browser does not support audio playback.
    </audio>
  );
}
